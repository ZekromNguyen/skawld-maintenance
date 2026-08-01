package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type fakeProvider struct {
	output json.RawMessage
}

func (p fakeProvider) Generate(context.Context, skawld.GenerateRequest) (skawld.GenerateResponse, error) {
	return skawld.GenerateResponse{
		Output: p.output,
		Metadata: skawld.ProviderMetadata{
			Provider: "fake", Model: "fake", ModelVersion: "v1",
		},
	}, nil
}

type fakeSearch struct {
	result knowledgedomain.SearchResult
}

func (s fakeSearch) Search(context.Context, identitydomain.Principal, knowledgedomain.SearchQuery) (knowledgedomain.SearchResult, error) {
	return s.result, nil
}

type fakeStore struct {
	saved       bool
	callOutcome string
	feedback    Feedback
}

func (s *fakeStore) LoadIncidentContext(context.Context, identitydomain.Principal, string, string) (IncidentContext, error) {
	return IncidentContext{
		OrganizationID: "organization", SiteID: "site", IncidentID: "incident",
		AssetID: "asset", Summary: "high vibration",
		Snapshot: json.RawMessage(`{"summary":"high vibration"}`),
	}, nil
}
func (s *fakeStore) SaveRecommendation(_ context.Context, _ identitydomain.Principal, _ string, _ GenerateRecommendation, value Recommendation, _ json.RawMessage) (Recommendation, bool, error) {
	s.saved = true
	return value, false, nil
}
func (s *fakeStore) GetRecommendation(context.Context, identitydomain.Principal, string) (Recommendation, error) {
	return Recommendation{}, ErrNotFound
}
func (s *fakeStore) RecordCall(_ context.Context, _ identitydomain.Principal, _ IncidentContext, _ skawld.Generation, outcome, _ string) error {
	s.callOutcome = outcome
	return nil
}
func (s *fakeStore) RecordFeedback(
	_ context.Context,
	_ identitydomain.Principal,
	_, _ string,
	value Feedback,
) error {
	s.feedback = value
	return nil
}

func TestGenerateRejectsInventedEvidenceID(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	service := Service{
		Store: store,
		Search: fakeSearch{result: knowledgedomain.SearchResult{
			Items: []knowledgedomain.Evidence{{
				ID: "document_chunk:eligible", Kind: "DOCUMENT_CHUNK",
				SourceID: "eligible", Content: "Inspect lubrication.",
			}},
		}},
		Router: skawld.Router{Providers: map[skawld.Capability]skawld.StructuredProvider{
			skawld.CapabilityRecommendation: fakeProvider{output: json.RawMessage(`{
				"status":"RECOMMENDATION",
				"recommendation":"Open the casing.",
				"evidence_ids":["document_chunk:invented"],
				"assumptions":[],
				"unknowns":[],
				"confidence":0.9,
				"risk_level":"ADVISORY",
				"requires_human_confirmation":true
			}`)},
		}},
	}
	_, _, err := service.Generate(
		context.Background(),
		identitydomain.Principal{
			ID: "principal", OrganizationID: "organization", SiteIDs: []string{"site"},
			Permissions: map[identitydomain.Permission]struct{}{
				identitydomain.PermissionRecommendationRun: {},
			},
		},
		"retry-key-123", "incident", GenerateRecommendation{},
	)
	if !errors.Is(err, ErrEvidenceInvalid) {
		t.Fatalf("error = %v, want ErrEvidenceInvalid", err)
	}
	if store.saved {
		t.Fatal("invalid recommendation must not be persisted")
	}
	if store.callOutcome != "INVALID_OUTPUT" {
		t.Fatalf("call outcome = %q", store.callOutcome)
	}
}

func TestFeedbackAcceptsQualityLabelsAndRejectsMismatchedCounts(
	t *testing.T,
) {
	t.Parallel()
	store := &fakeStore{}
	service := Service{Store: store}
	principal := identitydomain.Principal{
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionRecommendationReview: {},
		},
	}
	material, supported := 2, 1
	err := service.Feedback(
		context.Background(), principal, "quality-review-1", "recommendation",
		Feedback{
			Outcome: "UNSUPPORTED", Reason: "one claim lacks evidence",
			MaterialClaims: &material, SupportedClaims: &supported,
		},
	)
	if err != nil || store.feedback.Outcome != "UNSUPPORTED" {
		t.Fatalf("feedback = %+v, error = %v", store.feedback, err)
	}
	tooMany := 3
	err = service.Feedback(
		context.Background(), principal, "quality-review-2", "recommendation",
		Feedback{
			Outcome: "ACCEPTED", MaterialClaims: &material,
			SupportedClaims: &tooMany,
		},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("mismatched counts error = %v", err)
	}
}
