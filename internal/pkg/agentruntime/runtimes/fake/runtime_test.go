//go:build e2e

package fake

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentre-ai/agentre/internal/model/entity/agent_backend_entity"
	"github.com/agentre-ai/agentre/internal/pkg/agentruntime"
	"github.com/agentre-ai/agentre/internal/pkg/agentruntime/capability"
)

func TestRun_EchoesPromptThenDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r := New()
	events, result, err := r.Run(ctx, agentruntime.RunRequest{
		Backend:   &agent_backend_entity.AgentBackend{ID: 1, Type: string(agent_backend_entity.TypeClaudeCode)},
		SessionID: 42,
		UserText:  "ping",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	var text string
	var sawDone bool
	for ev := range events {
		switch e := ev.(type) {
		case agentruntime.TextDelta:
			text += e.Text
		case agentruntime.Done:
			sawDone = true
		}
	}

	assert.Equal(t, ReplyPrefix+"ping", text)
	assert.True(t, sawDone)
	assert.Equal(t, "e2e-fake-42", result.ProviderSessionID)
	assert.Equal(t, "e2e-fake-model", result.Model)
}

func TestRun_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before draining

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{SessionID: 7, UserText: "hello world this is a long enough prompt to span several chunks"})
	require.NoError(t, err)

	// Draining a pre-cancelled run must terminate (channel closes) without hanging.
	for range events { //nolint:revive // draining
	}
}

func TestRun_HonorsChunkDelayEnv(t *testing.T) {
	t.Setenv("AGENTRE_E2E_FAKE_CHUNK_DELAY_MS", "25")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID: 7,
		UserText:  "hello world this is long enough to span chunks",
	})
	require.NoError(t, err)

	first := <-events
	_, ok := first.(agentruntime.TextDelta)
	require.True(t, ok)
	start := time.Now()

	second := <-events
	_, ok = second.(agentruntime.TextDelta)
	require.True(t, ok)
	assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
}

