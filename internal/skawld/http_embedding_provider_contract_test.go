package skawld

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestEmbeddingProviderContract requires EMBEDDING_ENDPOINT, AI_API_KEY,
// EMBEDDING_MODEL, and EMBEDDING_MODEL_VERSION; it embeds two fixed texts
// against the configured deployment adapter. It skips when the credentials
// are absent so CI stays green without them.
func TestEmbeddingProviderContract(t *testing.T) {
	endpoint := os.Getenv("EMBEDDING_ENDPOINT")
	key := os.Getenv("AI_API_KEY")
	model := os.Getenv("EMBEDDING_MODEL")
	version := os.Getenv("EMBEDDING_MODEL_VERSION")
	if endpoint == "" || key == "" || model == "" || version == "" {
		t.Skip("embedding provider credentials not configured")
	}
	provider, err := NewHTTPEmbeddingProvider(HTTPEmbeddingConfig{
		Endpoint: endpoint, APIKey: key, Provider: "openai",
		Model: model, ModelVersion: version,
	}, &http.Client{Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := provider.Embed(context.Background(), []string{
		"vibration 8.1 mm/s on P-302 motor bearing",
		"temperature 94 celsius",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 {
		t.Fatalf("vector count = %d, want 2", len(vectors))
	}
	if len(vectors[0]) == 0 || len(vectors[0]) != len(vectors[1]) {
		t.Fatalf("inconsistent dimensions: %d vs %d", len(vectors[0]), len(vectors[1]))
	}
	if provider.Model().Metric != "COSINE" {
		t.Fatalf("metric = %q, want COSINE", provider.Model().Metric)
	}
}
