package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	handoverpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/handover/adapter/postgres"
	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPrepareHandoverPersistsExactProvenance(t *testing.T) {
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
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Handover Test', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, 'HO', 'Handover Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, siteID, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Outgoing Supervisor', 'ACTIVE', $2, $2)
	`, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	store := handoverpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Audit: audit.Sink{}, Idempotency: idempotency.Store{},
	}
	service := handoverapp.Service{
		Store: store,
		Router: skawld.Router{Providers: map[skawld.Capability]skawld.StructuredProvider{
			skawld.CapabilityShiftHandover: skawld.DeterministicProvider{},
		}},
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionHandoverWrite: {},
		},
	}
	value, replay, err := service.Prepare(
		ctx, principal, uuid.NewString(), handoverapp.PrepareDraft{
			SiteID: siteID, ShiftStart: now.Add(-12 * time.Hour), ShiftEnd: now,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if replay || value.State != "DRAFT" || value.Provider != skawld.DevelopmentProviderName ||
		value.PromptVersion == "" || value.InputSHA256 == "" || value.OutputSHA256 == "" {
		t.Fatalf("unexpected handover: %+v", value)
	}
	stored, err := store.Get(ctx, principal, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.OutputSHA256 != value.OutputSHA256 ||
		stored.Content.Summary != value.Content.Summary {
		t.Fatal("stored handover lost generated provenance")
	}
}

func TestHandoverListPagination(t *testing.T) {
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
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organizations (id, name, source_of_truth, version, created_at, updated_at)
		  VALUES ($1::uuid, 'Handover List', 'OWNED_BY_SKAWLD', 1, $2, $2)`,
			[]any{organizationID, now}},
		{`INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
		  VALUES ($1::uuid, $2::uuid, 'HL', 'Handover List Site', 'UTC', 'ACTIVE', 1, $3, $3)`,
			[]any{siteID, organizationID, now}},
		{`INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
		  VALUES ($1::uuid, $1, 'List Supervisor', 'ACTIVE', $2, $2)`,
			[]any{principalID, now}},
		{`INSERT INTO assets (id, organization_id, site_id, tag, name, asset_class, status, source_of_truth, attributes, version, created_at, updated_at)
		  VALUES ($1::uuid, $2::uuid, $3::uuid, 'P-HLIST', 'List Pump', 'CENTRIFUGAL_PUMP', 'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4)`,
			[]any{assetID, organizationID, siteID, now}},
		{`INSERT INTO maintenance_executions (id, organization_id, site_id, asset_id, purpose, state, assigned_to, version, created_by, created_at, updated_at)
		  VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'List handover evidence', 'ASSIGNED', $5::uuid, 1, $5::uuid, $6, $6)`,
			[]any{executionID, organizationID, siteID, assetID, principalID, now}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	insertHandover := func(id string, shiftStart time.Time) {
		statement := `INSERT INTO shift_handovers
			(id, organization_id, site_id, shift_start, shift_end, state,
			 structured_content, evidence_snapshot, provider, model, model_version,
			 prompt_version, input_sha256, output_sha256, version, created_by, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, 'DRAFT',
			 '{}'::jsonb, '[]'::jsonb, 'skawld-copilot', 'copilot-v1', '1', 'p1',
			 'a', 'b', 1, $6::uuid, $4, $4)`
		if _, err := pool.Exec(ctx, statement, id, organizationID, siteID, shiftStart, shiftStart.Add(12*time.Hour), principalID); err != nil {
			t.Fatal(err)
		}
	}
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		id := uuid.NewString()
		ids = append(ids, id)
		insertHandover(id, now.Add(-time.Duration(i)*time.Hour))
	}

	store := handoverpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Audit: audit.Sink{}, Idempotency: idempotency.Store{},
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionHandoverWrite: {},
		},
	}

	pageOne, hasMore, err := store.List(ctx, principal, handoverapp.HandoverFilter{PageSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !hasMore {
		t.Fatal("hasMore = false, want true for 5 rows at page_size 3")
	}
	if len(pageOne) != 3 {
		t.Fatalf("len(pageOne) = %d, want 3", len(pageOne))
	}
	if pageOne[0].ID != ids[0] {
		t.Fatalf("newest shift first: got %s, want %s", pageOne[0].ID, ids[0])
	}

	pageTwo, hasMoreTwo, err := store.List(ctx, principal, handoverapp.HandoverFilter{
		PageSize: 3,
		Cursor:   pageOne[2].ShiftStart.UTC().Format(time.RFC3339Nano) + "|" + pageOne[2].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasMoreTwo {
		t.Fatal("hasMore = true, want false after exhausting 5 rows")
	}
	if len(pageTwo) != 2 {
		t.Fatalf("len(pageTwo) = %d, want 2", len(pageTwo))
	}

	scoped, _, err := store.List(ctx, principal, handoverapp.HandoverFilter{PageSize: 25, SiteID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 0 {
		t.Fatalf("len(scoped) = %d, want 0 for foreign site", len(scoped))
	}
}
