package skawld

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBuildProvidersDeterministicDefaults(t *testing.T) {
	t.Parallel()
	structuredProviders, embedding, err := BuildProviders(AIConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []Capability{
		CapabilityRecommendation, CapabilityReportDraft, CapabilityShiftHandover,
	} {
		if _, ok := structuredProviders[capability]; !ok {
			t.Fatalf("missing provider for %q", capability)
		}
	}
	if _, ok := embedding.(DeterministicEmbeddingProvider); !ok {
		t.Fatalf("embedding provider = %T, want DeterministicEmbeddingProvider", embedding)
	}
	router := Router{Providers: structuredProviders}
	generation, err := router.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(generation.Output, &output); err != nil {
		t.Fatal(err)
	}
	if output["status"] != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("deterministic output = %+v", output)
	}
}

func TestBuildProvidersOpenAI(t *testing.T) {
	t.Parallel()
	config := AIConfig{
		StructuredProvider:     "openai",
		EmbeddingProvider:      "openai",
		StructuredEndpoint:     "https://example.test/structured",
		StructuredAPIKey:       "secret",
		StructuredModel:        "gpt-4o",
		StructuredModelVersion: "2026-08",
		EmbeddingEndpoint:      "https://example.test/embeddings",
		EmbeddingModel:         "text-embedding-3-small",
		EmbeddingModelVersion:  "2026-08",
	}
	structuredProviders, embedding, err := BuildProviders(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := structuredProviders[CapabilityRecommendation].(*HTTPStructuredProvider); !ok {
		t.Fatalf("structured provider = %T, want *HTTPStructuredProvider",
			structuredProviders[CapabilityRecommendation])
	}
	if _, ok := embedding.(*HTTPEmbeddingProvider); !ok {
		t.Fatalf("embedding provider = %T, want *HTTPEmbeddingProvider", embedding)
	}
}

func TestBuildProvidersAnthropic(t *testing.T) {
	t.Parallel()
	config := AIConfig{
		StructuredProvider:    "anthropic",
		AnthropicAPIKey:       "secret",
		AnthropicModel:        "claude-sonnet-4",
		AnthropicModelVersion: "2026-08",
	}
	structuredProviders, _, err := BuildProviders(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := structuredProviders[CapabilityRecommendation].(*AnthropicStructuredProvider); !ok {
		t.Fatalf("structured provider = %T, want *AnthropicStructuredProvider",
			structuredProviders[CapabilityRecommendation])
	}
}

func TestBuildProvidersRejectsUnknown(t *testing.T) {
	t.Parallel()
	if _, _, err := BuildProviders(AIConfig{StructuredProvider: "bogus"}, nil); err == nil {
		t.Fatal("expected unknown structured provider error")
	}
	if _, _, err := BuildProviders(AIConfig{EmbeddingProvider: "bogus"}, nil); err == nil {
		t.Fatal("expected unknown embedding provider error")
	}
}

func TestBuildProvidersOpenAIWithoutModel(t *testing.T) {
	t.Parallel()
	config := AIConfig{
		StructuredProvider: "openai",
		StructuredEndpoint: "https://example.test",
	}
	if _, _, err := BuildProviders(config, nil); err == nil {
		t.Fatal("expected missing model error")
	}
}
