package skill_svc

import (
	"context"
	"testing"

	"github.com/agentre-ai/agentre/internal/model/entity/agent_backend_entity"
	"github.com/agentre-ai/agentre/internal/model/entity/agent_entity"
	"github.com/agentre-ai/agentre/internal/pkg/agentskill"
	"github.com/agentre-ai/agentre/internal/service/skill_svc/mock_skill_svc"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

type fakeDisc struct{ packs []agentskill.SkillPack }

func (f fakeDisc) Discover(_ context.Context, _ agentskill.DiscoverQuery) ([]agentskill.SkillPack, error) {
	return f.packs, nil
}

func newForTest(a AgentLookup, b BackendLookup) *Service { return &Service{agent: a, backend: b} }

func TestListAgentSkillPacks(t *testing.T) {
	Convey("合并推荐 + 发现 + 授权标注", t, func() {
		ctrl := gomock.NewController(t)
		al := mock_skill_svc.NewMockAgentLookup(ctrl)
		bl := mock_skill_svc.NewMockBackendLookup(ctrl)
		ag := &agent_entity.Agent{ID: 1, AgentBackendID: 9}
		ag.SetSkills([]agent_entity.AgentSkillItem{{ID: "superpowers@claude-plugins-official", Enabled: true}})
		al.EXPECT().Find(gomock.Any(), int64(1)).Return(ag, nil).AnyTimes()
		bl.EXPECT().Find(gomock.Any(), int64(9)).Return(&agent_backend_entity.AgentBackend{Type: string(agent_backend_entity.TypeClaudeCode)}, nil).AnyTimes()
		restore := agentskill.SwapDiscovererForTest(agent_backend_entity.TypeClaudeCode, fakeDisc{[]agentskill.SkillPack{
			{ID: "superpowers@claude-plugins-official", Name: "superpowers", Installed: true, Source: agentskill.SourceInstalled},
			{ID: "opsctl@opskat", Name: "opsctl", Installed: true, Source: agentskill.SourceInstalled},
		}})
		defer restore()
		s := newForTest(al, bl)

		Convey("ListAgentSkillPacks", func() {
			cat, err := s.ListAgentSkillPacks(context.Background(), 1, false)
			So(err, ShouldBeNil)
			byID := map[string]SkillPackDTO{}
			for _, p := range cat.Packs {
				byID[p.ID] = p
			}
			So(byID["superpowers@claude-plugins-official"].Enabled, ShouldBeTrue)
			So(byID["superpowers@claude-plugins-official"].Installed, ShouldBeTrue)
			So(byID["superpowers@claude-plugins-official"].Recommended, ShouldBeTrue) // 推荐∩安装
			So(byID["opsctl@opskat"].Enabled, ShouldBeFalse)
			So(byID["code-review@claude-plugins-official"].Recommended, ShouldBeTrue)
			So(byID["code-review@claude-plugins-official"].Installed, ShouldBeFalse)
			So(byID["code-review@claude-plugins-official"].Enabled, ShouldBeFalse)
		})
		Convey("EnabledPluginsMap = 全部已安装 → 是否授予(含 false)", func() {
			m, err := s.EnabledPluginsMap(context.Background(), 1)
			So(err, ShouldBeNil)
			So(m["superpowers@claude-plugins-official"], ShouldBeTrue)
			So(m["opsctl@opskat"], ShouldBeFalse) // 已装未授予 → 显式 false(约束子集)
		})
	})
}
