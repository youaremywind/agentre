package blocks

import (
	"testing"

	cagoblocks "github.com/cago-frame/agents/agent/blocks"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOrgApprovalBlock_TypeAndAudience(t *testing.T) {
	Convey("OrgApprovalBlock 类型 + Audience", t, func() {
		b := OrgApprovalBlock{}
		So(b.Type(), ShouldEqual, "org_approval")
		So(b.Audience(), ShouldEqual, cagoblocks.ToUI)
	})
}

func TestOrgApprovalBlock_FactoryRoundTrip(t *testing.T) {
	Convey("OrgApprovalBlock Encode/Decode round-trip", t, func() {
		b := &OrgApprovalBlock{
			RequestID: "org-req-1",
			ToolName:  "org_invite",
			ToolInput: map[string]any{"user_id": "u-42"},
			Status:    "pending",
			Result:    "",
		}
		sb, err := cagoblocks.Encode(b)
		So(err, ShouldBeNil)
		So(sb.Type, ShouldEqual, "org_approval")

		decoded, err := cagoblocks.Decode(sb)
		So(err, ShouldBeNil)
		got, ok := decoded.(OrgApprovalBlock)
		So(ok, ShouldBeTrue)
		So(got.RequestID, ShouldEqual, "org-req-1")
		So(got.ToolName, ShouldEqual, "org_invite")
		So(got.ToolInput["user_id"], ShouldEqual, "u-42")
		So(got.Status, ShouldEqual, "pending")
	})
}
