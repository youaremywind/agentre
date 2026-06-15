package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606150001 给 group_members 加 nickname 列(群昵称 / 群内备注名)。
// 非空时即该成员在本群的「有效显示名」(roster / @mention / AI 群名单);为空回落 agent 全局名。
func migration202606150001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606150001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE group_members ADD COLUMN nickname TEXT NOT NULL DEFAULT ''`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE group_members DROP COLUMN nickname`).Error
		},
	}
}
