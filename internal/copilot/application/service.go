package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

var (
	ErrForbidden       = errors.New("copilot operation forbidden")
	ErrInvalid         = errors.New("invalid copilot command")
	ErrNotFound        = errors.New("copilot context not found")
	ErrEvidenceInvalid = errors.New("recommendation cited evidence outside the eligible packet")
	ErrUnsafeOutput    = errors.New("recommendation exceeded the advisory safety envelope")
)

const (
	recommendationSchema = "maintenance.recommendation.v1"
	recommendationPrompt = "maintenance.recommendation.evidence-first.v1"
)

type GenerateRecommendation struct {
	Question    string `json:"question,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
}

type RecommendationOutput struct {
	Status                    string   `json:"status"`
	Recommendation            string   `json:"recommendation"`
	EvidenceIDs               []string `json:"evidence_ids"`
	Assumptions               []string `json:"assumptions"`
	Unknowns                  []string `json:"unknowns"`
	Confidence                float64  `json:"confidence"`
	RiskLevel                 string   `json:"risk_level"`
	RequiresHumanConfirmation bool     `json:"requires_human_confirmation"`
}

type Recommendation struct {
	ID             string                     `json:"id"`
	OrganizationID string                     `json:"organization_id"`
	SiteID         string                     `json:"site_id"`
	IncidentID     string                     `json:"incident_id"`
	ExecutionID    string                     `json:"execution_id,omitempty"`
	Output         RecommendationOutput       `json:"output"`
	Evidence       []knowledgedomain.Evidence `json:"evidence"`
	RetrievalRunID string                     `json:"retrieval_run_id"`
	Provider       string                     `json:"provider"`
	Model          string                     `json:"model"`
	ModelVersion   string                     `json:"model_version"`
	PromptVersion  string                     `json:"prompt_version"`
	InputSHA256    string                     `json:"input_sha256"`
	OutputSHA256   string                     `json:"output_sha256"`
	LatencyMS      int64                      `json:"latency_ms"`
	CreatedAt      time.Time                  `json:"created_at"`
}

type IncidentContext struct {
	OrganizationID string          `json:"organization_id"`
	SiteID         string          `json:"site_id"`
	IncidentID     string          `json:"incident_id"`
	ExecutionID    string          `json:"execution_id,omitempty"`
	AssetID        string          `json:"asset_id"`
	Summary        string          `json:"summary"`
	Snapshot       json.RawMessage `json:"snapshot"`
}

type Feedback struct {
	Outcome           string `json:"outcome"`
	Correction        string `json:"correction,omitempty"`
	Reason            string `json:"reason,omitempty"`
	MaterialClaims    *int   `json:"material_claims,omitempty"`
	SupportedClaims   *int   `json:"supported_claims,omitempty"`
	RetrievedEvidence *int   `json:"retrieved_evidence,omitempty"`
	RelevantEvidence  *int   `json:"relevant_evidence,omitempty"`
}

type Searcher interface {
	Search(context.Context, identitydomain.Principal, knowledgedomain.SearchQuery) (knowledgedomain.SearchResult, error)
}

type Store interface {
	LoadIncidentContext(context.Context, identitydomain.Principal, string, string) (IncidentContext, error)
	SaveRecommendation(context.Context, identitydomain.Principal, string, GenerateRecommendation, Recommendation, json.RawMessage) (Recommendation, bool, error)
	GetRecommendation(context.Context, identitydomain.Principal, string) (Recommendation, error)
	RecordCall(context.Context, identitydomain.Principal, IncidentContext, skawld.Generation, string, string) error
	RecordFeedback(context.Context, identitydomain.Principal, string, string, Feedback) error
}

type Service struct {
	Store  Store
	Search Searcher
	Router skawld.Router
}

func (s Service) Generate(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command GenerateRecommendation,
) (Recommendation, bool, error) {
	if !principal.Has(identitydomain.PermissionRecommendationRun) {
		return Recommendation{}, false, ErrForbidden
	}
	command.Question = strings.TrimSpace(command.Question)
	if len(strings.TrimSpace(key)) < 8 || len(key) > 200 ||
		len(command.Question) > 500 {
		return Recommendation{}, false, ErrInvalid
	}
	incident, err := s.Store.LoadIncidentContext(
		ctx, principal, incidentID, command.ExecutionID,
	)
	if err != nil {
		return Recommendation{}, false, err
	}
	queryText := incident.Summary
	if command.Question != "" {
		queryText += " " + command.Question
	}
	retrieval, err := s.Search.Search(ctx, principal, knowledgedomain.SearchQuery{
		SiteID: incident.SiteID, AssetID: incident.AssetID,
		Query: queryText, Limit: 8,
	})
	if err != nil {
		return Recommendation{}, false, err
	}
	packet := make([]skawld.Evidence, 0, len(retrieval.Items))
	for _, evidence := range retrieval.Items {
		packet = append(packet, skawld.Evidence{
			ID: evidence.ID, Kind: evidence.Kind, SourceID: evidence.SourceID,
			Revision: evidence.Revision, Locator: evidence.Locator,
			Authority: string(evidence.Authority), Content: evidence.Content,
		})
	}
	generation, err := s.Router.Generate(ctx, skawld.GenerateRequest{
		Capability:    skawld.CapabilityRecommendation,
		SchemaVersion: recommendationSchema, PromptVersion: recommendationPrompt,
		Context: incident.Snapshot, Evidence: packet,
	})
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, incident, generation, "PROVIDER_ERROR", errorCode(err))
		return Recommendation{}, false, fmt.Errorf("generate recommendation: %w", err)
	}
	output, err := decodeRecommendation(generation.Output, packet)
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, incident, generation, "INVALID_OUTPUT", errorCode(err))
		return Recommendation{}, false, err
	}
	outcome := "SUCCEEDED"
	if output.Status == "INSUFFICIENT_EVIDENCE" {
		outcome = "INSUFFICIENT_EVIDENCE"
	}
	if err := s.Store.RecordCall(ctx, principal, incident, generation, outcome, ""); err != nil {
		return Recommendation{}, false, err
	}
	result := Recommendation{
		OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
		IncidentID: incident.IncidentID, ExecutionID: incident.ExecutionID,
		Output: output, Evidence: retrieval.Items, RetrievalRunID: retrieval.RetrievalRunID,
		Provider: generation.Metadata.Provider, Model: generation.Metadata.Model,
		ModelVersion: generation.Metadata.ModelVersion, PromptVersion: generation.Prompt,
		InputSHA256: generation.InputHash, OutputSHA256: generation.OutputHash,
		LatencyMS: generation.Latency.Milliseconds(),
	}
	return s.Store.SaveRecommendation(
		ctx, principal, key, command, result, incident.Snapshot,
	)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	recommendationID string,
) (Recommendation, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeRead) {
		return Recommendation{}, ErrForbidden
	}
	return s.Store.GetRecommendation(ctx, principal, recommendationID)
}

func (s Service) Feedback(
	ctx context.Context,
	principal identitydomain.Principal,
	key, recommendationID string,
	command Feedback,
) error {
	if !principal.Has(identitydomain.PermissionRecommendationReview) {
		return ErrForbidden
	}
	switch command.Outcome {
	case "ACCEPTED", "REJECTED", "CORRECTED", "UNSAFE",
		"UNSUPPORTED", "INCORRECT_NEXT_STEP":
	default:
		return ErrInvalid
	}
	command.Correction = strings.TrimSpace(command.Correction)
	command.Reason = strings.TrimSpace(command.Reason)
	if command.Outcome == "CORRECTED" && command.Correction == "" {
		return ErrInvalid
	}
	if command.Outcome == "UNSAFE" && command.Reason == "" {
		return ErrInvalid
	}
	if !validReviewCounts(
		command.MaterialClaims,
		command.SupportedClaims,
	) || !validReviewCounts(
		command.RetrievedEvidence,
		command.RelevantEvidence,
	) {
		return ErrInvalid
	}
	if len(strings.TrimSpace(key)) < 8 {
		return ErrInvalid
	}
	return s.Store.RecordFeedback(ctx, principal, key, recommendationID, command)
}

func validReviewCounts(total, accepted *int) bool {
	if (total == nil) != (accepted == nil) {
		return false
	}
	if total == nil {
		return true
	}
	return *total >= 0 && *accepted >= 0 && *accepted <= *total
}

func decodeRecommendation(
	raw json.RawMessage,
	evidence []skawld.Evidence,
) (RecommendationOutput, error) {
	var output RecommendationOutput
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return RecommendationOutput{}, errors.Join(skawld.ErrInvalidOutput, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return RecommendationOutput{}, skawld.ErrInvalidOutput
	}
	if output.Confidence < 0 || output.Confidence > 1 ||
		!output.RequiresHumanConfirmation ||
		len(output.Assumptions) > 10 || len(output.Unknowns) > 10 {
		return RecommendationOutput{}, skawld.ErrInvalidOutput
	}
	switch output.Status {
	case "RECOMMENDATION":
		if strings.TrimSpace(output.Recommendation) == "" || len(output.EvidenceIDs) == 0 {
			return RecommendationOutput{}, ErrEvidenceInvalid
		}
	case "INSUFFICIENT_EVIDENCE":
		if output.Recommendation != "" || len(output.EvidenceIDs) != 0 {
			return RecommendationOutput{}, ErrEvidenceInvalid
		}
	default:
		return RecommendationOutput{}, skawld.ErrInvalidOutput
	}
	switch output.RiskLevel {
	case "INFORMATIONAL", "ADVISORY":
	default:
		return RecommendationOutput{}, ErrUnsafeOutput
	}
	eligible := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		eligible[item.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(output.EvidenceIDs))
	for _, evidenceID := range output.EvidenceIDs {
		if _, ok := eligible[evidenceID]; !ok {
			return RecommendationOutput{}, ErrEvidenceInvalid
		}
		if _, duplicate := seen[evidenceID]; duplicate {
			return RecommendationOutput{}, ErrEvidenceInvalid
		}
		seen[evidenceID] = struct{}{}
	}
	return output, nil
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "TIMEOUT"
	case errors.Is(err, ErrEvidenceInvalid):
		return "INVENTED_EVIDENCE"
	case errors.Is(err, ErrUnsafeOutput):
		return "UNSAFE_RISK"
	case errors.Is(err, skawld.ErrInvalidOutput):
		return "INVALID_JSON"
	default:
		return "PROVIDER_ERROR"
	}
}
