package initialize

import "gorm.io/gorm"

func dropRemovedAIAssistantTables(db *gorm.DB) error {
	return db.Exec("DROP TABLE IF EXISTS tb_ai_chat_history").Error
}
