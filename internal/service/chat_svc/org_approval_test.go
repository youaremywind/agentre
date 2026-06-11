package chat_svc

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"github.com/agentre-ai/agentre/internal/model/entity/chat_entity"
	"github.com/agentre-ai/agentre/internal/repository/chat_repo"
	"github.com/agentre-ai/agentre/internal/repository/chat_repo/mock_chat_repo"
	"github.com/agentre-ai/agentre/internal/service/chat_svc/blocks"
)

// recordEmitter 收集每次 Emit 的 (stream, payload),供审批事件断言。
// BeginOrgApproval/FinishOrgApproval emit 的是裸 map[string]any(org_approval 事件)
// 与 ChatStreamEvent(session_status patch),两类都收。
type recordEmitter struct {
	streams  []string
	payloads []any
}

func (r *recordEmitter) Emit(_ context.Context, name string, payload any) {
	r.streams = append(r.streams, name)
	r.payloads = append(r.payloads, payload)
}

// orgApprovalEvents 过滤出 kind=org_approval 的裸 map 事件。
func (r *recordEmitter) orgApprovalEvents() []map[string]any {
	var out []map[string]any
	for _, p := range r.payloads {
		if m, ok := p.(map[string]any); ok && m["kind"] == "org_approval" {
			out = append(out, m)
		}
	}
	return out
}

func TestOrgApprovalLifecycle(t *testing.T) {
	Convey("org 审批编排 Begin/Finish/take/snapshot 全生命周期", t, func() {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		sessRepo := mock_chat_repo.NewMockSessionRepo(ctrl)
		prev := chat_repo.Session()
		chat_repo.RegisterSession(sessRepo)
		t.Cleanup(func() { chat_repo.RegisterSession(prev) })

		// Begin/Finish 都会 Find(sessionID) 然后 markSessionWaiting/Running 写库。
		// 用宽松期望:Find 总返回一个 session,Update 任意次成功。
		sessRepo.EXPECT().Find(gomock.Any(), int64(42)).
			DoAndReturn(func(_ context.Context, id int64) (*chat_entity.Session, error) {
				return &chat_entity.Session{ID: id, AgentStatus: "running"}, nil
			}).AnyTimes()
		sessRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		em := &recordEmitter{}
		svc := &chatSvc{
			emitter:      em,
			orgApprovals: map[int64][]*blocks.OrgApprovalBlock{},
		}
		ctx := context.Background()

		// 1. 无活跃 turn → Begin 返回 error。
		blk1 := &blocks.OrgApprovalBlock{
			RequestID: "req-1", ToolName: "org_invite",
			ToolInput: map[string]any{"user_id": "u-1"}, Status: "pending",
		}
		So(svc.BeginOrgApproval(ctx, 42, blk1), ShouldNotBeNil)
		So(em.orgApprovalEvents(), ShouldHaveLength, 0)

		// 2. 登记活跃 turn 流名后 Begin 成功,emitter 收到 pending 事件。
		svc.activeTurnStreams.Store(int64(42), StreamName(42, 7))
		So(svc.BeginOrgApproval(ctx, 42, blk1), ShouldBeNil)
		evs := em.orgApprovalEvents()
		So(evs, ShouldHaveLength, 1)
		So(evs[0]["requestId"], ShouldEqual, "req-1")
		So(evs[0]["toolName"], ShouldEqual, "org_invite")
		So(evs[0]["status"], ShouldEqual, "pending")

		// 3. FinishOrgApproval(denied) → emitter 收到 denied;内部块状态已更新。
		So(svc.FinishOrgApproval(ctx, 42, "req-1", "denied", "用户拒绝"), ShouldBeNil)
		evs = em.orgApprovalEvents()
		So(evs, ShouldHaveLength, 2)
		So(evs[1]["requestId"], ShouldEqual, "req-1")
		So(evs[1]["status"], ShouldEqual, "denied")
		So(evs[1]["result"], ShouldEqual, "用户拒绝")
		// 内部登记块已被改为 denied。
		snap := svc.snapshotOrgApprovals(42)
		So(snap, ShouldHaveLength, 1)
		So(snap[0].Status, ShouldEqual, "denied")

		// 4. 再 Begin 一条 pending;take 返回 2 条(denied 保留 + 新 pending 标 expired),
		//    且 take 后 snapshot 变空。
		blk2 := &blocks.OrgApprovalBlock{
			RequestID: "req-2", ToolName: "org_set_manager",
			ToolInput: map[string]any{"user_id": "u-2"}, Status: "pending",
		}
		So(svc.BeginOrgApproval(ctx, 42, blk2), ShouldBeNil)
		So(svc.snapshotOrgApprovals(42), ShouldHaveLength, 2)

		taken := svc.takeOrgApprovals(42)
		So(taken, ShouldHaveLength, 2)
		statusByID := map[string]string{}
		for _, b := range taken {
			statusByID[b.RequestID] = b.Status
		}
		So(statusByID["req-1"], ShouldEqual, "denied")  // 终态保留
		So(statusByID["req-2"], ShouldEqual, "expired") // pending → expired
		So(svc.snapshotOrgApprovals(42), ShouldHaveLength, 0)

		// 5. Finish 已取走的 requestID → error。
		So(svc.FinishOrgApproval(ctx, 42, "req-1", "approved", ""), ShouldNotBeNil)

		// 6. snapshot 返回的是拷贝,改它不影响内部登记。
		svc.activeTurnStreams.Store(int64(42), StreamName(42, 9))
		blk3 := &blocks.OrgApprovalBlock{RequestID: "req-3", ToolName: "org_invite", Status: "pending"}
		So(svc.BeginOrgApproval(ctx, 42, blk3), ShouldBeNil)
		snap2 := svc.snapshotOrgApprovals(42)
		So(snap2, ShouldHaveLength, 1)
		snap2[0].Status = "mutated-copy"
		So(svc.snapshotOrgApprovals(42)[0].Status, ShouldEqual, "pending")
	})
}
