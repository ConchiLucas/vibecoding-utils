package system

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
	systemModel "github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

const defaultAIConfigKey = "default"

type AIConfigService struct{}

func (s *AIConfigService) CurrentAIConfig() config.AI {
	if global.GVA_DB == nil {
		return normalizePersistentAIConfig(global.GVA_CONFIG.AI)
	}

	aiConfig, err := s.loadAIConfigFromDB()
	if err == nil {
		global.GVA_CONFIG.AI = aiConfig
		return aiConfig
	}

	aiConfig = normalizePersistentAIConfig(global.GVA_CONFIG.AI)
	if hasUsableAIConfig(aiConfig) && errors.Is(err, gorm.ErrRecordNotFound) {
		_ = s.SaveAIConfig(aiConfig)
	}
	return aiConfig
}

func (s *AIConfigService) SaveAIConfig(aiConfig config.AI) error {
	aiConfig = normalizePersistentAIConfig(aiConfig)
	providerBytes, err := json.Marshal(aiConfig.Providers)
	if err != nil {
		return err
	}

	if global.GVA_DB != nil {
		var row systemModel.TbAIConfig
		err = global.GVA_DB.Where("config_key = ?", defaultAIConfigKey).First(&row).Error
		if err == nil {
			row.Active = strings.TrimSpace(aiConfig.Active)
			row.Providers = string(providerBytes)
			if err := global.GVA_DB.Save(&row).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			row = systemModel.TbAIConfig{
				ConfigKey: defaultAIConfigKey,
				Active:    strings.TrimSpace(aiConfig.Active),
				Providers: string(providerBytes),
			}
			if err := global.GVA_DB.Create(&row).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	global.GVA_CONFIG.AI = aiConfig
	return nil
}

func (s *AIConfigService) SaveActiveAIConfig(active string) error {
	active = strings.TrimSpace(active)
	aiConfig := s.CurrentAIConfig()
	if _, err := aiConfig.ResolveProvider(active); err != nil {
		return err
	}
	aiConfig.Active = active
	return s.SaveAIConfig(aiConfig)
}

func (s *AIConfigService) loadAIConfigFromDB() (config.AI, error) {
	var row systemModel.TbAIConfig
	if err := global.GVA_DB.Where("config_key = ?", defaultAIConfigKey).First(&row).Error; err != nil {
		return config.AI{}, err
	}

	var providers map[string]config.AIProvider
	if strings.TrimSpace(row.Providers) != "" {
		if err := json.Unmarshal([]byte(row.Providers), &providers); err != nil {
			return config.AI{}, err
		}
	}
	aiConfig := normalizePersistentAIConfig(config.AI{
		Active:    row.Active,
		Providers: providers,
	})
	return aiConfig, nil
}

func normalizePersistentAIConfig(aiConfig config.AI) config.AI {
	if len(aiConfig.Providers) == 0 && (strings.TrimSpace(aiConfig.BaseURL) != "" || strings.TrimSpace(aiConfig.Model) != "") {
		aiConfig.Providers = map[string]config.AIProvider{
			"default": {
				Label:     "默认模型",
				Type:      config.AIProviderTypeOpenAICompatible,
				BaseURL:   strings.TrimRight(strings.TrimSpace(aiConfig.BaseURL), "/"),
				ApiKey:    strings.TrimSpace(aiConfig.ApiKey),
				Model:     strings.TrimSpace(aiConfig.Model),
				MaxTokens: aiConfig.MaxTokens,
			},
		}
		if strings.TrimSpace(aiConfig.Active) == "" {
			aiConfig.Active = "default"
		}
	}
	if aiConfig.Providers == nil {
		aiConfig.Providers = map[string]config.AIProvider{}
	}
	for id, provider := range aiConfig.Providers {
		provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
		provider.ApiKey = strings.TrimSpace(provider.ApiKey)
		provider.Model = strings.TrimSpace(provider.Model)
		if strings.TrimSpace(provider.Type) == "" {
			provider.Type = config.AIProviderTypeOpenAICompatible
		}
		if provider.MaxTokens <= 0 {
			provider.MaxTokens = 4096
		}
		aiConfig.Providers[id] = provider
	}
	return config.AI{
		Active:    strings.TrimSpace(aiConfig.Active),
		Providers: aiConfig.Providers,
	}
}

func hasUsableAIConfig(aiConfig config.AI) bool {
	return len(aiConfig.Providers) > 0 || strings.TrimSpace(aiConfig.BaseURL) != "" || strings.TrimSpace(aiConfig.Model) != ""
}
