package skawld

import (
	"context"
	"errors"
	"testing"
	"time"

	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkflowLearningCompilesReviewsPublishesAndRetiresPumpWorkflow(
	t *testing.T,
) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	demonstrationGateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	first := captureReviewedPumpDemonstration(
		t, ctx, pool, fixture, demonstrationGateway, "first",
	)
	second := captureReviewedPumpDemonstration(
		t, ctx, pool, fixture, demonstrationGateway, "second",
	)
	principal := fixture.principal()
	principal.Roles = []identitydomain.Role{identitydomain.RoleAdministrator}
	principal.Permissions = make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(
		identitydomain.RoleAdministrator,
	) {
		principal.Permissions[permission] = struct{}{}
	}
	authorities := fixedWorkflowAuthorities{values: []identitydomain.ApprovalAuthority{{
		ID:             uuid.NewString(),
		SubjectID:      principal.ID,
		OrganizationID: principal.OrganizationID,
		SiteID:         fixture.siteID,
		MaximumRisk:    identitydomain.RiskOperationalLow,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
	}}}
	gateway := WorkflowLearningGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	service := workflowapp.Service{
		Gateway: gateway, Authorities: authorities, Now: time.Now,
	}
	candidate, err := service.Compile(ctx, principal, workflowapp.Compile{
		Name:             "High vibration centrifugal pump inspection",
		Description:      "Evidence-linked Phase 4 fixture",
		DemonstrationIDs: []string{first.ID, second.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Status != "CANDIDATE" ||
		len(candidate.Steps) < 3 ||
		candidate.ToolCatalogDigest == "" ||
		len(candidate.SourceDemonstrations) != 2 {
		t.Fatalf("candidate = %+v", candidate)
	}
	for _, step := range candidate.Steps {
		if len(step.Evidence) != 2 {
			t.Fatalf("step %s evidence = %+v", step.ID, step.Evidence)
		}
		for _, evidence := range step.Evidence {
			if len(evidence.EventIDs) == 0 {
				t.Fatalf("step %s has empty event evidence", step.ID)
			}
		}
	}
	if len(candidate.Improvements) != 2 {
		t.Fatalf(
			"improvement candidates = %d, want 2",
			len(candidate.Improvements),
		)
	}

	var assetID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM assets
		WHERE organization_id = $1::uuid AND tag = 'P-302'
	`, fixture.organizationID).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	effectiveAt := time.Now().UTC().Add(-time.Minute)
	reviewAt := effectiveAt.AddDate(1, 0, 0)
	approved, err := service.Review(
		ctx, principal, candidate.WorkflowID, candidate.Version,
		workflowapp.Review{
			Decision: "APPROVED",
			Reason:   "two coherent traces and safe guidance tools reviewed",
			Applicability: []workflowapp.Applicability{{
				SiteID: fixture.siteID, AssetID: assetID,
				AssetClass:       "CENTRIFUGAL_PUMP",
				ValidationStatus: "VALIDATED",
			}},
			Prerequisites:        []string{"ENERGY_ISOLATION_WHEN_INTRUSIVE"},
			RequiredCompetencies: []string{"CENTRIFUGAL_PUMP"},
			EffectiveAt:          effectiveAt, ReviewAt: reviewAt,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "APPROVED" || len(approved.Reviews) != 1 {
		t.Fatalf("approved candidate = %+v", approved)
	}

	published, err := service.Publish(
		ctx, principal, candidate.WorkflowID, candidate.Version,
		workflowapp.Publish{Reason: "evaluation gates and authority verified"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != "PUBLISHED" ||
		len(published.Evaluations) != 1 ||
		!published.Evaluations[0].GatesPassed {
		t.Fatalf("published workflow = %+v", published)
	}
	expanded, err := service.ExpandApplicability(
		ctx, principal, candidate.WorkflowID, candidate.Version,
		workflowapp.ExpandApplicability{
			Reason: "separate manufacturer scope reviewed",
			Applicability: workflowapp.Applicability{
				SiteID: fixture.siteID, AssetClass: "CENTRIFUGAL_PUMP",
				Manufacturer:     "Fictional Pump Co.",
				ValidationStatus: "LIKELY_APPLICABLE",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded.Applicability) != 2 {
		t.Fatalf(
			"applicability expansion = %+v",
			expanded.Applicability,
		)
	}
	applicable, err := service.Applicable(ctx, principal, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(applicable) != 1 ||
		applicable[0].WorkflowID != candidate.WorkflowID {
		t.Fatalf("applicable workflows = %+v", applicable)
	}

	_, err = pool.Exec(ctx, `
		UPDATE workflow_versions
		SET sdk_payload = jsonb_set(sdk_payload, '{workflow,name}', '"tampered"')
		WHERE workflow_id = $1::uuid AND version = $2
	`, candidate.WorkflowID, candidate.Version)
	if err == nil {
		t.Fatal("published SDK payload mutation must be rejected")
	}
	retired, err := service.Retire(
		ctx, principal, candidate.WorkflowID, candidate.Version,
		workflowapp.Retire{Reason: "fixture retirement"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if retired.Status != "RETIRED" {
		t.Fatalf("retired workflow = %+v", retired)
	}
	applicable, err = service.Applicable(ctx, principal, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(applicable) != 0 {
		t.Fatalf("retired workflow remained applicable: %+v", applicable)
	}
}

func TestWorkflowLearningRejectsUnapprovedDemonstration(t *testing.T) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	gateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	first := captureReviewedPumpDemonstration(
		t, ctx, pool, fixture, gateway, "approved",
	)
	second := capturePumpDemonstration(
		t, ctx, pool, fixture, gateway, "pending-review",
	)
	learning := WorkflowLearningGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	_, err := learning.Compile(
		ctx, fixture.principal(), workflowapp.Compile{
			Name:             "Must reject",
			DemonstrationIDs: []string{first.ID, second.ID},
		},
	)
	if !errors.Is(err, workflowapp.ErrConflict) {
		t.Fatalf("compile error = %v, want conflict", err)
	}
}

func captureReviewedPumpDemonstration(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture demonstrationFixture,
	gateway DemonstrationGateway,
	label string,
) demonstrationapp.Demonstration {
	t.Helper()
	demonstration := capturePumpDemonstration(
		t, ctx, pool, fixture, gateway, label,
	)
	_, err := gateway.Review(
		ctx, fixture.principal(), demonstration.ID,
		demonstrationapp.Review{
			Decision: "APPROVED",
			Reason:   "semantic trace reviewed for workflow learning",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return demonstration
}

func capturePumpDemonstration(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture demonstrationFixture,
	gateway DemonstrationGateway,
	label string,
) demonstrationapp.Demonstration {
	t.Helper()
	service := demonstrationapp.Service{Gateway: gateway}
	demonstration, err := service.Start(
		ctx, fixture.principal(), demonstrationapp.Start{
			SubjectKind: demonstrationapp.SubjectExecution,
			SubjectID:   fixture.executionID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	recommendationID := uuid.NewString()
	base := time.Now().UTC()
	values := []events.Event{
		{
			ID: uuid.NewString(), OrganizationID: fixture.organizationID,
			SiteID: fixture.siteID, ActorID: fixture.principalID,
			Type: "measurement.recorded", AggregateType: "measurement",
			AggregateID: uuid.NewString(), AggregateVersion: 1,
			SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
			Payload: map[string]interface{}{
				"value": map[string]interface{}{
					"measurement_type": "VIBRATION_VELOCITY",
					"value":            "8.1", "unit": "MM_PER_S",
				},
			},
			OccurredAt: base.Add(time.Millisecond),
		},
		{
			ID: uuid.NewString(), OrganizationID: fixture.organizationID,
			SiteID: fixture.siteID, ActorID: fixture.principalID,
			Type:          "copilot.recommendation.generated",
			AggregateType: "recommendation", AggregateID: recommendationID,
			AggregateVersion: 1, SubjectKind: "EXECUTION",
			SubjectID: fixture.executionID,
			Payload: map[string]interface{}{
				"value": map[string]interface{}{
					"recommendation": "Inspect lubrication condition",
				},
			},
			OccurredAt: base.Add(2 * time.Millisecond),
		},
		{
			ID: uuid.NewString(), OrganizationID: fixture.organizationID,
			SiteID: fixture.siteID, ActorID: fixture.principalID,
			Type:          "copilot.recommendation.reviewed",
			AggregateType: "recommendation_feedback",
			AggregateID:   uuid.NewString(), AggregateVersion: 1,
			SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
			Payload: map[string]interface{}{
				"recommendation_id": recommendationID,
				"value": map[string]interface{}{
					"outcome":    "CORRECTED",
					"correction": "Inspect alignment",
					"reason":     "Coupling was recently disturbed " + label,
				},
			},
			OccurredAt: base.Add(3 * time.Millisecond),
		},
		{
			ID: uuid.NewString(), OrganizationID: fixture.organizationID,
			SiteID: fixture.siteID, ActorID: fixture.principalID,
			Type:          "execution.completed",
			AggregateType: "maintenance_execution",
			AggregateID:   fixture.executionID, AggregateVersion: 2,
			SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
			Payload: map[string]interface{}{
				"outcome_summary": "Alignment corrected " + label,
			},
			OccurredAt: base.Add(4 * time.Millisecond),
		},
	}
	for _, event := range values {
		appendFixtureEvent(t, ctx, pool, event)
	}
	processor := CaptureProcessor{
		Pool: pool, Clock: clock.System{},
		Store: ObservationStore{Pool: pool, Clock: clock.System{}},
	}
	if err := processor.ProcessDemonstration(
		ctx, demonstration.ID, 100,
	); err != nil {
		t.Fatal(err)
	}
	demonstration, err = service.Complete(
		ctx, fixture.principal(), demonstration.ID,
		demonstrationapp.Complete{Outcome: "completed " + label},
	)
	if err != nil {
		t.Fatal(err)
	}
	return demonstration
}

type fixedWorkflowAuthorities struct {
	values []identitydomain.ApprovalAuthority
}

func (f fixedWorkflowAuthorities) ForSubject(
	context.Context,
	string,
	string,
	string,
) ([]identitydomain.ApprovalAuthority, error) {
	return f.values, nil
}
