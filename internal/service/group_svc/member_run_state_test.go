package group_svc_test

import (
	"context"
	"testing"
	"time"

	"github.com/cago-frame/cago/pkg/consts"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"agentre/internal/model/entity/group_entity"
	"agentre/internal/repository/group_repo"
	"agentre/internal/repository/group_repo/mock_group_repo"
	"agentre/internal/service/chat_svc"
	"agentre/internal/service/group_svc"
	"agentre/internal/service/group_svc/mock_group_svc"
)

type memberRunStateEvent struct {
	memberID int64
	runState string
}

// memberRunStateEmitter 抽出 member_run_state 事件的 (memberID, runState) 投到 ch。
func memberRunStateEmitter(ch chan<- memberRunStateEvent) group_svc.EmitterFunc {
	return func(_ context.Context, _ string, payload any) {
		p, ok := payload.(map[string]any)
		if !ok || p["kind"] != "member_run_state" {
			return
		}
		mid, _ := p["memberID"].(int64)
		rs, _ := p["runState"].(string)
		select {
		case ch <- memberRunStateEvent{memberID: mid, runState: rs}:
		default:
		}
	}
}

func waitForMemberRunState(ch <-chan memberRunStateEvent, memberID int64, want string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case e := <-ch:
			if e.memberID == memberID && e.runState == want {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func TestScheduler_EmitsMemberRunStateRunningThenIdle(t *testing.T) {
	Convey("成员 turn 起 → 推 member_run_state running; turn 结束 → 推 idle", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		gw := mock_group_svc.NewMockChatGateway(ctrl)
		groupRepo := mock_group_repo.NewMockGroupRepo(ctrl)
		memberRepo := mock_group_repo.NewMockGroupMemberRepo(ctrl)
		msgRepo := mock_group_repo.NewMockGroupMessageRepo(ctrl)
		group_repo.RegisterGroup(groupRepo)
		group_repo.RegisterMember(memberRepo)
		group_repo.RegisterMessage(msgRepo)

		g := &group_entity.Group{ID: 5, RunStatus: group_entity.RunRunning, Status: consts.ACTIVE}
		groupRepo.EXPECT().Find(gomock.Any(), int64(5)).Return(g, nil).AnyTimes()
		groupRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		members := []*group_entity.GroupMember{
			{ID: 1, GroupID: 5, AgentID: 1, BackingSessionID: 11, Role: group_entity.RoleHost, Status: group_entity.MemberActive},
		}
		memberRepo.EXPECT().ListByGroup(gomock.Any(), int64(5)).Return(members, nil).AnyTimes()
		msgRepo.EXPECT().NextSeq(gomock.Any(), int64(5)).Return(1, nil).AnyTimes()
		msgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		ch11 := make(chan chat_svc.TurnResult, 1)
		gw.EXPECT().ObserveTurn(int64(11)).Return((<-chan chat_svc.TurnResult)(ch11), func() {}).AnyTimes()
		gw.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, req *chat_svc.SendRequest) (*chat_svc.SendResponse, error) {
				return &chat_svc.SendResponse{SessionID: req.SessionID}, nil
			}).AnyTimes()

		svc := group_svc.NewForTestWithNames(gw, map[int64]string{1: "林队"})
		runStateCh := make(chan memberRunStateEvent, 16)
		group_svc.SetEmitterForTest(svc, memberRunStateEmitter(runStateCh))

		So(svc.SendGroupMessage(ctx, &group_svc.SendGroupMessageRequest{GroupID: 5, Text: "干活", RecipientMemberIDs: []int64{1}}), ShouldBeNil)
		So(waitForMemberRunState(runStateCh, 1, "running", 2*time.Second), ShouldBeTrue)

		ch11 <- chat_svc.TurnResult{SessionID: 11}
		So(waitForMemberRunState(runStateCh, 1, "idle", 2*time.Second), ShouldBeTrue)
	})
}

func TestLoadGroup_MemberRunStatesReflectInflight(t *testing.T) {
	Convey("LoadGroup 的 MemberRunStates 跟随调度器在跑态: 在跑→running, 否则 idle", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		gw := mock_group_svc.NewMockChatGateway(ctrl)
		groupRepo := mock_group_repo.NewMockGroupRepo(ctrl)
		memberRepo := mock_group_repo.NewMockGroupMemberRepo(ctrl)
		msgRepo := mock_group_repo.NewMockGroupMessageRepo(ctrl)
		group_repo.RegisterGroup(groupRepo)
		group_repo.RegisterMember(memberRepo)
		group_repo.RegisterMessage(msgRepo)

		g := &group_entity.Group{ID: 5, RunStatus: group_entity.RunRunning, Status: consts.ACTIVE}
		groupRepo.EXPECT().Find(gomock.Any(), int64(5)).Return(g, nil).AnyTimes()
		members := []*group_entity.GroupMember{
			{ID: 1, GroupID: 5, AgentID: 1, BackingSessionID: 11, Role: group_entity.RoleHost, Status: group_entity.MemberActive},
			{ID: 2, GroupID: 5, AgentID: 2, BackingSessionID: 12, Status: group_entity.MemberActive},
		}
		memberRepo.EXPECT().ListByGroup(gomock.Any(), int64(5)).Return(members, nil).AnyTimes()
		msgRepo.EXPECT().ListByGroup(gomock.Any(), int64(5)).Return(nil, nil).AnyTimes()

		svc := group_svc.NewForTestWithNames(gw, map[int64]string{1: "林队", 2: "后端"})
		group_svc.MarkInflightForTest(svc, 5, 2)

		d, err := svc.LoadGroup(ctx, 5)
		So(err, ShouldBeNil)
		So(d.MemberRunStates[1], ShouldEqual, "idle")
		So(d.MemberRunStates[2], ShouldEqual, "running")
	})
}
