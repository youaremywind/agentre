package app

import (
	"agentre/internal/model/entity/group_entity"
	"agentre/internal/service/group_svc"
)

type GroupItem struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	RunStatus  string `json:"runStatus"`
	RoundCount int    `json:"roundCount"`
	Pinned     bool   `json:"pinned"`
	Createtime int64  `json:"createtime"`
	Updatetime int64  `json:"updatetime"`
}

type GroupMemberItem struct {
	ID               int64  `json:"id"`
	AgentID          int64  `json:"agentID"`
	BackingSessionID int64  `json:"backingSessionID"`
	Role             string `json:"role"`
	// Status 是成员身份(active/left)。RunState 是运行态(running/idle),区别于身份 ——
	// roster 的状态点用 RunState 表示该成员此刻是否在跑一轮 turn,空串按 idle 处理。
	Status   string `json:"status"`
	RunState string `json:"runState"`
}

type GroupMessageItem struct {
	ID                 int64   `json:"id"`
	Seq                int     `json:"seq"`
	SenderKind         string  `json:"senderKind"`
	SenderMemberID     int64   `json:"senderMemberID"`
	RecipientMemberIDs []int64 `json:"recipientMemberIDs"`
	ToUser             bool    `json:"toUser"`
	Content            string  `json:"content"`
	Createtime         int64   `json:"createtime"`
}

type GroupDetailResponse struct {
	Group    *GroupItem          `json:"group"`
	Members  []*GroupMemberItem  `json:"members"`
	Messages []*GroupMessageItem `json:"messages"`
}

type GroupCreateRequest struct {
	Title          string  `json:"title"`
	HostAgentID    int64   `json:"hostAgentID"`
	DepartmentID   int64   `json:"departmentID"`
	ProjectID      int64   `json:"projectID"`
	MemberAgentIDs []int64 `json:"memberAgentIDs"`
}

type GroupSendRequest struct {
	GroupID            int64   `json:"groupID"`
	Text               string  `json:"text"`
	RecipientMemberIDs []int64 `json:"recipientMemberIDs"`
	ToUser             bool    `json:"toUser"`
}

func toGroupItem(g *group_entity.Group) *GroupItem {
	return &GroupItem{ID: g.ID, Title: g.Title, RunStatus: g.RunStatus, RoundCount: g.RoundCount, Pinned: g.Pinned, Createtime: g.Createtime, Updatetime: g.Updatetime}
}

func toGroupMemberItem(m *group_entity.GroupMember) *GroupMemberItem {
	return &GroupMemberItem{ID: m.ID, AgentID: m.AgentID, BackingSessionID: m.BackingSessionID, Role: m.Role, Status: m.Status}
}

func toGroupDetail(d *group_svc.GroupDetail) *GroupDetailResponse {
	members := make([]*GroupMemberItem, 0, len(d.Members))
	for _, m := range d.Members {
		item := toGroupMemberItem(m)
		item.RunState = d.MemberRunStates[m.ID]
		members = append(members, item)
	}
	msgs := make([]*GroupMessageItem, 0, len(d.Messages))
	for _, m := range d.Messages {
		msgs = append(msgs, &GroupMessageItem{ID: m.ID, Seq: m.Seq, SenderKind: m.SenderKind, SenderMemberID: m.SenderMemberID, RecipientMemberIDs: m.Recipients(), ToUser: m.ToUser, Content: m.Content, Createtime: m.Createtime})
	}
	return &GroupDetailResponse{Group: toGroupItem(d.Group), Members: members, Messages: msgs}
}

func (a *App) GroupList() ([]*GroupItem, error) {
	gs, err := group_svc.Default().ListGroups(a.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*GroupItem, 0, len(gs))
	for _, g := range gs {
		items = append(items, toGroupItem(g))
	}
	return items, nil
}

func (a *App) GroupCreate(req *GroupCreateRequest) (*GroupDetailResponse, error) {
	d, err := group_svc.Default().CreateGroup(a.ctx, &group_svc.CreateGroupRequest{Title: req.Title, HostAgentID: req.HostAgentID, DepartmentID: req.DepartmentID, ProjectID: req.ProjectID, MemberAgentIDs: req.MemberAgentIDs})
	if err != nil {
		return nil, err
	}
	return toGroupDetail(d), nil
}

func (a *App) GroupLoad(id int64) (*GroupDetailResponse, error) {
	d, err := group_svc.Default().LoadGroup(a.ctx, id)
	if err != nil {
		return nil, err
	}
	return toGroupDetail(d), nil
}

func (a *App) GroupSend(req *GroupSendRequest) error {
	return group_svc.Default().SendGroupMessage(a.ctx, &group_svc.SendGroupMessageRequest{GroupID: req.GroupID, Text: req.Text, RecipientMemberIDs: req.RecipientMemberIDs, ToUser: req.ToUser})
}

func (a *App) GroupAddMember(groupID, agentID int64) (*GroupMemberItem, error) {
	m, err := group_svc.Default().AddGroupMember(a.ctx, groupID, agentID)
	if err != nil {
		return nil, err
	}
	return toGroupMemberItem(m), nil
}

func (a *App) GroupRemoveMember(memberID int64) error {
	return group_svc.Default().RemoveGroupMember(a.ctx, memberID)
}
func (a *App) GroupStop(id int64) error  { return group_svc.Default().StopGroup(a.ctx, id) }
func (a *App) GroupPause(id int64) error { return group_svc.Default().PauseGroup(a.ctx, id) }
func (a *App) GroupResume(id int64) error {
	return group_svc.Default().ResumeGroup(a.ctx, id)
}
func (a *App) GroupRename(id int64, title string) error {
	return group_svc.Default().RenameGroup(a.ctx, id, title)
}
func (a *App) GroupSetPinned(id int64, pinned bool) error {
	return group_svc.Default().SetGroupPinned(a.ctx, id, pinned)
}
func (a *App) GroupArchive(id int64) error { return group_svc.Default().ArchiveGroup(a.ctx, id) }
