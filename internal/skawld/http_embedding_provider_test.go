package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestEmbeddingProvider(t *testing.T, handler http.HandlerFunc) *HTTPEmbeddingProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider, err := NewHTTPEmbeddingProvider(HTTPEmbeddingConfig{
		Endpoint: server.URL, APIKey: "secret",
		Provider: "openai", Model: "text-embedding-3-small", ModelVersion: "2026-08",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestHTTPEmbeddingProviderEmbed(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing provider authorization")
		}
		var request struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Model != "text-embedding-3-small" {
			t.Errorf("model = %q", request.Model)
		}
		if len(request.Input) != 2 {
			t.Errorf("input count = %d, want 2", len(request.Input))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{"embedding": [0.1, 0.2, 0.3]},
				{"embedding": [0.4, 0.5, 0.6]}
			],
			"model": "text-embedding-3-small"
		}`))
	})
	vectors, err := provider.Embed(context.Background(), []string{"vibration", "temperature"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 {
		t.Fatalf("vector count = %d, want 2", len(vectors))
	}
	if len(vectors[0]) != 3 || len(vectors[1]) != 3 {
		t.Fatalf("vector dimensions = %d/%d, want 3", len(vectors[0]), len(vectors[1]))
	}
	model := provider.Model()
	if model.Provider != "openai" || model.Model != "text-embedding-3-small" ||
		model.Metric != "COSINE" {
		t.Fatalf("model = %+v", model)
	}
}

func TestHTTPEmbeddingProviderRejectsEmptyInput(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected request for empty input")
	})
	if _, err := provider.Embed(context.Background(), []string{}); err == nil {
		t.Fatal("expected empty input to be rejected")
	}
}

func TestHTTPEmbeddingProviderRejectsInconsistentDimensions(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{"embedding": [0.1, 0.2, 0.3]},
				{"embedding": [0.4, 0.5]}
			],
			"model": "text-embedding-3-small"
		}`))
	})
	if _, err := provider.Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected inconsistent dimensions to be rejected")
	}
}

func TestHTTPEmbeddingProviderConfigValidation(t *testing.T) {
	t.Parallel()
	cases := []HTTPEmbeddingConfig{
		{Endpoint: "not-a-url", APIKey: "k", Provider: "openai", Model: "m", ModelVersion: "v"},
		{Endpoint: "https://example.test", APIKey: "k", Provider: "openai", Model: "", ModelVersion: "v"},
	}
	for _, config := range cases {
		if _, err := NewHTTPEmbeddingProvider(config, &http.Client{}); err == nil {
			t.Fatalf("expected constructor error for %+v", config)
		}
	}
	if _, err := NewHTTPEmbeddingProvider(HTTPEmbeddingConfig{
		Endpoint: "https://example.test", APIKey: "k",
		Provider: "openai", Model: "m", ModelVersion: "v",
	}, nil); err == nil {
		t.Fatal("expected nil-client error")
	}
}
