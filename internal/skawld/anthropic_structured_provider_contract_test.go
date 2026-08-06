package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestAnthropicStructuredProviderContract requires ANTHROPIC_API_KEY,
// ANTHROPIC_MODEL, and ANTHROPIC_MODEL_VERSION; it performs one real
// generation against the Anthropic Messages API. It skips when the
// credentials are absent so CI stays green without them.
func TestAnthropicStructuredProviderContract(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	model := os.Getenv("ANTHROPIC_MODEL")
	version := os.Getenv("ANTHROPIC_MODEL_VERSION")
	if key == "" || model == "" || version == "" {
		t.Skip("Anthropic credentials not configured")
	}
	provider, err := NewAnthropicStructuredProvider(AnthropicStructuredConfig{
		APIKey: key, Model: model, ModelVersion: version,
	}, &http.Client{Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	const evidenceID = "00000000-0000-0000-0000-0000000000bb"
	response, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
		PromptVersion: "contract-v1",
		Context:       json.RawMessage(`{"incident":"P-302 high vibration"}`),
		Evidence: []Evidence{{
			ID: evidenceID, Kind: "document",
			SourceID: "doc-1", Locator: "sop://p-302", Authority: "SITE_APPROVED",
			Content: "Isolate energy before intrusive work.",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Status      string   `json:"status"`
		EvidenceIDs []string `json:"evidence_ids"`
		RiskLevel   string   `json:"risk_level"`
		Confidence  float64  `json:"confidence"`
	}
	if err := json.Unmarshal(response.Output, &output); err != nil {
		t.Fatalf("output is not the recommendation shape: %v", err)
	}
	if output.Status == "" || output.Confidence < 0 || output.Confidence > 1 {
		t.Fatalf("malformed recommendation output: %+v", output)
	}
	seen := make(map[string]bool)
	for _, id := range output.EvidenceIDs {
		if id != evidenceID {
			t.Fatalf("evidence id %q not in the request evidence set", id)
		}
		if seen[id] {
			t.Fatalf("duplicate evidence id %q", id)
		}
		seen[id] = true
	}
	if output.RiskLevel != "INFORMATIONAL" && output.RiskLevel != "ADVISORY" {
		t.Fatalf("risk_level %q outside the closed set accepted by the copilot decoder", output.RiskLevel)
	}
	if response.Metadata.Provider != "anthropic" ||
		response.Metadata.Model == "" || response.Metadata.ModelVersion == "" {
		t.Fatalf("metadata = %+v", response.Metadata)
	}
}
