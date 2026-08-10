package application

import (
	"context"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

func TestReviewRequiresApprovalAuthorityBeyondRBAC(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	principal := identitydomain.Principal{
		ID: "reviewer", OrganizationID: "org",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationRead:   {},
			identitydomain.PermissionDemonstrationReview: {},
		},
	}
	gateway := &fakeGateway{demo: Demonstration{
		ID: "demo", OrganizationID: "org", SiteID: "site", Status: "completed",
	}}
	service := Service{
		Gateway: gateway, Authorities: fakeAuthorities{}, Now: func() time.Time { return now },
	}
	_, err := service.Review(context.Background(), principal, "demo", Review{
		Decision: "APPROVED", Reason: "coherent trace",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("review error = %v, want forbidden", err)
	}

	service.Authorities = fakeAuthorities{values: []identitydomain.ApprovalAuthority{{
		SubjectID: "reviewer", OrganizationID: "org", SiteID: "site",
		ScopeKind: "demonstration", ScopeID: "demo",
		MaximumRisk: identitydomain.RiskAdvisory,
	}}}
	if _, err := service.Review(
		context.Background(), principal, "demo",
		Review{Decision: "APPROVED", Reason: "coherent trace"},
	); err != nil {
		t.Fatal(err)
	}
	if gateway.reviewCalls != 1 {
		t.Fatalf("review calls = %d, want 1", gateway.reviewCalls)
	}
}

func TestRedactionPathFailsClosed(t *testing.T) {
	t.Parallel()
	service := Service{Gateway: &fakeGateway{}}
	principal := identitydomain.Principal{
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationReview: {},
		},
	}
	_, err := service.RedactEvent(
		context.Background(), principal, "demo", "event",
		RedactEvent{
			JSONPath: "principal.actor_id", Action: "DROP", Reason: "hide actor",
		},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("redaction error = %v, want invalid", err)
	}
}

type fakeAuthorities struct {
	values []identitydomain.ApprovalAuthority
}

func (f fakeAuthorities) ForSubject(
	context.Context,
	string,
	string,
	string,
) ([]identitydomain.ApprovalAuthority, error) {
	return f.values, nil
}

type fakeGateway struct {
	demo        Demonstration
	reviewCalls int
}

func (f *fakeGateway) Start(
	context.Context,
	identitydomain.Principal,
	Start,
) (Demonstration, error) {
	return f.demo, nil
}
func (f *fakeGateway) Get(
	context.Context,
	identitydomain.Principal,
	string,
) (Demonstration, error) {
	return f.demo, nil
}
func (f *fakeGateway) List(
	context.Context,
	identitydomain.Principal,
	string,
	ListFilter,
) ([]Demonstration, bool, error) {
	return []Demonstration{f.demo}, false, nil
}
func (f *fakeGateway) Complete(
	context.Context,
	identitydomain.Principal,
	string,
	Complete,
) (Demonstration, error) {
	return f.demo, nil
}
func (f *fakeGateway) RecordEvidenceView(
	context.Context,
	identitydomain.Principal,
	string,
	RecordEvidenceView,
) (Event, error) {
	return Event{}, nil
}
func (f *fakeGateway) Review(
	_ context.Context,
	_ identitydomain.Principal,
	_ string,
	command Review,
) (ReviewRecord, error) {
	f.reviewCalls++
	return ReviewRecord{Decision: command.Decision}, nil
}
func (f *fakeGateway) RedactEvent(
	context.Context,
	identitydomain.Principal,
	string,
	string,
	RedactEvent,
) (Redaction, error) {
	return Redaction{}, nil
}
