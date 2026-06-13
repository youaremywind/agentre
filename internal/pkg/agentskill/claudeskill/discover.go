// Package claudeskill 用 `claude plugin list --json` 发现该安装的技能包。
package claudeskill

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/agentre-ai/agentre/internal/model/entity/agent_backend_entity"
	"github.com/agentre-ai/agentre/internal/pkg/agentskill"
)

func init() {
	agentskill.RegisterDiscoverer(agent_backend_entity.TypeClaudeCode, Discoverer{})
}

// Discoverer 用 claude CLI 枚举已安装技能包。
type Discoverer struct{}

// rawPlugin 映射 `claude plugin list --json` 单元素。Enabled/Scope 反映 CLI 输出
// 形状,当前发现不消费(授予与否由 skill_svc 按 agent 授权集决定,非 CLI 的 enabled)。
type rawPlugin struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Scope   string `json:"scope"`
}

func parsePluginList(b []byte) ([]agentskill.SkillPack, error) {
	out := []agentskill.SkillPack{}
	if len(b) == 0 {
		return out, nil
	}
	var raws []rawPlugin
	if err := json.Unmarshal(b, &raws); err != nil {
		return out, nil // 坏 JSON 视为无发现,不阻断
	}
	for _, r := range raws {
		name := r.ID
		if i := strings.Index(r.ID, "@"); i > 0 {
			name = r.ID[:i]
		}
		out = append(out, agentskill.SkillPack{
			ID:        r.ID,
			Name:      name,
			Source:    agentskill.SourceInstalled,
			Installed: true,
		})
	}
	return out, nil
}

// Discover 调用 claude plugin list --json 枚举已安装技能包。CLI 不可用时软降级返回空。
func (Discoverer) Discover(ctx context.Context, q agentskill.DiscoverQuery) ([]agentskill.SkillPack, error) {
	bin := strings.TrimSpace(q.CLIPath)
	if bin == "" {
		bin = "claude"
	}
	cmd := exec.CommandContext(ctx, bin, "plugin", "list", "--json")
	b, err := cmd.Output()
	if err != nil {
		return []agentskill.SkillPack{}, nil // CLI 不可用 → 软降级(空发现)
	}
	return parsePluginList(b)
}
