// Package group_svc 提供群聊编排的业务逻辑层。
package group_svc

import (
	"context"

	"github.com/agentre-ai/agentre/internal/pkg/agentruntime/capability"
	"github.com/agentre-ai/agentre/internal/service/chat_svc"
)

//go:generate mockgen -source gateway.go -destination mock_group_svc/mock_gateway.go

// ChatGateway 是 group_svc 对 chat_svc 的窄依赖(只用这几个 seam, ISP)。
type ChatGateway interface {
	EnsureSession(ctx context.Context, req *chat_svc.EnsureSessionRequest) (*chat_svc.EnsureSessionResponse, error)
	Send(ctx context.Context, req *chat_svc.SendRequest) (*chat_svc.SendResponse, error)
	ObserveTurn(sessionID int64) (<-chan chat_svc.TurnResult, func())
	Stop(ctx context.Context, req *chat_svc.StopRequest) (*chat_svc.StopResponse, error)
	// DeleteSession 软删某 backing session(成员离群/群归档时清理, 避免遗留 group_id>0
	// 的 ACTIVE 会话经 ListAgents 的 IncludingGroups 变体残留在 agent 侧栏)。
	DeleteSession(ctx context.Context, sessionID int64) error
	AgentBackendHasCapability(ctx context.Context, agentID int64, wantCap capability.Capability) (bool, error)
}

// chatSvcGateway 委托给 chat_svc 默认单例。
type chatSvcGateway struct{}

func (chatSvcGateway) EnsureSession(ctx context.Context, req *chat_svc.EnsureSessionRequest) (*chat_svc.EnsureSessionResponse, error) {
	return chat_svc.Chat().EnsureSession(ctx, req)
}

func (chatSvcGateway) Send(ctx context.Context, req *chat_svc.SendRequest) (*chat_svc.SendResponse, error) {
	return chat_svc.Chat().Send(ctx, req)
}

func (chatSvcGateway) ObserveTurn(sessionID int64) (<-chan chat_svc.TurnResult, func()) {
	return chat_svc.Chat().ObserveTurn(sessionID)
}

func (chatSvcGateway) Stop(ctx context.Context, req *chat_svc.StopRequest) (*chat_svc.StopResponse, error) {
	return chat_svc.Chat().Stop(ctx, req)
}

func (chatSvcGateway) DeleteSession(ctx context.Context, sessionID int64) error {
	_, err := chat_svc.Chat().Delete(ctx, &chat_svc.DeleteRequest{SessionID: sessionID})
	return err
}

func (chatSvcGateway) AgentBackendHasCapability(ctx context.Context, agentID int64, wantCap capability.Capability) (bool, error) {
	return chat_svc.Chat().AgentBackendHasCapability(ctx, agentID, wantCap)
}
