package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606110002 群任务卡编排基线:group_tasks / workflows 两张新表 +
// group_messages.task_id/task_event + groups.workflow_id。
// 同一特性的 schema 合并为一个迁移(先例:202606030001 群聊基线)。
func migration202606110002() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606110002",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS group_tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	group_id INTEGER NOT NULL DEFAULT 0,
	task_no INTEGER NOT NULL DEFAULT 0,
	title TEXT NOT NULL DEFAULT '',
	brief TEXT NOT NULL DEFAULT '',
	creator_member_id INTEGER NOT NULL DEFAULT 0,
	assignee_member_id INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'open',
	result TEXT NOT NULL DEFAULT '',
	parent_task_no INTEGER NOT NULL DEFAULT 0,
	createtime INTEGER NOT NULL DEFAULT 0,
	updatetime INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_group_tasks_group ON group_tasks(group_id, status, task_no)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_group_tasks_group_no ON group_tasks(group_id, task_no)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE group_messages ADD COLUMN task_id INTEGER NOT NULL DEFAULT 0`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE group_messages ADD COLUMN task_event TEXT NOT NULL DEFAULT ''`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS workflows (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL DEFAULT '',
	status INTEGER NOT NULL DEFAULT 1,
	createtime INTEGER NOT NULL DEFAULT 0,
	updatetime INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
				return err
			}
			return tx.Exec(`ALTER TABLE groups ADD COLUMN workflow_id INTEGER NOT NULL DEFAULT 0`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Exec(`ALTER TABLE groups DROP COLUMN workflow_id`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP TABLE IF EXISTS workflows`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE group_messages DROP COLUMN task_event`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE group_messages DROP COLUMN task_id`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP INDEX IF EXISTS uniq_group_tasks_group_no`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`DROP INDEX IF EXISTS idx_group_tasks_group`).Error; err != nil {
				return err
			}
			return tx.Exec(`DROP TABLE IF EXISTS group_tasks`).Error
		},
	}
}
