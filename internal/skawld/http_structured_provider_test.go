package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestStructuredProvider(t *testing.T, handler http.HandlerFunc) *HTTPStructuredProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider, err := NewHTTPStructuredProvider(HTTPStructuredConfig{
		Endpoint: server.URL, APIKey: "secret",
		Provider: "openai", Model: "gpt-4o", ModelVersion: "2026-08",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestHTTPStructuredProviderGenerate(t *testing.T) {
	t.Parallel()
	provider := newTestStructuredProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing provider authorization")
		}
		var request struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Model != "gpt-4o" {
			t.Errorf("model = %q, want gpt-4o", request.Model)
		}
		if request.ResponseFormat.Type != "json_object" {
			t.Errorf("response_format.type = %q, want json_object", request.ResponseFormat.Type)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"choices": [{"message": {"content": "{\"status\":\"INFORMATIONAL\"}"}}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 2}
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
	if response.Metadata.Provider != "openai" ||
		response.Metadata.Model != "gpt-4o" ||
		response.Metadata.TokensIn != 5 || response.Metadata.TokensOut != 2 {
		t.Fatalf("metadata = %+v", response.Metadata)
	}
}

func TestHTTPStructuredProviderRejectsEmptyContent(t *testing.T) {
	t.Parallel()
	provider := newTestStructuredProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices": [{"message": {"content": ""}}]}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); err == nil {
		t.Fatal("expected empty content to be rejected")
	}
}

func TestHTTPStructuredProviderRejectsNonJSON(t *testing.T) {
	t.Parallel()
	provider := newTestStructuredProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices": [{"message": {"content": "not json"}}]}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); err == nil {
		t.Fatal("expected non-JSON content to be rejected")
	}
}

func TestHTTPStructuredProviderRejectsNon2xx(t *testing.T) {
	t.Parallel()
	provider := newTestStructuredProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusInternalServerError)
	})
	_, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error = %v, want status 500", err)
	}
}

func TestHTTPStructuredProviderConfigValidation(t *testing.T) {
	t.Parallel()
	client := &http.Client{Timeout: time.Second}
	cases := []HTTPStructuredConfig{
		{Endpoint: "not-a-url", APIKey: "k", Provider: "openai", Model: "m", ModelVersion: "v"},
		{Endpoint: "https://example.test", APIKey: "k", Provider: "", Model: "m", ModelVersion: "v"},
		{Endpoint: "https://example.test", APIKey: "k", Provider: "openai", Model: "", ModelVersion: "v"},
	}
	for _, config := range cases {
		if _, err := NewHTTPStructuredProvider(config, client); err == nil {
			t.Fatalf("expected constructor error for %+v", config)
		}
	}
	if _, err := NewHTTPStructuredProvider(HTTPStructuredConfig{
		Endpoint: "https://example.test", APIKey: "k",
		Provider: "openai", Model: "m", ModelVersion: "v",
	}, nil); err == nil {
		t.Fatal("expected nil-client error")
	}
}
