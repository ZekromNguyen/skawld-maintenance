package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	reportpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/report/adapter/postgres"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type emptySearch struct{}

func (emptySearch) Search(context.Context, identitydomain.Principal, knowledgedomain.SearchQuery) (knowledgedomain.SearchResult, error) {
	return knowledgedomain.SearchResult{}, nil
}

func TestReportDraftEditAndApprovalPersistence(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID := uuid.NewString(), uuid.NewString()
	principalID, assetID, executionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Report Test', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, 'RP', 'Report Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, siteID, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Report Technician', 'ACTIVE', $2, $2)
	`, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class, status,
			source_of_truth, attributes, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'P-REPORT', 'Report Pump',
			'CENTRIFUGAL_PUMP', 'ACTIVE', 'OWNED_BY_SKAWLD',
			'{}'::jsonb, 1, $4, $4
		)
	`, assetID, organizationID, siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO maintenance_executions (
			id, organization_id, site_id, asset_id, purpose, state,
			assigned_to, version, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid,
			'High vibration evidence report', 'ASSIGNED',
			$5::uuid, 1, $5::uuid, $6, $6
		)
	`, executionID, organizationID, siteID, assetID, principalID, now)
	if err != nil {
		t.Fatal(err)
	}

	store := reportpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Audit: audit.Sink{}, Idempotency: idempotency.Store{},
	}
	service := reportapp.Service{
		Store: store, Search: emptySearch{},
		Router: skawld.Router{Providers: map[skawld.Capability]skawld.StructuredProvider{
			skawld.CapabilityReportDraft: skawld.DeterministicProvider{},
		}},
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionReportWrite: {},
		},
	}
	value, replay, err := service.Draft(
		ctx, principal, uuid.NewString(), executionID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replay || value.State != "DRAFT" || value.Provider != skawld.DevelopmentProviderName {
		t.Fatalf("unexpected draft: %+v", value)
	}
	value.Content.Outcome = "Bearing condition documented for supervisor review."
	value, _, err = service.Edit(
		ctx, principal, uuid.NewString(), value.ID,
		reportapp.Edit{ExpectedVersion: value.Version, Content: value.Content},
	)
	if err != nil {
		t.Fatal(err)
	}
	value, _, err = service.Submit(
		ctx, principal, uuid.NewString(), value.ID,
		reportapp.Transition{ExpectedVersion: value.Version},
	)
	if err != nil {
		t.Fatal(err)
	}
	value, _, err = store.Approve(
		ctx, principal, uuid.NewString(), value.ID,
		reportapp.Transition{ExpectedVersion: value.Version},
	)
	if err != nil {
		t.Fatal(err)
	}
	if value.State != "APPROVED" || value.ApprovedBy != principalID ||
		value.Content.Outcome != "Bearing condition documented for supervisor review." {
		t.Fatalf("unexpected approved report: %+v", value)
	}
	var editCount int
	if err := pool.QueryRow(
		ctx, `SELECT count(*) FROM maintenance_report_edits WHERE report_id = $1::uuid`,
		value.ID,
	).Scan(&editCount); err != nil {
		t.Fatal(err)
	}
	if editCount != 1 {
		t.Fatalf("edit history count = %d", editCount)
	}
}
