package skawld

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	DevelopmentProviderName = "skawld-development"
	DevelopmentModelName    = "deterministic-structured"
	DevelopmentModelVersion = "v1"
)

type DeterministicProvider struct{}

func (DeterministicProvider) Generate(
	_ context.Context,
	request GenerateRequest,
) (GenerateResponse, error) {
	var output any
	switch request.Capability {
	case CapabilityRecommendation:
		output = deterministicRecommendation(request.Evidence)
	case CapabilityReportDraft:
		output = deterministicReport(request.Context, request.Evidence)
	case CapabilityShiftHandover:
		output = deterministicHandover(request.Context, request.Evidence)
	default:
		return GenerateResponse{}, ErrProviderUnavailable
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		return GenerateResponse{}, err
	}
	return GenerateResponse{
		Output: encoded,
		Metadata: ProviderMetadata{
			Provider: DevelopmentProviderName, Model: DevelopmentModelName,
			ModelVersion: DevelopmentModelVersion,
		},
	}, nil
}

func deterministicRecommendation(evidence []Evidence) map[string]any {
	if len(evidence) == 0 {
		return map[string]any{
			"status":                      "INSUFFICIENT_EVIDENCE",
			"recommendation":              "",
			"evidence_ids":                []string{},
			"assumptions":                 []string{},
			"unknowns":                    []string{"No eligible approved evidence was retrieved."},
			"confidence":                  0,
			"risk_level":                  "INFORMATIONAL",
			"requires_human_confirmation": true,
		}
	}
	combined := strings.ToLower(joinEvidence(evidence))
	recommendation := "Review the cited current procedure and perform its next non-intrusive inspection step."
	if strings.Contains(combined, "lubric") {
		recommendation = "Inspect lubrication condition before considering intrusive work."
	} else if strings.Contains(combined, "alignment") {
		recommendation = "Inspect pump and motor alignment indicators before considering intrusive work."
	}
	ids := make([]string, 0, min(3, len(evidence)))
	for _, item := range evidence[:min(3, len(evidence))] {
		ids = append(ids, item.ID)
	}
	return map[string]any{
		"status":         "RECOMMENDATION",
		"recommendation": recommendation,
		"evidence_ids":   ids,
		"assumptions": []string{
			"Retrieved evidence is current and applicable to the selected asset context.",
		},
		"unknowns": []string{
			"Instrument calibration status", "Current permit and energy-isolation status",
		},
		"confidence":                  0.72,
		"risk_level":                  "ADVISORY",
		"requires_human_confirmation": true,
	}
}

func deterministicReport(context json.RawMessage, evidence []Evidence) map[string]any {
	var source map[string]any
	if err := json.Unmarshal(context, &source); err != nil {
		source = map[string]any{}
	}
	return map[string]any{
		"summary":               stringValue(source["summary"]),
		"measurements":          describeItems(source["measurements"]),
		"observations":          describeItems(source["observations"]),
		"actions":               describeItems(source["actions"]),
		"outcome":               stringValue(source["outcome"]),
		"evidence_ids":          evidenceIDs(evidence),
		"unknowns":              []string{"Supervisor review and final technical wording"},
		"requires_human_review": true,
	}
}

func deterministicHandover(context json.RawMessage, evidence []Evidence) map[string]any {
	var source map[string]any
	if err := json.Unmarshal(context, &source); err != nil {
		source = map[string]any{}
	}
	return map[string]any{
		"summary":               stringValue(source["summary"]),
		"open_incidents":        describeItems(source["open_incidents"]),
		"active_executions":     describeItems(source["active_executions"]),
		"safety_concerns":       describeItems(source["safety_concerns"]),
		"follow_up":             describeItems(source["follow_up"]),
		"evidence_ids":          evidenceIDs(evidence),
		"unknowns":              []string{"Incoming supervisor review and external permit/isolation status"},
		"requires_human_review": true,
	}
}

func evidenceIDs(evidence []Evidence) []string {
	ids := make([]string, 0, len(evidence))
	for _, item := range evidence {
		ids = append(ids, item.ID)
	}
	sort.Strings(ids)
	return ids
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func describeItems(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, stringValue(item))
	}
	return result
}

func joinEvidence(evidence []Evidence) string {
	var builder strings.Builder
	for _, item := range evidence {
		builder.WriteString(item.Content)
		builder.WriteByte('\n')
	}
	return builder.String()
}

type DeterministicEmbeddingProvider struct {
	Dimensions int
}

func (p DeterministicEmbeddingProvider) Model() EmbeddingModel {
	dimensions := p.Dimensions
	if dimensions <= 0 {
		dimensions = 64
	}
	return EmbeddingModel{
		Provider: DevelopmentProviderName, Model: "hash-bag-of-words",
		ModelVersion: "v1", Dimensions: dimensions, Metric: "COSINE",
	}
}

var embeddingToken = regexp.MustCompile(`[[:alnum:]_]+`)

func (p DeterministicEmbeddingProvider) Embed(
	_ context.Context,
	inputs []string,
) ([][]float32, error) {
	model := p.Model()
	result := make([][]float32, 0, len(inputs))
	for _, input := range inputs {
		vector := make([]float32, model.Dimensions)
		for _, token := range embeddingToken.FindAllString(strings.ToLower(input), -1) {
			sum := sha256.Sum256([]byte(token))
			index := int(binary.LittleEndian.Uint64(sum[:8]) % uint64(model.Dimensions))
			sign := float32(1)
			if sum[8]&1 == 1 {
				sign = -1
			}
			vector[index] += sign
		}
		var norm float64
		for _, value := range vector {
			norm += float64(value * value)
		}
		if norm > 0 {
			scale := float32(1 / math.Sqrt(norm))
			for index := range vector {
				vector[index] *= scale
			}
		}
		result = append(result, vector)
	}
	return result, nil
}

type UnavailableTranscriptionProvider struct{}

func (UnavailableTranscriptionProvider) Model() ProviderMetadata {
	return ProviderMetadata{
		Provider: "unavailable", Model: "unavailable", ModelVersion: "none",
	}
}

func (UnavailableTranscriptionProvider) Transcribe(
	context.Context,
	io.Reader,
	string,
) (Transcript, error) {
	return Transcript{}, ErrProviderUnavailable
}
