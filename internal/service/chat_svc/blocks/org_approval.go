package blocks

import cagoblocks "github.com/cago-frame/agents/agent/blocks"

// OrgApprovalBlock 组织架构工具(orgtool)写操作的服务端审批卡。
// Status: pending → approved | denied | expired(超时 / turn 中止 / app 重启悬空)。
// Result 仅 approved 后有值(执行结果或业务错误摘要)。
type OrgApprovalBlock struct {
	RequestID string         `json:"request_id"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input,omitempty"`
	Status    string         `json:"status"`
	Result    string         `json:"result,omitempty"`
}

func (OrgApprovalBlock) Type() string                      { return "org_approval" }
func (OrgApprovalBlock) Audience() cagoblocks.AudienceMask { return cagoblocks.ToUI }

func init() { cagoblocks.RegisterFactory[OrgApprovalBlock]() }
