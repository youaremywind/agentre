package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606110001 adds agents.tools_json — agent 级内置工具开关
// （首个工具 key="org"，组织架构读写）。CEO(system_badge=DEFAULT) 默认开启 org。
func migration202606110001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606110001",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`ALTER TABLE agents ADD COLUMN tools_json TEXT NOT NULL DEFAULT '[]'`).Error; err != nil {
				return err
			}
			return tx.Exec(`UPDATE agents SET tools_json = '[{"key":"org","enabled":true}]' WHERE system_badge = 'DEFAULT'`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE agents DROP COLUMN tools_json`).Error
		},
	}
}
