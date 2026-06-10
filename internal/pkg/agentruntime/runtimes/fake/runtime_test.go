//go:build e2e

package fake

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestCapabilities_DeclaresMCPTools(t *testing.T) {
	// 群聊门控(group_svc.backendSupportsGroup)要求后端声明 CapMCPTools;
	// fake 不声明的话,e2e 里建群入口一个可选 agent 都没有,群聊流程无法验证。
	caps := New().Capabilities()
	assert.True(t, caps.Has(capability.CapMCPTools))
	assert.True(t, caps.Has(capability.CapAbort))
}
