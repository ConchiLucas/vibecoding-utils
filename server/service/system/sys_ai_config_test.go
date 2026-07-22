package system

import (
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
	systemModel "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAIConfigServicePersistsAndSwitchesActiveProvider(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ai-config-service?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&systemModel.TbAIConfig{}); err != nil {
		t.Fatalf("migrate AI config: %v", err)
	}

	oldDB := global.GVA_DB
	oldAI := global.GVA_CONFIG.AI
	t.Cleanup(func() {
		global.GVA_DB = oldDB
		global.GVA_CONFIG.AI = oldAI
	})
	global.GVA_DB = db
	global.GVA_CONFIG.AI = config.AI{}

	want := config.AI{
		Active: "primary",
		Providers: map[string]config.AIProvider{
			"primary": {
				Label:     "Primary",
				Type:      config.AIProviderTypeOpenAICompatible,
				BaseURL:   "http://primary.example/v1/",
				ApiKey:    "primary-key",
				Model:     "primary-model",
				MaxTokens: 2048,
			},
			"secondary": {
				Label:     "Secondary",
				Type:      config.AIProviderTypeAnthropicCompatible,
				BaseURL:   "http://secondary.example/",
				ApiKey:    "secondary-key",
				Model:     "secondary-model",
				MaxTokens: 4096,
			},
		},
	}

	service := &AIConfigService{}
	if err := service.SaveAIConfig(want); err != nil {
		t.Fatalf("SaveAIConfig: %v", err)
	}
	loaded := service.CurrentAIConfig()
	if loaded.Active != "primary" || len(loaded.Providers) != 2 {
		t.Fatalf("loaded config = %#v", loaded)
	}
	if loaded.Providers["primary"].BaseURL != "http://primary.example/v1" {
		t.Fatalf("primary base URL was not normalized: %q", loaded.Providers["primary"].BaseURL)
	}

	if err := service.SaveActiveAIConfig("secondary"); err != nil {
		t.Fatalf("SaveActiveAIConfig: %v", err)
	}
	reloaded := service.CurrentAIConfig()
	if reloaded.Active != "secondary" {
		t.Fatalf("active provider = %q, want secondary", reloaded.Active)
	}
	if len(reloaded.Providers) != 2 || reloaded.Providers["primary"].ApiKey != "primary-key" || reloaded.Providers["secondary"].ApiKey != "secondary-key" {
		t.Fatalf("providers changed while switching active provider: %#v", reloaded.Providers)
	}
}
