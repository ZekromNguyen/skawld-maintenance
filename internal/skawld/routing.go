package skawld

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

type Capability string

const (
	CapabilityRecommendation Capability = "maintenance.recommendation"
	CapabilityReportDraft    Capability = "maintenance.report_draft"
	CapabilityShiftHandover  Capability = "maintenance.shift_handover"
	CapabilityEmbedding      Capability = "knowledge.embedding"
	CapabilityTranscription  Capability = "evidence.transcription"
	CapabilityVision         Capability = "evidence.vision_candidate"
)

var (
	ErrProviderUnavailable = errors.New("AI capability provider unavailable")
	ErrInvalidOutput       = errors.New("AI provider returned invalid structured output")
)

type Evidence struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	SourceID  string `json:"source_id"`
	Revision  string `json:"revision,omitempty"`
	Locator   string `json:"locator"`
	Authority string `json:"authority"`
	Content   string `json:"content"`
}

type GenerateRequest struct {
	Capability    Capability      `json:"capability"`
	SchemaVersion string          `json:"schema_version"`
	PromptVersion string          `json:"prompt_version"`
	Context       json.RawMessage `json:"context"`
	Evidence      []Evidence      `json:"evidence"`
}

type ProviderMetadata struct {
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	ModelVersion        string `json:"model_version"`
	TokensIn            int    `json:"tokens_in,omitempty"`
	TokensOut           int    `json:"tokens_out,omitempty"`
	EstimatedCostMicros int64  `json:"estimated_cost_micros,omitempty"`
}

type GenerateResponse struct {
	Output   json.RawMessage
	Metadata ProviderMetadata
}

type StructuredProvider interface {
	Generate(context.Context, GenerateRequest) (GenerateResponse, error)
}

type Generation struct {
	Output     json.RawMessage
	Metadata   ProviderMetadata
	InputHash  string
	OutputHash string
	Latency    time.Duration
	Prompt     string
	Schema     string
	Capability Capability
}

type Router struct {
	Providers map[Capability]StructuredProvider
}

func (r Router) Generate(
	ctx context.Context,
	request GenerateRequest,
) (Generation, error) {
	provider := r.Providers[request.Capability]
	if provider == nil {
		return Generation{}, ErrProviderUnavailable
	}
	input, err := json.Marshal(request)
	if err != nil {
		return Generation{}, fmt.Errorf("encode structured provider request: %w", err)
	}
	started := time.Now()
	response, err := provider.Generate(ctx, request)
	latency := time.Since(started)
	if err != nil {
		return Generation{InputHash: hash(input), Latency: latency}, err
	}
	if !json.Valid(response.Output) {
		return Generation{InputHash: hash(input), Latency: latency}, ErrInvalidOutput
	}
	return Generation{
		Output: response.Output, Metadata: response.Metadata,
		InputHash: hash(input), OutputHash: hash(response.Output),
		Latency: latency, Prompt: request.PromptVersion,
		Schema: request.SchemaVersion, Capability: request.Capability,
	}, nil
}

type EmbeddingModel struct {
	Provider     string
	Model        string
	ModelVersion string
	Dimensions   int
	Metric       string
}

type EmbeddingProvider interface {
	Model() EmbeddingModel
	Embed(context.Context, []string) ([][]float32, error)
}

type Transcript struct {
	Text       string
	Language   string
	Confidence float64
	Metadata   ProviderMetadata
}

type TranscriptionProvider interface {
	Model() ProviderMetadata
	Transcribe(context.Context, io.Reader, string) (Transcript, error)
}

type ObservationCandidate struct {
	Component  string  `json:"component,omitempty"`
	Property   string  `json:"property"`
	Status     string  `json:"status"`
	Narrative  string  `json:"narrative"`
	Confidence float64 `json:"confidence"`
}

// VisionProvider may only create unverified candidates. Converting a
// candidate into a maintenance observation is a separate human command.
type VisionProvider interface {
	Model() ProviderMetadata
	Analyze(context.Context, io.Reader, string) ([]ObservationCandidate, error)
}

func HashBytes(value []byte) string {
	return hash(value)
}

func hash(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
