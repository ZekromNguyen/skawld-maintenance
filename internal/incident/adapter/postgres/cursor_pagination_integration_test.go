package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	incidentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/incident/adapter/postgres"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIncidentListCursorPagination(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	seedOrgSitePrincipal(t, ctx, pool, organizationID, siteID, principalID, now)
	assetID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class,
			status, source_of_truth, attributes, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'P-302', 'Pump P-302', 'PUMP',
			'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4
		)
	`, assetID, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}

	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}
	store := incidentpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{},
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	for index := 0; index < 3; index++ {
		if _, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
			SiteID: siteID, AssetID: assetID,
			Summary:  "Vibration " + string(rune('0'+index)),
			Priority: "MEDIUM", SourceOfTruth: "OWNED_BY_SKAWLD",
			DetectedAt: now.Add(time.Duration(index) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	page1, hasMore, err := store.List(ctx, principal, incidentapp.Filter{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || !hasMore {
		t.Fatalf("page 1 = %d items, hasMore %v; want 2, true", len(page1), hasMore)
	}
	last := page1[len(page1)-1]
	cursor := last.DetectedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
	page2, hasMore2, err := store.List(ctx, principal, incidentapp.Filter{PageSize: 2, Cursor: cursor})
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
		t.Fatalf("distinct incidents across pages = %d, want 3", len(seen))
	}

	// The reporter defaults to the creating principal and the date defaults
	// to the detection time; a resolved incident exposes completion time.
	created, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: assetID,
		Summary: "Reporter default", Priority: "LOW",
		SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ReporterID != principalID {
		t.Fatalf("reporter = %q, want principal %q", created.ReporterID, principalID)
	}
	if created.OccurredAt == nil || !created.OccurredAt.Equal(now) {
		t.Fatalf("occurred_at = %v, want detection time", created.OccurredAt)
	}
	if created.TimeToCompleteSeconds != nil {
		t.Fatalf("open incident must not have completion time, got %v", *created.TimeToCompleteSeconds)
	}
	_, _, err = store.Resolve(ctx, principal, uuid.NewString(), created.ID, incidentapp.ResolveIncident{
		ExpectedVersion: created.Version, ResolutionSummary: "Verified fixed",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := store.Get(ctx, principal, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != "RESOLVED" || resolved.TimeToCompleteSeconds == nil {
		t.Fatalf("resolved incident missing completion time: %#v", resolved)
	}
}

func seedOrgSitePrincipal(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	organizationID, siteID, principalID string,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Pagination Test', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'PAGE', 'Pagination Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Pagination Tester', 'ACTIVE', $2, $2)
	`, principalID, now); err != nil {
		t.Fatal(err)
	}
}
