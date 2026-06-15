package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigration202606150002GroupCreateToolDefault(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// agents 表 + CEO seed 由 202605220004 建；tools_json 列 + CEO 默认 org 由 202606110001 建。
	require.NoError(t, migration202605220002().Migrate(gdb))
	require.NoError(t, migration202605220003().Migrate(gdb))
	require.NoError(t, migration202605220004().Migrate(gdb))
	require.NoError(t, migration202606110001().Migrate(gdb))

	require.NoError(t, migration202606150002().Migrate(gdb))

	// CEO(DEFAULT) 此后同时带 org 与 group_create(均 enabled)。
	var ceoTools string
	require.NoError(t, gdb.Table("agents").
		Where("system_badge = ?", "DEFAULT").
		Pluck("tools_json", &ceoTools).Error)
	require.JSONEq(t, `[{"key":"org","enabled":true},{"key":"group_create","enabled":true}]`, ceoTools)

	// 非 CEO 行不受影响,仍为默认 '[]'。
	require.NoError(t, gdb.Exec(`INSERT INTO agents (name, department_id, parent_agent_id, agent_backend_id, status) VALUES ('t', 0, 0, 0, 1)`).Error)
	var plain string
	require.NoError(t, gdb.Table("agents").Where("name = 't'").Pluck("tools_json", &plain).Error)
	require.Equal(t, `[]`, plain)

	// Rollback 还原为仅 org。
	require.NoError(t, migration202606150002().Rollback(gdb))
	require.NoError(t, gdb.Table("agents").
		Where("system_badge = ?", "DEFAULT").
		Pluck("tools_json", &ceoTools).Error)
	require.JSONEq(t, `[{"key":"org","enabled":true}]`, ceoTools)
}
