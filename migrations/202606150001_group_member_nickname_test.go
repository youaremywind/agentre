package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigration202606150001GroupMemberNickname(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 前置:群聊基线建 group_members 依赖 agents + chat_sessions 表。
	require.NoError(t, migration202605220004().Migrate(gdb))
	require.NoError(t, migration202605220006().Migrate(gdb))
	require.NoError(t, migration202606030001().Migrate(gdb))
	require.NoError(t, migration202606150001().Migrate(gdb))

	// nickname 默认空串。
	require.NoError(t, gdb.Exec(`INSERT INTO group_members (group_id, agent_id, backing_session_id) VALUES (1, 2, 3)`).Error)
	var nick string
	require.NoError(t, gdb.Table("group_members").Select("nickname").Where("group_id = 1").Scan(&nick).Error)
	require.Equal(t, "", nick)

	// 可写入群昵称。
	require.NoError(t, gdb.Exec(`UPDATE group_members SET nickname = '前端工程师' WHERE group_id = 1`).Error)
	require.NoError(t, gdb.Table("group_members").Select("nickname").Where("group_id = 1").Scan(&nick).Error)
	require.Equal(t, "前端工程师", nick)

	// Rollback 干净:列消失。
	require.NoError(t, migration202606150001().Rollback(gdb))
	require.Error(t, gdb.Exec(`SELECT nickname FROM group_members`).Error)
}
