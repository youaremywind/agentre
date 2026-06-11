package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigration202606110001AddsAgentTools(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// agents 表 + CEO seed 由 202605220004 建（依赖 202605220002 departments / 202605220003 backends）
	require.NoError(t, migration202605220002().Migrate(gdb))
	require.NoError(t, migration202605220003().Migrate(gdb))
	require.NoError(t, migration202605220004().Migrate(gdb))

	require.NoError(t, migration202606110001().Migrate(gdb))

	var ceoTools string
	require.NoError(t, gdb.Table("agents").
		Where("system_badge = ?", "DEFAULT").
		Pluck("tools_json", &ceoTools).Error)
	require.JSONEq(t, `[{"key":"org","enabled":true}]`, ceoTools)

	// 非 CEO 行默认 '[]'
	require.NoError(t, gdb.Exec(`INSERT INTO agents (name, department_id, parent_agent_id, agent_backend_id, status) VALUES ('t', 0, 0, 0, 1)`).Error)
	var plain string
	require.NoError(t, gdb.Table("agents").Where("name = 't'").Pluck("tools_json", &plain).Error)
	require.Equal(t, `[]`, plain)
}
