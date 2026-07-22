package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
	"go.uber.org/zap"
)

// AICompletionService provides non-stream completions for retained product
// features such as table-sample data generation.
type AICompletionService struct{}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiCompletionResponseFormat struct {
	Type string `json:"type"`
}

type aiCompletionRequest struct {
	Model          string                      `json:"model"`
	Messages       []AIMessage                 `json:"messages"`
	Stream         bool                        `json:"stream"`
	MaxTokens      int                         `json:"max_tokens,omitempty"`
	ResponseFormat *aiCompletionResponseFormat `json:"response_format,omitempty"`
}

type aiCompletionResponse struct {
	Choices []struct {
		Message AIMessage `json:"message"`
	} `json:"choices"`
}

type aiCompletionAnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiCompletionAnthropicRequest struct {
	Model     string                         `json:"model"`
	System    string                         `json:"system,omitempty"`
	Messages  []aiCompletionAnthropicMessage `json:"messages"`
	Stream    bool                           `json:"stream"`
	MaxTokens int                            `json:"max_tokens"`
}

// CompleteOnce calls the configured provider once and returns its final text.
func (s *AICompletionService) CompleteOnce(messages []AIMessage, providerID string) (string, config.ResolvedAIProvider, error) {
	aiConfig := (&AIConfigService{}).CurrentAIConfig()
	provider, err := aiConfig.ResolveProvider(providerID)
	if err != nil {
		return "", config.ResolvedAIProvider{}, err
	}

	if provider.Type == config.AIProviderTypeAnthropicCompatible {
		content, err := s.completeAnthropicOnce(provider, messages)
		return content, provider, err
	}

	content, err := s.completeOpenAIOnce(provider, messages)
	return content, provider, err
}

// CompleteJSONOnce requests a strict JSON object where the provider supports it.
func (s *AICompletionService) CompleteJSONOnce(messages []AIMessage, providerID string) (string, config.ResolvedAIProvider, error) {
	aiConfig := (&AIConfigService{}).CurrentAIConfig()
	provider, err := aiConfig.ResolveProvider(providerID)
	if err != nil {
		return "", config.ResolvedAIProvider{}, err
	}

	if provider.Type == config.AIProviderTypeAnthropicCompatible {
		content, err := s.completeAnthropicOnce(provider, messages)
		return content, provider, err
	}

	jsonContent, jsonErr := s.completeOpenAIOnceWithResponseFormat(provider, messages, &aiCompletionResponseFormat{Type: "json_object"})
	if jsonErr == nil {
		return jsonContent, provider, nil
	}
	if global.GVA_LOG != nil {
		global.GVA_LOG.Warn("AI JSON 模式请求失败，回退普通完成",
			zap.String("provider", provider.ID),
			zap.String("model", provider.Model),
			zap.Error(jsonErr),
		)
	}

	content, fallbackErr := s.completeOpenAIOnce(provider, messages)
	if fallbackErr != nil {
		return "", provider, fmt.Errorf("JSON 模式请求失败: %v；普通模式也失败: %w", jsonErr, fallbackErr)
	}
	return content, provider, nil
}

func (s *AICompletionService) completeOpenAIOnce(provider config.ResolvedAIProvider, messages []AIMessage) (string, error) {
	return s.completeOpenAIOnceWithResponseFormat(provider, messages, nil)
}

func (s *AICompletionService) completeOpenAIOnceWithResponseFormat(provider config.ResolvedAIProvider, messages []AIMessage, responseFormat *aiCompletionResponseFormat) (string, error) {
	req := aiCompletionRequest{
		Model:          provider.Model,
		Messages:       messages,
		Stream:         false,
		MaxTokens:      normalizedAICompletionMaxTokens(provider),
		ResponseFormat: responseFormat,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("编码 AI 请求失败: %w", err)
	}
	resp, err := s.doRequest(provider, reqBody)
	if err != nil {
		return "", fmt.Errorf("请求 AI 模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var completionResp aiCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("AI 模型没有返回内容")
	}
	content := strings.TrimSpace(completionResp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("AI 模型返回空内容")
	}
	return content, nil
}

func (s *AICompletionService) completeAnthropicOnce(provider config.ResolvedAIProvider, messages []AIMessage) (string, error) {
	systemText, nonSystemMessages := splitAICompletionSystemMessages(messages)
	req := aiCompletionAnthropicRequest{
		Model:     provider.Model,
		System:    systemText,
		Messages:  toAICompletionAnthropicMessages(nonSystemMessages),
		Stream:    false,
		MaxTokens: normalizedAICompletionMaxTokens(provider),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("编码 AI 请求失败: %w", err)
	}
	resp, err := s.doRequest(provider, reqBody)
	if err != nil {
		return "", fmt.Errorf("请求 AI 模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI 模型返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	var builder strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" || block.Type == "" {
			builder.WriteString(block.Text)
		}
	}
	content := strings.TrimSpace(builder.String())
	if content == "" {
		return "", fmt.Errorf("AI 模型返回空内容")
	}
	return content, nil
}

func (s *AICompletionService) doRequest(provider config.ResolvedAIProvider, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, aiCompletionProviderEndpoint(provider), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if provider.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)
		if provider.Type == config.AIProviderTypeAnthropicCompatible {
			req.Header.Set("x-api-key", provider.ApiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	}
	return http.DefaultClient.Do(req)
}

func aiCompletionProviderEndpoint(provider config.ResolvedAIProvider) string {
	if provider.Type == config.AIProviderTypeAnthropicCompatible {
		return appendAICompletionEndpoint(provider.BaseURL, "/v1/messages")
	}
	return appendAICompletionEndpoint(provider.BaseURL, "/v1/chat/completions")
}

func appendAICompletionEndpoint(baseURL string, suffix string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(suffix, "/v1/") {
		return base + strings.TrimPrefix(suffix, "/v1")
	}
	return base + suffix
}

func normalizedAICompletionMaxTokens(provider config.ResolvedAIProvider) int {
	if provider.MaxTokens > 0 {
		return provider.MaxTokens
	}
	return 4096
}

func splitAICompletionSystemMessages(messages []AIMessage) (string, []AIMessage) {
	var systemParts []string
	nonSystemMessages := make([]AIMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			if strings.TrimSpace(message.Content) != "" {
				systemParts = append(systemParts, message.Content)
			}
			continue
		}
		nonSystemMessages = append(nonSystemMessages, message)
	}
	return strings.Join(systemParts, "\n\n"), nonSystemMessages
}

func toAICompletionAnthropicMessages(messages []AIMessage) []aiCompletionAnthropicMessage {
	result := make([]aiCompletionAnthropicMessage, 0, len(messages))
	for _, message := range messages {
		role := message.Role
		if role != "assistant" {
			role = "user"
		}
		result = append(result, aiCompletionAnthropicMessage{Role: role, Content: message.Content})
	}
	return result
}
