package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigration202606120001AgentSkillsReset(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// agents 表 + CEO seed 由 202605220004 建（依赖 202605220002 departments / 202605220003 backends）
	require.NoError(t, migration202605220002().Migrate(gdb))
	require.NoError(t, migration202605220003().Migrate(gdb))
	require.NoError(t, migration202605220004().Migrate(gdb))

	// 在 agents 表中插入一个带旧语义 skills_json 的行（模拟迁移前状态）
	require.NoError(t, gdb.Exec(
		`INSERT INTO agents (name, department_id, parent_agent_id, agent_backend_id, status, skills_json)
		 VALUES ('test-agent', 0, 0, 0, 1, '[{"label":"old","enabled":true}]')`).Error)

	// 同时给 CEO 也设置旧 skills_json（确保 UPDATE 作用于全部行）
	require.NoError(t, gdb.Exec(
		`UPDATE agents SET skills_json = '[{"label":"old","enabled":true}]' WHERE system_badge = 'DEFAULT'`).Error)

	// 执行本迁移
	require.NoError(t, migration202606120001().Migrate(gdb))

	// 断言所有行的 skills_json 都被重置为 '[]'
	var count int64
	require.NoError(t, gdb.Raw(`SELECT COUNT(*) FROM agents WHERE skills_json != '[]'`).Scan(&count).Error)
	require.Equal(t, int64(0), count, "所有 agents 行的 skills_json 应被重置为 '[]'")

	// 具体验证插入的 test-agent 行
	var v string
	require.NoError(t, gdb.Raw(`SELECT skills_json FROM agents WHERE name = 'test-agent'`).Scan(&v).Error)
	require.Equal(t, "[]", v)
}
