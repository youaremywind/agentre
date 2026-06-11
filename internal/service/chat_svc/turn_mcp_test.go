package chat_svc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/agentre-ai/agentre/internal/model/entity/agent_entity"
	"github.com/agentre-ai/agentre/internal/pkg/agentruntime"
)

func TestAppendTurnMCP(t *testing.T) {
	a := &agent_entity.Agent{ID: 5}
	base := []agentruntime.MCPServerSpec{{Name: "group"}}

	// provider 未注册 → 原样
	RegisterTurnMCPProvider(nil)
	require.Equal(t, base, appendTurnMCP(context.Background(), base, a, 42, true))

	// 注册后 + 支持 CapMCPTools → 追加
	RegisterTurnMCPProvider(func(_ context.Context, ag *agent_entity.Agent, sid int64) []agentruntime.MCPServerSpec {
		require.Equal(t, int64(5), ag.ID)
		require.Equal(t, int64(42), sid)
		return []agentruntime.MCPServerSpec{{Name: "org"}}
	})
	got := appendTurnMCP(context.Background(), base, a, 42, true)
	require.Len(t, got, 2)
	require.Equal(t, "org", got[1].Name)

	// runner 不支持 CapMCPTools → 不追加(软降级)
	require.Equal(t, base, appendTurnMCP(context.Background(), base, a, 42, false))
	RegisterTurnMCPProvider(nil) // 清理,防测试间串台
}
