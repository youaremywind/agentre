package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigration202606110002GroupTasksWorkflows(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 前置:群聊基线依赖 agents + chat_sessions 表。
	require.NoError(t, migration202605220004().Migrate(gdb))
	require.NoError(t, migration202605220006().Migrate(gdb))
	require.NoError(t, migration202606030001().Migrate(gdb))
	require.NoError(t, migration202606110002().Migrate(gdb))

	// group_tasks 表可写入且 (group_id, task_no) 唯一。
	require.NoError(t, gdb.Exec(`INSERT INTO group_tasks
		(group_id, task_no, title, brief, creator_member_id, assignee_member_id, status)
		VALUES (1, 1, '重构设置页', '按新稿', 100, 101, 'open')`).Error)
	err = gdb.Exec(`INSERT INTO group_tasks
		(group_id, task_no, title, creator_member_id, assignee_member_id, status)
		VALUES (1, 1, '重复编号', 100, 101, 'open')`).Error
	require.Error(t, err, "同群同 task_no 必须被唯一索引拒绝")

	// group_messages 两列默认值。
	require.NoError(t, gdb.Exec(`INSERT INTO group_messages (group_id, seq, content) VALUES (1, 1, 'x')`).Error)
	var msg struct {
		TaskID    int64
		TaskEvent string
	}
	require.NoError(t, gdb.Table("group_messages").Select("task_id, task_event").
		Where("group_id = 1").Scan(&msg).Error)
	require.Equal(t, int64(0), msg.TaskID)
	require.Equal(t, "", msg.TaskEvent)

	// workflows 表 + groups.workflow_id。
	require.NoError(t, gdb.Exec(`INSERT INTO workflows (name, content, status) VALUES ('产品开发流程', '# 流程', 1)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO groups (title, host_agent_id, workflow_id) VALUES ('g', 1, 1)`).Error)
	var wf int64
	require.NoError(t, gdb.Table("groups").Select("workflow_id").Where("title = 'g'").Scan(&wf).Error)
	require.Equal(t, int64(1), wf)

	// Rollback 干净。
	require.NoError(t, migration202606110002().Rollback(gdb))
	require.Error(t, gdb.Exec(`SELECT 1 FROM group_tasks`).Error)
	require.Error(t, gdb.Exec(`SELECT 1 FROM workflows`).Error)
}
