package system

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AIConfigApi struct{}

type AIProviderConfigItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url"`
	ApiKey    string `json:"api_key"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Active    bool   `json:"active"`
}

type AIConfigRequest struct {
	Active    string                 `json:"active"`
	Providers []AIProviderConfigItem `json:"providers"`
}

type AIActiveConfigRequest struct {
	Active string `json:"active"`
}

func (a *AIConfigApi) GetConfig(c *gin.Context) {
	aiConfig := aiConfigService.CurrentAIConfig()
	listProviders := aiConfig.ListProviders()
	providers := make([]AIProviderConfigItem, 0, len(listProviders))
	for _, provider := range listProviders {
		raw := aiConfig.Providers[provider.ID]
		providers = append(providers, AIProviderConfigItem{
			ID:        provider.ID,
			Label:     provider.Label,
			Type:      provider.Type,
			BaseURL:   provider.BaseURL,
			ApiKey:    raw.ApiKey,
			Model:     provider.Model,
			MaxTokens: provider.MaxTokens,
			Active:    provider.Active,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": AIConfigRequest{
			Active:    effectiveAIProviderID(listProviders, aiConfig.Active),
			Providers: providers,
		},
		"msg": "获取成功",
	})
}

func (a *AIConfigApi) SaveConfig(c *gin.Context) {
	var req AIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responseError(c, "参数错误: "+err.Error())
		return
	}

	aiConfig, err := normalizeAIConfig(req)
	if err != nil {
		responseError(c, err.Error())
		return
	}

	if err := aiConfigService.SaveAIConfig(aiConfig); err != nil {
		global.GVA_LOG.Error("保存 AI 配置失败", zap.Error(err))
		responseError(c, "保存失败: "+err.Error())
		return
	}

	c.JSON(200, gin.H{"code": 0, "data": aiConfig.ListProviders(), "msg": "保存成功"})
}

func (a *AIConfigApi) SaveActiveConfig(c *gin.Context) {
	var req AIActiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responseError(c, "参数错误: "+err.Error())
		return
	}

	active := strings.TrimSpace(req.Active)
	if active == "" {
		responseError(c, "请选择默认 AI 配置")
		return
	}
	if err := aiConfigService.SaveActiveAIConfig(active); err != nil {
		global.GVA_LOG.Error("保存默认 AI 配置失败", zap.Error(err))
		responseError(c, "保存失败: "+err.Error())
		return
	}

	aiConfig := aiConfigService.CurrentAIConfig()
	c.JSON(200, gin.H{"code": 0, "data": aiConfig.ListProviders(), "msg": "保存成功"})
}

func normalizeAIConfig(req AIConfigRequest) (config.AI, error) {
	providers := make(map[string]config.AIProvider, len(req.Providers))
	active := strings.TrimSpace(req.Active)

	for _, item := range req.Providers {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return config.AI{}, fmt.Errorf("请填写 AI 配置 ID")
		}
		if _, exists := providers[id]; exists {
			return config.AI{}, fmt.Errorf("AI 配置 ID「%s」重复", id)
		}
		providerType := strings.TrimSpace(item.Type)
		if providerType == "" {
			providerType = config.AIProviderTypeOpenAICompatible
		}
		if providerType != config.AIProviderTypeOpenAICompatible && providerType != config.AIProviderTypeAnthropicCompatible {
			return config.AI{}, fmt.Errorf("AI 配置「%s」的类型不支持", id)
		}
		if strings.TrimSpace(item.BaseURL) == "" {
			return config.AI{}, fmt.Errorf("请填写 AI 配置「%s」的 Base URL", id)
		}
		if strings.TrimSpace(item.Model) == "" {
			return config.AI{}, fmt.Errorf("请填写 AI 配置「%s」的模型名称", id)
		}
		maxTokens := item.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 4096
		}
		providers[id] = config.AIProvider{
			Label:     strings.TrimSpace(item.Label),
			Type:      providerType,
			BaseURL:   strings.TrimRight(strings.TrimSpace(item.BaseURL), "/"),
			ApiKey:    strings.TrimSpace(item.ApiKey),
			Model:     strings.TrimSpace(item.Model),
			MaxTokens: maxTokens,
		}
	}

	if len(providers) == 0 {
		return config.AI{}, fmt.Errorf("请至少保留一个 AI 配置")
	}
	if active == "" {
		active = firstAIProviderID(providers)
	}
	if _, exists := providers[active]; !exists {
		return config.AI{}, fmt.Errorf("默认 AI 配置「%s」不存在", active)
	}

	return config.AI{Active: active, Providers: providers}, nil
}

func firstAIProviderID(providers map[string]config.AIProvider) string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func effectiveAIProviderID(providers []config.ResolvedAIProvider, fallback string) string {
	for _, provider := range providers {
		if provider.Active {
			return provider.ID
		}
	}
	return strings.TrimSpace(fallback)
}

func responseError(c *gin.Context, msg string) {
	c.JSON(200, gin.H{"code": 7, "msg": msg})
}
