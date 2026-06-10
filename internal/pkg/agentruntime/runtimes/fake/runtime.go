//go:build e2e

// Package fake 提供 e2e 专用的确定性 agent runtime:不起任何子进程,按 req.UserText
// 回显一段固定前缀文本后正常结束。仅在 `-tags e2e` 构建中编译,生产二进制不含本包。
package fake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/agentre-ai/agentre/internal/pkg/agentruntime"
	"github.com/agentre-ai/agentre/internal/pkg/agentruntime/capability"
)

// ReplyPrefix 是所有假回复的前缀,前端据此断言并与用户消息区分。
const ReplyPrefix = "e2e-fake-reply: "

// Runtime 实现 agentruntime.Runtime。
type Runtime struct{}

// New 返回一个 fake runtime。
func New() *Runtime { return &Runtime{} }

// Capabilities 返回驱动聊天 + 群聊 UI 的最小能力集:CapAbort 支撑停止按钮;
// CapMCPTools 让群聊门控(group_svc.backendSupportsGroup)放行,否则 e2e 里
// 建群入口没有任何可选 agent。fake 实际忽略注入的 MCPServers(只回显文本)。
func (r *Runtime) Capabilities() capability.Capabilities {
	return capability.Capabilities{
		Set: map[capability.Capability]bool{
			capability.CapAbort:    true,
			capability.CapMCPTools: true,
		},
	}
}

// Run 把 ReplyPrefix+UserText 分片流式发送后 emit Done。
func (r *Runtime) Run(ctx context.Context, req agentruntime.RunRequest) (<-chan agentruntime.Event, *agentruntime.RunResult, error) {
	out := make(chan agentruntime.Event, 8)
	result := &agentruntime.RunResult{
		ProviderSessionID: fmt.Sprintf("e2e-fake-%d", req.SessionID),
		Model:             "e2e-fake-model",
	}
	reply := ReplyPrefix + req.UserText
	chunkDelay := configuredChunkDelay()
	go func() {
		defer close(out)
		for i, chunk := range splitChunks(reply, 8) {
			if i > 0 && chunkDelay > 0 {
				timer := time.NewTimer(chunkDelay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			select {
			case <-ctx.Done():
				return
			case out <- agentruntime.TextDelta{Text: chunk}:
			}
		}
		// 群成员 turn:像真 CLI 一样,把本轮回复经注入的 group MCP server 调 group_send
		// 冒泡进群转录区(IngestAgentMessage → group_messages)。无注入(单聊 / 非群)→ 跳过,
		// 行为不变。ctx 取消时上面的循环已提前 return,不会走到这里。
		if spec, ok := findGroupSendServer(req.MCPServers); ok {
			if err := postGroupSend(ctx, spec, reply); err != nil {
				// 尽力而为:发失败不报 ErrorEvent(避免误把 backing session 标成出错),
				// 只写日志;群气泡缺失会被 e2e spec 当作显式失败信号抓到。
				fmt.Fprintf(os.Stderr, "fake: group_send failed: %v\n", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case out <- agentruntime.Done{}:
		}
	}()
	return out, result, nil
}

// findGroupSendServer 返回首个广告 group_send tool 的注入 MCP server(群成员 turn 才有)。
func findGroupSendServer(specs []agentruntime.MCPServerSpec) (agentruntime.MCPServerSpec, bool) {
	for _, s := range specs {
		for _, t := range s.Tools {
			if t == "group_send" {
				return s, true
			}
		}
	}
	return agentruntime.MCPServerSpec{}, false
}

// postGroupSend 对注入的 group MCP server 发一次无状态 tools/call(group_send),body=回显文本,
// mentions=["用户"] 回人类来源(ingest 后无 agent 收件人 → 本轮自然收敛,不触发 agent 互投)。
// handler 的 tools/call 分支无状态,无需先做 initialize 握手。
func postGroupSend(ctx context.Context, spec agentruntime.MCPServerSpec, body string) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "group_send",
			"arguments": map[string]any{
				"body":     body,
				"mentions": []string{"用户"},
			},
		},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, spec.URL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range spec.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("group_send: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func configuredChunkDelay() time.Duration {
	raw := os.Getenv("AGENTRE_E2E_FAKE_CHUNK_DELAY_MS")
	if raw == "" {
		return 0
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// splitChunks 按 rune 边界把 s 切成最多 n 个 rune 的片段。
func splitChunks(s string, n int) []string {
	if n <= 0 || s == "" {
		return nil
	}
	runes := []rune(s)
	out := make([]string, 0, (len(runes)+n-1)/n)
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}
