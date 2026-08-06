package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestStructuredProviderContract requires AI_ENDPOINT, AI_API_KEY, AI_MODEL,
// and AI_MODEL_VERSION; it performs one real generation against the
// configured deployment adapter. It skips when the credentials are absent so
// CI stays green without them.
func TestStructuredProviderContract(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	key := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")
	version := os.Getenv("AI_MODEL_VERSION")
	if endpoint == "" || key == "" || model == "" || version == "" {
		t.Skip("AI provider credentials not configured")
	}
	provider, err := NewHTTPStructuredProvider(HTTPStructuredConfig{
		Endpoint: endpoint, APIKey: key, Provider: "openai",
		Model: model, ModelVersion: version,
	}, &http.Client{Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	const evidenceID = "00000000-0000-0000-0000-0000000000aa"
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
	for _, id := range output.EvidenceIDs {
		if id != evidenceID {
			t.Fatalf("evidence id %q not in the request evidence set", id)
		}
	}
	if output.RiskLevel != "INFORMATIONAL" && output.RiskLevel != "ADVISORY" {
		t.Fatalf("risk_level %q outside the closed set accepted by the copilot decoder", output.RiskLevel)
	}
	if response.Metadata.Provider == "" || response.Metadata.Model == "" ||
		response.Metadata.ModelVersion == "" {
		t.Fatalf("metadata = %+v", response.Metadata)
	}
}
