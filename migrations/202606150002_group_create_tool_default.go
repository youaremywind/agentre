package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606150002 把 CEO(system_badge=DEFAULT) 的内置工具默认补上 group_create
// (拉群带流程)。基线由 202606110001 固定为仅 org,故直接覆写为固定两项数组。
// 其余 agent 默认不带 group_create,需在设置里手动开启(per-agent 门控,镜像 org)。
func migration202606150002() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606150002",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`UPDATE agents SET tools_json = '[{"key":"org","enabled":true},{"key":"group_create","enabled":true}]' WHERE system_badge = 'DEFAULT'`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`UPDATE agents SET tools_json = '[{"key":"org","enabled":true}]' WHERE system_badge = 'DEFAULT'`).Error
		},
	}
}
