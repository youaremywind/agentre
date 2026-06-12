package group_svc

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGroupMCPCreateToken(t *testing.T) {
	Convey("MintCreateToken/lookupCreate 往返;成员 token 不被 create 通道接受、create token 不被成员通道接受", t, func() {
		h := newGroupMCP(nil)
		tok := h.MintCreateToken(7, 99)
		ref, ok := h.lookupCreate(tok)
		So(ok, ShouldBeTrue)
		So(ref.agentID, ShouldEqual, 7)
		So(ref.sessionID, ShouldEqual, 99)
		// 同 (agent, session) 确定性
		So(h.MintCreateToken(7, 99), ShouldEqual, tok)
		// 成员 token 进 create 通道 → 拒
		_, ok = h.lookupCreate(h.MintToken(5, 100))
		So(ok, ShouldBeFalse)
		// create token 进成员通道 → 拒
		_, ok = h.lookup(tok)
		So(ok, ShouldBeFalse)
	})
}

func TestGroupMCPGroupCreateTool(t *testing.T) {
	Convey("group_create → 回调收到 agentID/sessionID/title/memberNames/brief,响应回传回调 text", t, func() {
		var gotAgent, gotSession int64
		var gotTitle, gotBrief string
		var gotMembers []string
		h := newGroupMCP(nil)
		h.groupCreate = func(_ context.Context, agentID, sessionID int64, title string, memberNames []string, brief string) (string, error) {
			gotAgent, gotSession, gotTitle, gotMembers, gotBrief = agentID, sessionID, title, memberNames, brief
			return "group created: id=12 title=" + title, nil
		}
		tok := h.MintCreateToken(7, 99)
		rr := postMCP(h, tok, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"group_create","arguments":{"title":"新功能开发组","memberNames":["开发","测试"],"brief":"按设计稿重构 UI,验收:e2e 通过"}}}`)
		So(rr.Code, ShouldEqual, 200)
		So(gotAgent, ShouldEqual, 7)
		So(gotSession, ShouldEqual, 99)
		So(gotTitle, ShouldEqual, "新功能开发组")
		So(gotMembers, ShouldResemble, []string{"开发", "测试"})
		So(gotBrief, ShouldEqual, "按设计稿重构 UI,验收:e2e 通过")
		So(rr.Body.String(), ShouldContainSubstring, "group created: id=12")
	})

	Convey("成员 token 调 group_create → 401;create token 调 group_send → 401", t, func() {
		h := newGroupMCP(nil)
		h.groupCreate = func(context.Context, int64, int64, string, []string, string) (string, error) { return "", nil }
		rr := postMCP(h, h.MintToken(5, 100), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"group_create","arguments":{"title":"x","memberNames":["a"],"brief":"b"}}}`)
		So(rr.Code, ShouldEqual, 401)
		rr = postMCP(h, h.MintCreateToken(7, 99), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"group_send","arguments":{"body":"hi"}}}`)
		So(rr.Code, ShouldEqual, 401)
	})

	Convey("groupCreate 未装配时 create token 调 group_create → 防御分支命中(200 + JSON-RPC error)", t, func() {
		h := newGroupMCP(nil)
		rr := postMCP(h, h.MintCreateToken(7, 99), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"group_create","arguments":{"title":"x","memberNames":["a"],"brief":"b"}}}`)
		So(rr.Code, ShouldEqual, 200)
		So(rr.Body.String(), ShouldContainSubstring, "group create not wired")
	})

	Convey("tools/list 含 group_create schema", t, func() {
		h := newGroupMCP(nil)
		rr := postMCP(h, "", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
		So(rr.Body.String(), ShouldContainSubstring, `"group_create"`)
	})
}
