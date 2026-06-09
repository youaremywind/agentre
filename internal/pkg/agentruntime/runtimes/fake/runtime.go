//go:build e2e

// Package fake 提供 e2e 专用的确定性 agent runtime:不起任何子进程,按 req.UserText
// 回显一段固定前缀文本后正常结束。仅在 `-tags e2e` 构建中编译,生产二进制不含本包。
package fake

import (
	"context"
	"fmt"

	"agentre/internal/pkg/agentruntime"
	"agentre/internal/pkg/agentruntime/capability"
)

// ReplyPrefix 是所有假回复的前缀,前端据此断言并与用户消息区分。
const ReplyPrefix = "e2e-fake-reply: "

// Runtime 实现 agentruntime.Runtime。
type Runtime struct{}

// New 返回一个 fake runtime。
func New() *Runtime { return &Runtime{} }

// Capabilities 返回一个最小能力集(仅支持中止),足够驱动聊天 UI。
func (r *Runtime) Capabilities() capability.Capabilities {
	return capability.Capabilities{
		Set: map[capability.Capability]bool{
			capability.CapAbort: true,
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
	go func() {
		defer close(out)
		for _, chunk := range splitChunks(reply, 8) {
			select {
			case <-ctx.Done():
				return
			case out <- agentruntime.TextDelta{Text: chunk}:
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
