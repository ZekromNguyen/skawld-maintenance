package application

import (
	"context"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

func TestPublishRequiresPermissionAndSeparateApprovalAuthority(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	current := Version{
		WorkflowID: "workflow-1", Version: 1, Status: "APPROVED",
		SiteID: "site-1", AssetClass: "CENTRIFUGAL_PUMP",
		EffectiveAt: &now,
		ReviewAt: func() *time.Time {
			value := now.Add(time.Hour)
			return &value
		}(),
		Applicability: []Applicability{{
			SiteID: "site-1", AssetClass: "CENTRIFUGAL_PUMP",
			ValidationStatus: "VALIDATED",
		}},
	}
	gateway := workflowGatewayStub{current: current}
	principal := identitydomain.Principal{
		ID: "principal-1", OrganizationID: "organization-1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionWorkflowRead:    {},
			identitydomain.PermissionWorkflowPublish: {},
		},
	}
	service := Service{
		Gateway:     gateway,
		Authorities: workflowAuthorityStub{},
		Now:         func() time.Time { return now },
	}
	_, err := service.Publish(
		context.Background(), principal, current.WorkflowID, 1,
		Publish{Reason: "release"},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("publish without authority = %v, want forbidden", err)
	}

	service.Authorities = workflowAuthorityStub{
		values: []identitydomain.ApprovalAuthority{{
			SubjectID: "principal-1", OrganizationID: "organization-1",
			SiteID: "site-1", ScopeKind: "workflow",
			ScopeID: "workflow-1", Competency: "CENTRIFUGAL_PUMP",
			MaximumRisk: identitydomain.RiskOperationalLow,
			ValidFrom:   now.Add(-time.Hour), ValidUntil: now.Add(time.Hour),
		}},
	}
	published, err := service.Publish(
		context.Background(), principal, current.WorkflowID, 1,
		Publish{Reason: "release"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != "PUBLISHED" {
		t.Fatalf("published = %+v", published)
	}
}

func TestCompileRequiresTwoDistinctDemonstrations(t *testing.T) {
	t.Parallel()
	service := Service{Gateway: workflowGatewayStub{}}
	principal := identitydomain.Principal{
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionWorkflowReview: {},
		},
	}
	_, err := service.Compile(
		context.Background(), principal,
		Compile{
			Name:             "Pump",
			DemonstrationIDs: []string{"demo-1", "demo-1"},
		},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("compile duplicate demonstrations = %v, want invalid", err)
	}
}

func TestInitialReviewCannotExpandLearnedApplicability(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	current := Version{
		WorkflowID: "workflow-1", Version: 1, Status: "CANDIDATE",
		SiteID: "site-1", AssetClass: "CENTRIFUGAL_PUMP",
	}
	principal := identitydomain.Principal{
		ID: "principal-1", OrganizationID: "organization-1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionWorkflowRead:   {},
			identitydomain.PermissionWorkflowReview: {},
		},
	}
	service := Service{
		Gateway: workflowGatewayStub{current: current},
		Now:     func() time.Time { return now },
	}
	_, err := service.Review(
		context.Background(), principal, current.WorkflowID, current.Version,
		Review{
			Decision: "APPROVED", Reason: "attempted scope expansion",
			Applicability: []Applicability{{
				SiteID: "site-2", AssetClass: current.AssetClass,
				ValidationStatus: "LIKELY_APPLICABLE",
			}},
			EffectiveAt: now, ReviewAt: now.Add(time.Hour),
		},
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-site initial review = %v, want invalid", err)
	}
}

type workflowGatewayStub struct {
	current Version
}

func (s workflowGatewayStub) Compile(
	context.Context,
	identitydomain.Principal,
	Compile,
) (Version, error) {
	return s.current, nil
}

func (s workflowGatewayStub) Get(
	context.Context,
	identitydomain.Principal,
	string,
	int,
) (Version, error) {
	return s.current, nil
}

func (s workflowGatewayStub) List(
	context.Context,
	identitydomain.Principal,
) ([]Version, error) {
	return []Version{s.current}, nil
}

func (s workflowGatewayStub) Review(
	context.Context,
	identitydomain.Principal,
	string,
	int,
	Review,
) (Version, error) {
	return s.current, nil
}

func (s workflowGatewayStub) Publish(
	_ context.Context,
	_ identitydomain.Principal,
	_ string,
	_ int,
	_ Publish,
) (Version, error) {
	value := s.current
	value.Status = "PUBLISHED"
	return value, nil
}

func (s workflowGatewayStub) ExpandApplicability(
	context.Context,
	identitydomain.Principal,
	string,
	int,
	ExpandApplicability,
) (Version, error) {
	return s.current, nil
}

func (s workflowGatewayStub) Retire(
	context.Context,
	identitydomain.Principal,
	string,
	int,
	Retire,
) (Version, error) {
	return s.current, nil
}

func (s workflowGatewayStub) Applicable(
	context.Context,
	identitydomain.Principal,
	string,
) ([]Version, error) {
	return []Version{s.current}, nil
}

type workflowAuthorityStub struct {
	values []identitydomain.ApprovalAuthority
}

func (s workflowAuthorityStub) ForSubject(
	context.Context,
	string,
	string,
	string,
) ([]identitydomain.ApprovalAuthority, error) {
	return s.values, nil
}
