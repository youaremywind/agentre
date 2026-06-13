package claudeskill

import (
	"testing"

	"github.com/agentre-ai/agentre/internal/pkg/agentskill"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParsePluginList(t *testing.T) {
	Convey("解析 plugin list --json", t, func() {
		raw := []byte(`[
		  {"id":"superpowers@claude-plugins-official","enabled":true,"scope":"user"},
		  {"id":"opsctl@opskat","enabled":false,"scope":"user"}
		]`)
		packs, err := parsePluginList(raw)
		So(err, ShouldBeNil)
		So(len(packs), ShouldEqual, 2)
		So(packs[0].ID, ShouldEqual, "superpowers@claude-plugins-official")
		So(packs[0].Name, ShouldEqual, "superpowers") // id 取 @ 前段
		So(packs[0].Installed, ShouldBeTrue)
		So(packs[0].Source, ShouldEqual, agentskill.SourceInstalled)
		Convey("空/坏 JSON → 空,不 panic", func() {
			p, _ := parsePluginList([]byte(""))
			So(p, ShouldResemble, []agentskill.SkillPack{})
			p2, _ := parsePluginList([]byte("not json"))
			So(p2, ShouldResemble, []agentskill.SkillPack{})
		})
		Convey("无 @ 的裸 id → Name 即 id", func() {
			p, _ := parsePluginList([]byte(`[{"id":"barepack","enabled":true,"scope":"user"}]`))
			So(len(p), ShouldEqual, 1)
			So(p[0].ID, ShouldEqual, "barepack")
			So(p[0].Name, ShouldEqual, "barepack")
		})
	})
}
