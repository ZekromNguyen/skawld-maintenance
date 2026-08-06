package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/postgres"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sinkFixture struct {
	organizationID string
	siteID         string
	principalID    string
	principal      identitydomain.Principal
}

func seedSinkFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) sinkFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Import Integration Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'IMPORT', 'Import Integration Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Import Administrator', 'ACTIVE', $2, $2)
	`, principalID, now); err != nil {
		t.Fatal(err)
	}
	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	return sinkFixture{
		organizationID: organizationID, siteID: siteID, principalID: principalID,
		principal: identitydomain.Principal{
			ID: principalID, OrganizationID: organizationID,
			SiteIDs: []string{siteID}, Permissions: permissions,
		},
	}
}

func sinkFor(pool *pgxpool.Pool) integrationpostgres.Sink {
	return integrationpostgres.Sink{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
}

func externalAsset(fixture sinkFixture, externalID string) integrationdomain.ExternalRecord {
	return integrationdomain.ExternalRecord{
		Kind:            "ASSET",
		OrganizationID:  fixture.organizationID,
		SiteID:          fixture.siteID,
		ExternalSystem:  "CMMS-X",
		ExternalID:      externalID,
		ExternalVersion: "v1",
		ObservedAt:      time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC),
		Attributes: map[string]interface{}{
			"tag":          "P-302",
			"name":         "Centrifugal Pump P-302",
			"asset_class":  "PUMP",
			"manufacturer": "Grundfos",
			"model":        "CR-95",
			"status":       "ACTIVE",
		},
	}
}

func openSinkTestPool(t *testing.T) (*pgxpool.Pool, context.Context) {
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

func TestSinkInsertsExternalAsset(t *testing.T) {
	pool, ctx := openSinkTestPool(t)
	fixture := seedSinkFixture(t, ctx, pool)
	record := externalAsset(fixture, "a-1001")
	record.Attributes["tag"] = "P-302"
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{record}); err != nil {
		t.Fatal(err)
	}
	var sourceOfTruth, externalSystem, externalID, externalVersion, syncStatus string
	err := pool.QueryRow(ctx, `
		SELECT source_of_truth, external_system, external_id, external_version, sync_status
		FROM assets
		WHERE organization_id = $1::uuid AND external_system = $2 AND external_id = $3
	`, fixture.organizationID, "CMMS-X", "a-1001").
		Scan(&sourceOfTruth, &externalSystem, &externalID, &externalVersion, &syncStatus)
	if err != nil {
		t.Fatal(err)
	}
	if sourceOfTruth != "EXTERNAL_REFERENCE" || externalSystem != "CMMS-X" ||
		externalID != "a-1001" || externalVersion != "v1" || syncStatus != "IN_SYNC" {
		t.Fatalf("projected asset = %q %q %q %q %q",
			sourceOfTruth, externalSystem, externalID, externalVersion, syncStatus)
	}
}

func TestSinkPreservesOwnedBySkawld(t *testing.T) {
	pool, ctx := openSinkTestPool(t)
	fixture := seedSinkFixture(t, ctx, pool)
	now := time.Now().UTC().Truncate(time.Second)
	ownedID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class,
			manufacturer, model, status, source_of_truth,
			attributes, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'P-302-OWNED', 'Owned Pump', 'PUMP',
			'Vendor', 'Model-A', 'ACTIVE', 'OWNED_BY_SKAWLD',
			'{}'::jsonb, 1, $4, $4
		)
	`, ownedID, fixture.organizationID, fixture.siteID, now); err != nil {
		t.Fatal(err)
	}
	record := externalAsset(fixture, "a-2001")
	record.Attributes["tag"] = "P-302-EXT"
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{record}); err != nil {
		t.Fatal(err)
	}
	// The owned record must be untouched: same version and name.
	var name string
	var version int64
	if err := pool.QueryRow(ctx, `
		SELECT name, version FROM assets WHERE id = $1::uuid
	`, ownedID).Scan(&name, &version); err != nil {
		t.Fatal(err)
	}
	if name != "Owned Pump" || version != 1 {
		t.Fatalf("owned asset mutated: name %q version %d", name, version)
	}
}

func TestSinkUpsertByVersion(t *testing.T) {
	pool, ctx := openSinkTestPool(t)
	fixture := seedSinkFixture(t, ctx, pool)
	record := externalAsset(fixture, "a-3001")
	record.Attributes["tag"] = "P-302"
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{record}); err != nil {
		t.Fatal(err)
	}
	// Same version is a no-op (idempotent replay).
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{record}); err != nil {
		t.Fatal(err)
	}
	var version int64
	if err := pool.QueryRow(ctx, `
		SELECT version FROM assets
		WHERE organization_id = $1::uuid AND external_system = 'CMMS-X' AND external_id = 'a-3001'
	`, fixture.organizationID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("version after replay = %d, want 1", version)
	}
	// New version updates the row and bumps version.
	updated := record
	updated.ExternalVersion = "v2"
	updated.Attributes["name"] = "Centrifugal Pump P-302 revised"
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{updated}); err != nil {
		t.Fatal(err)
	}
	var externalVersion, name string
	if err := pool.QueryRow(ctx, `
		SELECT external_version, name FROM assets
		WHERE organization_id = $1::uuid AND external_system = 'CMMS-X' AND external_id = 'a-3001'
	`, fixture.organizationID).Scan(&externalVersion, &name); err != nil {
		t.Fatal(err)
	}
	if externalVersion != "v2" || name != "Centrifugal Pump P-302 revised" {
		t.Fatalf("updated asset = %q %q", externalVersion, name)
	}
	if err := pool.QueryRow(ctx, `
		SELECT version FROM assets
		WHERE organization_id = $1::uuid AND external_system = 'CMMS-X' AND external_id = 'a-3001'
	`, fixture.organizationID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("version after update = %d, want 2", version)
	}
}

func TestSinkRejectsUnsupportedKind(t *testing.T) {
	pool, ctx := openSinkTestPool(t)
	fixture := seedSinkFixture(t, ctx, pool)
	record := externalAsset(fixture, "a-4001")
	record.Kind = "WORK_REFERENCE"
	if err := sinkFor(pool).Apply(ctx, fixture.principal, []integrationdomain.ExternalRecord{record}); err == nil {
		t.Fatal("expected unsupported kind to be rejected")
	}
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM assets WHERE organization_id = $1::uuid
	`, fixture.organizationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rejected page inserted %d assets, want 0", count)
	}
}
