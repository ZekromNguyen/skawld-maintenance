package skawld

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPumpDemonstrationCapturesCorrectionAndOutcome(t *testing.T) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	gateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	principal := fixture.principal()
	service := demonstrationapp.Service{Gateway: gateway}
	demo, err := service.Start(ctx, principal, demonstrationapp.Start{
		SubjectKind: demonstrationapp.SubjectExecution,
		SubjectID:   fixture.executionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	recommendationID := uuid.NewString()
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: uuid.NewString(), OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type: "measurement.recorded", AggregateType: "measurement",
		AggregateID: uuid.NewString(), AggregateVersion: 1,
		SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
		Payload: map[string]any{
			"execution_id": fixture.executionID,
			"value": map[string]any{
				"measurement_type": "VIBRATION_VELOCITY",
				"value":            "8.1", "unit": "MM_PER_S",
			},
		},
		OccurredAt: time.Now().UTC(),
	})
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: uuid.NewString(), OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type: "copilot.recommendation.generated", AggregateType: "recommendation",
		AggregateID: recommendationID, AggregateVersion: 1,
		SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
		Payload: map[string]any{
			"execution_id": fixture.executionID,
			"value": map[string]any{
				"recommendation": "Inspect lubrication condition",
				"evidence_ids":   []string{"measurement:test"},
			},
		},
		OccurredAt: time.Now().UTC(),
	})
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: uuid.NewString(), OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type:          "copilot.recommendation.reviewed",
		AggregateType: "recommendation_feedback",
		AggregateID:   uuid.NewString(), AggregateVersion: 1,
		SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
		Payload: map[string]any{
			"execution_id":      fixture.executionID,
			"recommendation_id": recommendationID,
			"value": map[string]any{
				"outcome":    "CORRECTED",
				"correction": "Inspect alignment",
				"reason":     "Coupling was recently disturbed",
			},
		},
		OccurredAt: time.Now().UTC(),
	})
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: uuid.NewString(), OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type: "execution.completed", AggregateType: "maintenance_execution",
		AggregateID: fixture.executionID, AggregateVersion: 2,
		SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
		Payload: map[string]any{
			"outcome_summary": "Alignment corrected; vibration reduced to 3.2 mm/s",
		},
		OccurredAt: time.Now().UTC(),
	})

	processor := CaptureProcessor{
		Pool: pool, Clock: clock.System{},
		Store: ObservationStore{Pool: pool, Clock: clock.System{}},
	}
	if err := processor.ProcessPending(ctx, 100); err != nil {
		t.Fatal(err)
	}
	demo, err = service.Get(ctx, principal, demo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(demo.Events) != 5 {
		t.Fatalf("event count = %d, want 5", len(demo.Events))
	}
	correction := demo.Events[3]
	if correction.Action != "copilot.recommendation.corrected" ||
		correction.CorrectionOf != demo.Events[2].ID {
		t.Fatalf("correction = %+v", correction)
	}
	if demo.Events[4].Result == nil {
		t.Fatal("execution outcome was not captured as result")
	}
	completed, err := service.Complete(ctx, principal, demo.ID, demonstrationapp.Complete{
		Outcome: "Pump high-vibration diagnostic completed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.Capture.Failed != 0 {
		t.Fatalf("completed demonstration = %+v", completed)
	}
}

func TestShiftHandoverProducesNormalWorkDemonstration(t *testing.T) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	handoverID := uuid.NewString()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO shift_handovers (
			id, organization_id, site_id, shift_start, shift_end, state,
			structured_content, evidence_snapshot, provider, model,
			model_version, prompt_version, input_sha256, output_sha256,
			version, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, 'DRAFT',
			'{"summary":"Night shift"}', '[]', 'fixture', 'fixture',
			'1', 'fixture.v1', repeat('a', 64), repeat('b', 64),
			1, $6::uuid, $7, $7
		)
	`, handoverID, fixture.organizationID, fixture.siteID,
		now.Add(-12*time.Hour), now, fixture.principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	gateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	service := demonstrationapp.Service{Gateway: gateway}
	demo, err := service.Start(ctx, fixture.principal(), demonstrationapp.Start{
		SubjectKind: demonstrationapp.SubjectHandover, SubjectID: handoverID,
	})
	if err != nil {
		t.Fatal(err)
	}
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: uuid.NewString(), OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type: "handover.submitted", AggregateType: "shift_handover",
		AggregateID: handoverID, AggregateVersion: 2,
		SubjectKind: "HANDOVER", SubjectID: handoverID,
		Payload: map[string]any{
			"value": map[string]any{
				"state":           "SUBMITTED",
				"safety_concerns": []string{"P-302 remains isolated"},
			},
		},
		OccurredAt: time.Now().UTC(),
	})
	processor := CaptureProcessor{
		Pool: pool, Clock: clock.System{},
		Store: ObservationStore{Pool: pool, Clock: clock.System{}},
	}
	if err := processor.ProcessPending(ctx, 20); err != nil {
		t.Fatal(err)
	}
	demo, err = service.Get(ctx, fixture.principal(), demo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if demo.WorkflowKey != "maintenance.shift_handover" ||
		len(demo.Events) != 2 ||
		demo.Events[1].Action != "handover.submitted" {
		t.Fatalf("handover demonstration = %+v", demo)
	}
}

func TestCaptureFailureIsVisibleWithoutRollingBackDomainEvent(t *testing.T) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	gateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	service := demonstrationapp.Service{Gateway: gateway}
	demo, err := service.Start(ctx, fixture.principal(), demonstrationapp.Start{
		SubjectKind: demonstrationapp.SubjectExecution,
		SubjectID:   fixture.executionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	domainEventID := uuid.NewString()
	appendFixtureEvent(t, ctx, pool, events.Event{
		ID: domainEventID, OrganizationID: fixture.organizationID,
		SiteID: fixture.siteID, ActorID: fixture.principalID,
		Type:          "copilot.recommendation.reviewed",
		AggregateType: "recommendation_feedback",
		AggregateID:   uuid.NewString(), AggregateVersion: 1,
		SubjectKind: "EXECUTION", SubjectID: fixture.executionID,
		Payload: map[string]any{
			"execution_id":      fixture.executionID,
			"recommendation_id": uuid.NewString(),
			"value": map[string]any{
				"outcome": "CORRECTED", "correction": "Inspect alignment",
			},
		},
		OccurredAt: time.Now().UTC(),
	})
	processor := CaptureProcessor{
		Pool: pool, Clock: clock.System{},
		Store: ObservationStore{Pool: pool, Clock: clock.System{}},
	}
	if err := processor.ProcessPending(ctx, 20); err != nil {
		t.Fatal(err)
	}
	demo, err = service.Get(ctx, fixture.principal(), demo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if demo.Capture.Failed != 1 || demo.Capture.LastError == "" {
		t.Fatalf("capture health = %+v", demo.Capture)
	}
	var eventCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM domain_events WHERE id = $1::uuid
	`, domainEventID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("authoritative domain event count = %d, want 1", eventCount)
	}
	_, err = service.Complete(
		ctx, fixture.principal(), demo.ID,
		demonstrationapp.Complete{Outcome: "must not complete with capture gap"},
	)
	if !errors.Is(err, demonstrationapp.ErrCapturePending) {
		t.Fatalf("complete error = %v, want capture pending", err)
	}
}

