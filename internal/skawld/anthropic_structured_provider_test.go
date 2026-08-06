package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestAnthropicProvider(t *testing.T, handler http.HandlerFunc) *AnthropicStructuredProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	original := anthropicMessagesEndpoint
	anthropicMessagesEndpoint = server.URL
	t.Cleanup(func() { anthropicMessagesEndpoint = original })
	provider, err := NewAnthropicStructuredProvider(AnthropicStructuredConfig{
		APIKey: "secret", Model: "claude-sonnet-4", ModelVersion: "2026-08",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestAnthropicStructuredProviderGenerate(t *testing.T) {
	provider := newTestAnthropicProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "secret" {
			t.Error("missing provider api key")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}
		var request struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			Messages  []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Model != "claude-sonnet-4" {
			t.Errorf("model = %q", request.Model)
		}
		if request.MaxTokens <= 0 {
			t.Errorf("max_tokens = %d", request.MaxTokens)
		}
		if len(request.Messages) != 1 || request.Messages[0].Role != "user" {
			t.Errorf("messages = %+v", request.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"content": [{"type": "text", "text": "{\"status\":\"INFORMATIONAL\"}"}],
			"usage": {"input_tokens": 5, "output_tokens": 2}
		}`))
	})
	response, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
		PromptVersion: "p1", Context: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(response.Output, &output); err != nil {
		t.Fatal(err)
	}
	if output["status"] != "INFORMATIONAL" {
		t.Fatalf("output = %+v", output)
	}
	if response.Metadata.Provider != "anthropic" ||
		response.Metadata.Model != "claude-sonnet-4" ||
		response.Metadata.TokensIn != 5 || response.Metadata.TokensOut != 2 {
		t.Fatalf("metadata = %+v", response.Metadata)
	}
}

func TestAnthropicStructuredProviderRejectsEmptyContent(t *testing.T) {
	provider := newTestAnthropicProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": [{"type": "text", "text": ""}]}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); err == nil {
		t.Fatal("expected empty content to be rejected")
	}
}

func TestAnthropicStructuredProviderRejectsNonJSON(t *testing.T) {
	provider := newTestAnthropicProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": [{"type": "text", "text": "not json"}]}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); err == nil {
		t.Fatal("expected non-JSON text to be rejected")
	}
}

func TestAnthropicStructuredProviderRejectsNon2xx(t *testing.T) {
	provider := newTestAnthropicProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusTooManyRequests)
	})
	_, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 429") {
		t.Fatalf("error = %v, want status 429", err)
	}
}

func TestAnthropicStructuredProviderConfigValidation(t *testing.T) {
	cases := []AnthropicStructuredConfig{
		{APIKey: "", Model: "m", ModelVersion: "v"},
		{APIKey: "k", Model: "", ModelVersion: "v"},
		{APIKey: "k", Model: "m", ModelVersion: ""},
	}
	for _, config := range cases {
		if _, err := NewAnthropicStructuredProvider(config, &http.Client{}); err == nil {
			t.Fatalf("expected constructor error for %+v", config)
		}
	}
	if _, err := NewAnthropicStructuredProvider(AnthropicStructuredConfig{
		APIKey: "k", Model: "m", ModelVersion: "v",
	}, nil); err == nil {
		t.Fatal("expected nil-client error")
	}
}
