package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606120001 重置 agents.skills_json:旧 {label,enabled} 语义改为
// {id,enabled}(id=plugin id)。旧数据无 UI 入口、未发布,直接清空为 '[]'。
func migration202606120001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606120001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`UPDATE agents SET skills_json = '[]'`).Error
		},
		Rollback: func(tx *gorm.DB) error { return nil },
	}
}
