package config

import (
	"fmt"
	"sort"
	"strings"
)

const (
	AIProviderTypeOpenAICompatible    = "openai-compatible"
	AIProviderTypeAnthropicCompatible = "anthropic-compatible"
)

// AI 大模型配置
type AI struct {
	Active    string                `mapstructure:"active" json:"active" yaml:"active"`             // 默认启用的 provider
	Providers map[string]AIProvider `mapstructure:"providers" json:"providers" yaml:"providers"`    // 可切换的厂商配置
	BaseURL   string                `mapstructure:"base-url" json:"base-url" yaml:"base-url"`       // 兼容旧配置
	ApiKey    string                `mapstructure:"api-key" json:"api-key" yaml:"api-key"`          // 兼容旧配置
	Model     string                `mapstructure:"model" json:"model" yaml:"model"`                // 兼容旧配置
	MaxTokens int                   `mapstructure:"max-tokens" json:"max-tokens" yaml:"max-tokens"` // 兼容旧配置
}

type AIProvider struct {
	Label     string `mapstructure:"label" json:"label" yaml:"label"`
	Type      string `mapstructure:"type" json:"type" yaml:"type"`
	BaseURL   string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`
	ApiKey    string `mapstructure:"api-key" json:"api-key" yaml:"api-key"`
	Model     string `mapstructure:"model" json:"model" yaml:"model"`
	MaxTokens int    `mapstructure:"max-tokens" json:"max-tokens" yaml:"max-tokens"`
}

type ResolvedAIProvider struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url"`
	ApiKey    string `json:"-"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Active    bool   `json:"active"`
}

func (a AI) ResolveProvider(id string) (ResolvedAIProvider, error) {
	if len(a.Providers) == 0 {
		return a.resolveLegacyProvider()
	}

	providerID := strings.TrimSpace(id)
	if providerID == "" {
		providerID = strings.TrimSpace(a.Active)
	}
	if providerID == "" {
		if _, ok := a.Providers["omlx"]; ok {
			providerID = "omlx"
		} else {
			providerID = firstProviderID(a.Providers)
		}
	}

	provider, ok := a.Providers[providerID]
	if !ok {
		return ResolvedAIProvider{}, fmt.Errorf("AI provider %q 不存在", providerID)
	}

	return resolveAIProvider(providerID, provider, providerID == activeProviderID(a)), nil
}

func (a AI) ListProviders() []ResolvedAIProvider {
	if len(a.Providers) == 0 {
		provider, err := a.resolveLegacyProvider()
		if err != nil {
			return nil
		}
		return []ResolvedAIProvider{provider}
	}

	ids := make([]string, 0, len(a.Providers))
	for id := range a.Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	providers := make([]ResolvedAIProvider, 0, len(ids))
	activeID := activeProviderID(a)
	for _, id := range ids {
		providers = append(providers, resolveAIProvider(id, a.Providers[id], id == activeID))
	}
	return providers
}

func (a AI) resolveLegacyProvider() (ResolvedAIProvider, error) {
	if strings.TrimSpace(a.BaseURL) == "" && strings.TrimSpace(a.Model) == "" {
		return ResolvedAIProvider{}, fmt.Errorf("AI provider 未配置")
	}
	return resolveAIProvider("default", AIProvider{
		Label:     "默认模型",
		Type:      AIProviderTypeOpenAICompatible,
		BaseURL:   a.BaseURL,
		ApiKey:    a.ApiKey,
		Model:     a.Model,
		MaxTokens: a.MaxTokens,
	}, true), nil
}

func activeProviderID(a AI) string {
	if strings.TrimSpace(a.Active) != "" {
		return strings.TrimSpace(a.Active)
	}
	if _, ok := a.Providers["omlx"]; ok {
		return "omlx"
	}
	return firstProviderID(a.Providers)
}

func firstProviderID(providers map[string]AIProvider) string {
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

func resolveAIProvider(id string, provider AIProvider, active bool) ResolvedAIProvider {
	providerType := strings.TrimSpace(provider.Type)
	if providerType == "" {
		providerType = AIProviderTypeOpenAICompatible
	}
	label := strings.TrimSpace(provider.Label)
	if label == "" {
		label = id
	}
	return ResolvedAIProvider{
		ID:        id,
		Label:     label,
		Type:      providerType,
		BaseURL:   strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/"),
		ApiKey:    strings.TrimSpace(provider.ApiKey),
		Model:     strings.TrimSpace(provider.Model),
		MaxTokens: provider.MaxTokens,
		Active:    active,
	}
}
