package app

import (
	"github.com/agentre-ai/agentre/internal/service/orgtool_svc"
)

// AnswerOrgApproval 组织架构工具写操作的审批决策(批准/拒绝)。
func (a *App) AnswerOrgApproval(req *orgtool_svc.AnswerOrgApprovalRequest) (*orgtool_svc.AnswerOrgApprovalResponse, error) {
	return orgtool_svc.Default().AnswerOrgApproval(a.ctx, req)
}
