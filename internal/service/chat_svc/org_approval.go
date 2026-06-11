package chat_svc

import (
	"context"
	"fmt"

	"github.com/agentre-ai/agentre/internal/repository/chat_repo"
	"github.com/agentre-ai/agentre/internal/service/chat_svc/blocks"
)

// orgApprovalBlockToChatBlock 历史回放/overlay 路径：持久化 block → 前端 ChatBlock。
func orgApprovalBlockToChatBlock(b blocks.OrgApprovalBlock) ChatBlock {
	return ChatBlock{
		Type: "org_approval",
		OrgApproval: &ChatBlockOrgApproval{
			RequestID: b.RequestID,
			ToolName:  b.ToolName,
			ToolInput: b.ToolInput,
			Status:    b.Status,
			Result:    b.Result,
		},
	}
}

// BeginOrgApproval 在 sessionID 当前活跃 turn 上登记一条 pending 审批并推流事件。
// 无活跃 turn 返回错误(orgtool MCP handler 据此拒绝工具调用)。
func (s *chatSvc) BeginOrgApproval(ctx context.Context, sessionID int64, blk *blocks.OrgApprovalBlock) error {
	streamAny, ok := s.activeTurnStreams.Load(sessionID)
	if !ok {
		return fmt.Errorf("chat_svc.BeginOrgApproval: no active turn for session %d", sessionID)
	}
	stream := streamAny.(string)
	s.orgApprovalsMu.Lock()
	s.orgApprovals[sessionID] = append(s.orgApprovals[sessionID], blk)
	snapshot := *blk
	s.orgApprovalsMu.Unlock()

	s.emitter.Emit(ctx, stream, orgApprovalEventPayload(snapshot))
	if sess, err := chat_repo.Session().Find(ctx, sessionID); err == nil && sess != nil {
		s.markSessionWaiting(ctx, sess, stream)
	}
	return nil
}

// FinishOrgApproval 把审批置为终态(approved/denied/expired)并推 resolved 事件。
func (s *chatSvc) FinishOrgApproval(ctx context.Context, sessionID int64, requestID, status, result string) error {
	s.orgApprovalsMu.Lock()
	var snapshot *blocks.OrgApprovalBlock
	for _, b := range s.orgApprovals[sessionID] {
		if b.RequestID == requestID {
			b.Status = status
			b.Result = result
			cp := *b
			snapshot = &cp
			break
		}
	}
	s.orgApprovalsMu.Unlock()
	if snapshot == nil {
		return fmt.Errorf("chat_svc.FinishOrgApproval: request %s not found (turn finalized?)", requestID)
	}
	if streamAny, ok := s.activeTurnStreams.Load(sessionID); ok {
		stream := streamAny.(string)
		s.emitter.Emit(ctx, stream, orgApprovalEventPayload(*snapshot))
		if sess, err := chat_repo.Session().Find(ctx, sessionID); err == nil && sess != nil {
			s.markSessionRunning(ctx, sess, stream)
		}
	}
	return nil
}

func orgApprovalEventPayload(b blocks.OrgApprovalBlock) map[string]any {
	return map[string]any{
		"kind":      "org_approval",
		"requestId": b.RequestID,
		"toolName":  b.ToolName,
		"toolInput": b.ToolInput,
		"status":    b.Status,
		"result":    b.Result,
	}
}

// takeOrgApprovals finalize 时取走本会话全部审批 block;仍 pending 的标 expired
// (turn 被 abort / 子进程死亡时挂起审批不再可决)。
func (s *chatSvc) takeOrgApprovals(sessionID int64) []*blocks.OrgApprovalBlock {
	s.orgApprovalsMu.Lock()
	defer s.orgApprovalsMu.Unlock()
	out := s.orgApprovals[sessionID]
	delete(s.orgApprovals, sessionID)
	for _, b := range out {
		if b.Status == "pending" {
			b.Status = "expired"
		}
	}
	return out
}

// snapshotOrgApprovals LoadSession overlay 用:拷贝当前登记的 block(不取走)。
func (s *chatSvc) snapshotOrgApprovals(sessionID int64) []blocks.OrgApprovalBlock {
	s.orgApprovalsMu.Lock()
	defer s.orgApprovalsMu.Unlock()
	out := make([]blocks.OrgApprovalBlock, 0, len(s.orgApprovals[sessionID]))
	for _, b := range s.orgApprovals[sessionID] {
		out = append(out, *b)
	}
	return out
}