// 群成员 turn:fake 像真 CLI 一样,把本轮回复经注入的 group MCP server 调 group_send 冒泡进群。
// 断言它确实对 spec.URL 发了一次 tools/call(group_send),带上注入的 Authorization,
// body=回显文本,mentions=["用户"](回人类来源,使本轮 ingest 后无 agent 收件人而自然收敛)。
func TestRun_PostsGroupSendWhenInjected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type captured struct {
		method, path, auth string
		body               []byte
	}
	gotCh := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotCh <- captured{r.Method, r.URL.Path, r.Header.Get("Authorization"), b}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"sent"}]}}`))
	}))
	defer srv.Close()

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID: 9,
		UserText:  "ping",
		MCPServers: []agentruntime.MCPServerSpec{{
			Name:    "group",
			URL:     srv.URL + "/mcp/group/",
			Headers: map[string]string{"Authorization": "Bearer tok"},
			Tools:   []string{"group_send"},
		}},
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	select {
	case got := <-gotCh:
		assert.Equal(t, http.MethodPost, got.method)
		assert.Equal(t, "/mcp/group/", got.path)
		assert.Equal(t, "Bearer tok", got.auth)
		var rpc struct {
			Method string `json:"method"`
			Params struct {
				Name      string `json:"name"`
				Arguments struct {
					Body     string   `json:"body"`
					Mentions []string `json:"mentions"`
				} `json:"arguments"`
			} `json:"params"`
		}
		require.NoError(t, json.Unmarshal(got.body, &rpc))
		assert.Equal(t, "tools/call", rpc.Method)
		assert.Equal(t, "group_send", rpc.Params.Name)
		assert.Equal(t, ReplyPrefix+"ping", rpc.Params.Arguments.Body)
		assert.Equal(t, []string{"用户"}, rpc.Params.Arguments.Mentions)
	case <-time.After(time.Second):
		t.Fatal("fake did not call group_send")
	}
}

// 防御:注入的 MCP server 不含 group_send tool(或无任何注入,如单聊 / 非群 turn)时,
// fake 绝不对外发任何请求 —— 守 smoke-chat / session-reload 这类无 MCPServers 链路不受影响。
func TestRun_SkipsGroupSendWithoutTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	calledCh := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case calledCh <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID: 3,
		UserText:  "ping",
		MCPServers: []agentruntime.MCPServerSpec{{
			Name:  "other",
			URL:   srv.URL + "/mcp/other/",
			Tools: []string{"some_other_tool"},
		}},
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	select {
	case <-calledCh:
		t.Fatal("fake made an outbound call despite no group_send tool")
	default:
	}
}

// 建群指令:三段冒号分隔(title:成员逗号分隔:brief),取指令所在行;缺段/空段 → !ok。
func TestParseGroupCreateDirective(t *testing.T) {
	title, members, brief, ok := parseGroupCreateDirective("(来自 用户)\ne2e-group-create:拉起群:E2E Member:e2e-brief 建群冒烟")
	require.True(t, ok)
	assert.Equal(t, "拉起群", title)
	assert.Equal(t, []string{"E2E Member"}, members)
	assert.Equal(t, "e2e-brief 建群冒烟", brief)

	title, members, brief, ok = parseGroupCreateDirective("e2e-group-create:多人群: A , B ,:brief")
	require.True(t, ok)
	assert.Equal(t, "多人群", title)
	assert.Equal(t, []string{"A", "B"}, members) // 空段剔除,前后空白裁剪
	assert.Equal(t, "brief", brief)

	for _, bad := range []string{
		"无指令文本",
		"e2e-group-create:只有两段:成员",
		"e2e-group-create::E2E Member:brief", // 空 title
		"e2e-group-create:群: ,:brief",       // 成员全空
		"e2e-group-create:群:成员:",            // 空 brief
	} {
		_, _, _, ok := parseGroupCreateDirective(bad)
		assert.False(t, ok, "input=%q", bad)
	}
}

// 单聊轮:用户指令 e2e-group-create:<title>:<members>:<brief> + 注入 group_create tool
// → fake 调一次 group_create(挂起等审批由 svc 侧负责,这里 server 即时应答)。
func TestRun_PostsGroupCreateOnDirective(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv, snapshot := taskCaptureServer(t)

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID: 21,
		UserText:  "e2e-group-create:拉起群:E2E Member:e2e-brief 建群冒烟",
		MCPServers: []agentruntime.MCPServerSpec{{
			Name:    "group",
			URL:     srv.URL + "/mcp/group/",
			Headers: map[string]string{"Authorization": "Bearer tok"},
			Tools:   []string{"group_create"},
		}},
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	calls := snapshot()
	require.Len(t, calls["group_create"], 1)
	args := calls["group_create"][0]
	assert.Equal(t, "拉起群", args["title"])
	assert.Equal(t, []any{"E2E Member"}, args["memberNames"])
	assert.Equal(t, "e2e-brief 建群冒烟", args["brief"])
	assert.Empty(t, calls["group_send"]) // 单聊注入只有 group_create,绝不误发 group_send
}

// 群成员轮(注入 group_send 等,但无 group_create)即便回显文本含指令,也绝不调 group_create
// —— 守已有群聊/任务链路不被新接缝串扰。
func TestRun_SkipsGroupCreateWithoutTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv, snapshot := taskCaptureServer(t)

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID:  22,
		UserText:   "(来自 用户)\ne2e-group-create:拉起群:E2E Member:brief",
		MCPServers: taskToolsSpec(srv.URL),
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	assert.Empty(t, snapshot()["group_create"])
}

// 注入了 group_create 但 UserText 无指令(普通单聊轮)→ 绝不调 group_create
// —— 对称 TestRun_SkipsTaskCallsWithoutPatterns,守 smoke-chat 这类普通单聊轮不被误触发。
func TestRun_SkipsGroupCreateWhenNoDirective(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv, snapshot := taskCaptureServer(t)

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID: 23,
		UserText:  "ping",
		MCPServers: []agentruntime.MCPServerSpec{{
			Name:    "group",
			URL:     srv.URL + "/mcp/group/",
			Headers: map[string]string{"Authorization": "Bearer tok"},
			Tools:   []string{"group_create"},
		}},
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	assert.Empty(t, snapshot()["group_create"])
}

func TestCapabilities_DeclaresMCPTools(t *testing.T) {
	// 群聊门控(group_svc.backendSupportsGroup)要求后端声明 CapMCPTools;
	// fake 不声明的话,e2e 里建群入口一个可选 agent 都没有,群聊流程无法验证。
	caps := New().Capabilities()
	assert.True(t, caps.Has(capability.CapMCPTools))
	assert.True(t, caps.Has(capability.CapAbort))
}

// taskCaptureServer 收集本轮 fake 发出的全部 tools/call,按 tool 名归档参数。
func taskCaptureServer(t *testing.T) (*httptest.Server, func() map[string][]map[string]any) {
	t.Helper()
	var mu sync.Mutex
	calls := map[string][]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var rpc struct {
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		require.NoError(t, json.Unmarshal(b, &rpc))
		mu.Lock()
		calls[rpc.Params.Name] = append(calls[rpc.Params.Name], rpc.Params.Arguments)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() map[string][]map[string]any {
		mu.Lock()
		defer mu.Unlock()
		out := map[string][]map[string]any{}
		for k, v := range calls {
			out[k] = append([]map[string]any(nil), v...)
		}
		return out
	}
}

func taskToolsSpec(url string) []agentruntime.MCPServerSpec {
	return []agentruntime.MCPServerSpec{{
		Name:    "group",
		URL:     url + "/mcp/group/",
		Headers: map[string]string{"Authorization": "Bearer tok"},
		Tools:   []string{"group_send", "group_task_create", "group_task_complete", "group_task_cancel"},
	}}
}

// 主持人轮:用户指令 e2e-task:<assignee>:<title> → fake 调 group_task_create 派活
// (brief 为确定性派生文本),group_send 照常发(回显冒泡进群)。
func TestRun_PostsTaskCreateOnDirective(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv, snapshot := taskCaptureServer(t)

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID:  11,
		UserText:   "(来自 用户)\ne2e-task:E2E Member:重构 UI",
		MCPServers: taskToolsSpec(srv.URL),
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	calls := snapshot()
	require.Len(t, calls["group_task_create"], 1)
	args := calls["group_task_create"][0]
	assert.Equal(t, "E2E Member", args["assignee"])
	assert.Equal(t, "重构 UI", args["title"])
	assert.Equal(t, "e2e-brief: 重构 UI", args["brief"])
	assert.Len(t, calls["group_send"], 1)         // 既有回显行为不受影响
	assert.Empty(t, calls["group_task_complete"]) // 指令轮绝不交付
}

// 成员轮:收到派活抬头「任务 #N：」→ fake 调 group_task_complete 交付,
// result 带 TaskResultPrefix(DB oracle 据此断言)。
func TestRun_PostsTaskCompleteOnAssignedTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv, snapshot := taskCaptureServer(t)

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{
		SessionID:  12,
		UserText:   "(来自 CEO 助手)\n任务 #3：重构 UI\ne2e-brief: 重构 UI",
		MCPServers: taskToolsSpec(srv.URL),
	})
	require.NoError(t, err)
	for range events { //nolint:revive // draining
	}

	calls := snapshot()
	require.Len(t, calls["group_task_complete"], 1)
	args := calls["group_task_complete"][0]
	assert.Equal(t, float64(3), args["taskId"]) // JSON 数字解码为 float64
	result, _ := args["result"].(string)
	assert.True(t, strings.HasPrefix(result, TaskResultPrefix), "result=%q", result)
	assert.Empty(t, calls["group_task_create"])
}

// 无指令、无派活抬头(含「任务 #N 已完成」回执)→ 绝不碰 task tool,只 group_send。
// 守主持人收 completed 后的末轮自然收敛 + 普通群聊轮(group-chat.spec)行为不变。
func TestRun_SkipsTaskCallsWithoutPatterns(t *testing.T) {
	for _, userText := range []string{
		"(来自 用户)\ngroup ping",
		"(来自 E2E Member)\n任务 #3 已完成\ne2e-fake-result: task #3",
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		srv, snapshot := taskCaptureServer(t)

		r := New()
		events, _, err := r.Run(ctx, agentruntime.RunRequest{
			SessionID:  13,
			UserText:   userText,
			MCPServers: taskToolsSpec(srv.URL),
		})
		require.NoError(t, err)
		for range events { //nolint:revive // draining
		}

		calls := snapshot()
		assert.Empty(t, calls["group_task_create"], "userText=%q", userText)
		assert.Empty(t, calls["group_task_complete"], "userText=%q", userText)
		assert.Len(t, calls["group_send"], 1, "userText=%q", userText)
		cancel()
	}
}
