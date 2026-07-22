package initialize

import (
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDropRemovedAIAssistantTablesDropsHistoryOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:removed-ai-assistant?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tb_ai_chat_history (id integer primary key, chat_id text)`).Error; err != nil {
		t.Fatalf("create retired history table: %v", err)
	}
	if err := db.AutoMigrate(&system.TbAIConfig{}); err != nil {
		t.Fatalf("migrate AI config: %v", err)
	}
	wantConfig := system.TbAIConfig{ConfigKey: "default", Active: "primary", Providers: `{"primary":{"model":"test"}}`}
	if err := db.Create(&wantConfig).Error; err != nil {
		t.Fatalf("seed AI config: %v", err)
	}

	if err := dropRemovedAIAssistantTables(db); err != nil {
		t.Fatalf("dropRemovedAIAssistantTables: %v", err)
	}
	if db.Migrator().HasTable("tb_ai_chat_history") {
		t.Fatal("tb_ai_chat_history still exists")
	}
	if !db.Migrator().HasTable(&system.TbAIConfig{}) {
		t.Fatal("tb_ai_config was removed")
	}
	var got system.TbAIConfig
	if err := db.Where("config_key = ?", "default").First(&got).Error; err != nil {
		t.Fatalf("load preserved AI config: %v", err)
	}
	if got.Active != wantConfig.Active || got.Providers != wantConfig.Providers {
		t.Fatalf("AI config changed: got %#v, want %#v", got, wantConfig)
	}
}
