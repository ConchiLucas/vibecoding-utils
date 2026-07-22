package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/config"
	"github.com/flipped-aurora/easy-deploy/server/global"
)

func TestAICompletionServiceCompleteJSONOnceUsesConfiguredProvider(t *testing.T) {
	oldDB := global.GVA_DB
	oldAI := global.GVA_CONFIG.AI
	t.Cleanup(func() {
		global.GVA_DB = oldDB
		global.GVA_CONFIG.AI = oldAI
	})
	global.GVA_DB = nil

	type capturedRequest struct {
		Path string
		Body map[string]any
	}
	captured := make(chan capturedRequest, 1)
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		captured <- capturedRequest{Path: r.URL.Path, Body: body}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"rows\":[]}"}}]}`))
	}))
	t.Cleanup(providerServer.Close)

	global.GVA_CONFIG.AI = config.AI{
		Active: "test-provider",
		Providers: map[string]config.AIProvider{
			"test-provider": {
				Label:     "Test Provider",
				Type:      config.AIProviderTypeOpenAICompatible,
				BaseURL:   providerServer.URL,
				ApiKey:    "test-key",
				Model:     "test-model",
				MaxTokens: 512,
			},
		},
	}

	content, provider, err := (&AICompletionService{}).CompleteJSONOnce([]AIMessage{
		{Role: "system", Content: "Return JSON."},
		{Role: "user", Content: "Generate rows."},
	}, "")
	if err != nil {
		t.Fatalf("CompleteJSONOnce returned error: %v", err)
	}
	if content != `{"rows":[]}` {
		t.Fatalf("content = %q, want JSON rows", content)
	}
	if provider.ID != "test-provider" {
		t.Fatalf("provider ID = %q, want test-provider", provider.ID)
	}

	request := <-captured
	if request.Path != "/v1/chat/completions" {
		t.Fatalf("request path = %q, want /v1/chat/completions", request.Path)
	}
	responseFormat, ok := request.Body["response_format"].(map[string]any)
	if !ok || responseFormat["type"] != "json_object" {
		t.Fatalf("response_format = %#v, want json_object", request.Body["response_format"])
	}
	messages, ok := request.Body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want two messages", request.Body["messages"])
	}
	userMessage, ok := messages[1].(map[string]any)
	if !ok || userMessage["role"] != "user" || userMessage["content"] != "Generate rows." {
		t.Fatalf("user message = %#v", messages[1])
	}
}