type demonstrationFixture struct {
	organizationID string
	siteID         string
	principalID    string
	executionID    string
}

func (f demonstrationFixture) principal() identitydomain.Principal {
	return identitydomain.Principal{
		ID: f.principalID, OrganizationID: f.organizationID,
		SiteIDs: []string{f.siteID},
		Roles:   []identitydomain.Role{identitydomain.RoleSeniorTechnician},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationRead:    {},
			identitydomain.PermissionDemonstrationCapture: {},
			identitydomain.PermissionKnowledgeRead:        {},
		},
	}
}

func demonstrationTestPool(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, ctx
}

func seedDemonstrationFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) demonstrationFixture {
	t.Helper()
	value := demonstrationFixture{
		organizationID: uuid.NewString(), siteID: uuid.NewString(),
		principalID: uuid.NewString(), executionID: uuid.NewString(),
	}
	assetID, incidentID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Phase 3 Fixture', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, value.organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version,
			created_at, updated_at
		) VALUES (
			$2::uuid, $1::uuid, 'P3', 'Phase 3 Site', 'UTC', 'ACTIVE', 1, $3, $3
		)
	`, value.organizationID, value.siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Senior Fixture', 'ACTIVE', $2, $2)
	`, value.principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO memberships (
			id, principal_id, organization_id, site_id, role, created_at
		) VALUES (
			gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, 'Senior Technician', $4
		)
	`, value.principalID, value.organizationID, value.siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class, status,
			source_of_truth, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'P-302', 'Process Pump',
			'CENTRIFUGAL_PUMP', 'ACTIVE', 'OWNED_BY_SKAWLD', 1, $4, $4
		)
	`, assetID, value.organizationID, value.siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO incidents (
			id, organization_id, site_id, asset_id, number, summary, priority,
			status, detected_at, source_of_truth, version, created_by,
			created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, 'INC-P3',
			'High vibration', 'HIGH', 'IN_PROGRESS', $5,
			'OWNED_BY_SKAWLD', 1, $6::uuid, $5, $5
		)
	`, incidentID, value.organizationID, value.siteID, assetID, now, value.principalID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO maintenance_executions (
			id, organization_id, site_id, incident_id, asset_id, purpose,
			state, assigned_to, version, started_at, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid,
			'High vibration diagnostic', 'IN_PROGRESS', $6::uuid, 1, $7,
			$6::uuid, $7, $7
		)
	`, value.executionID, value.organizationID, value.siteID, incidentID,
		assetID, value.principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return value
}

func appendFixtureEvent(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	event events.Event,
) {
	t.Helper()
	_, err := database.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		return struct{}{}, events.Append(ctx, tx, event)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDemonstrationListCursorPagination(t *testing.T) {
	pool, ctx := demonstrationTestPool(t)
	fixture := seedDemonstrationFixture(t, ctx, pool)
	gateway := DemonstrationGateway{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	for index := 0; index < 3; index++ {
		captureReviewedPumpDemonstration(
			t, ctx, pool, fixture, gateway,
			"page-"+string(rune('0'+index)),
		)
	}
	page1, hasMore, err := gateway.List(
		ctx, fixture.principal(), "", demonstrationapp.ListFilter{PageSize: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || !hasMore {
		t.Fatalf("page 1 = %d items, hasMore %v; want 2, true", len(page1), hasMore)
	}
	last := page1[len(page1)-1]
	cursor := last.StartedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
	page2, hasMore2, err := gateway.List(
		ctx, fixture.principal(), "", demonstrationapp.ListFilter{PageSize: 2, Cursor: cursor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 || hasMore2 {
		t.Fatalf("page 2 = %d items, hasMore %v; want 1, false", len(page2), hasMore2)
	}
	seen := map[string]bool{}
	for _, item := range append(page1, page2...) {
		seen[item.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("distinct demonstrations across pages = %d, want 3", len(seen))
	}
}
