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

func TestIncidentCustomValuesIntegration(t *testing.T) {
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
			$1::uuid, $2::uuid, $3::uuid, 'P-303', 'Pump P-303', 'PUMP',
			'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4
		)
	`, assetID, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	var definitionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO field_definitions (
			organization_id, entity_type, key, label, field_type, config, status, version, created_at, updated_at
		) VALUES (
			$1::uuid, 'incident', 'po_number', 'PO Number', 'TEXT',
			'{"required": true}'::jsonb, 'ACTIVE', 1, $2, $2
		)
		RETURNING id::text
	`, organizationID, now).Scan(&definitionID); err != nil {
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

	created, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: assetID,
		Summary: "Custom values create", Priority: "MEDIUM",
		SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
		CustomValues: map[string]any{definitionID: "PO-1"},
	})
	if err != nil {
		t.Fatalf("create with custom values: %v", err)
	}
	if created.CustomValues[definitionID] != "PO-1" {
		t.Fatalf("create response custom_values = %#v", created.CustomValues)
	}

	// Create without custom values must store an empty object (regression
	// for the nil-marshal path) and must not fail the NOT NULL column.
	plain, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
		SiteID: siteID, AssetID: assetID,
		Summary: "Plain create", Priority: "LOW",
		SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
	})
	if err != nil {
		t.Fatalf("plain create: %v", err)
	}
	if plain.CustomValues == nil || len(plain.CustomValues) != 0 {
		t.Fatalf("plain create custom_values = %#v, want empty map", plain.CustomValues)
	}

	// No-op update (empty values) must not bump the version.
	unchanged, _, err := store.UpdateCustomValues(ctx, principal, uuid.NewString(), created.ID, incidentapp.UpdateCustomValues{
		ExpectedVersion: created.Version,
		CustomValues:    map[string]any{},
	})
	if err != nil {
		t.Fatalf("no-op update: %v", err)
	}
	if unchanged.Version != created.Version {
		t.Fatalf("no-op update bumped version: %v -> %v", created.Version, unchanged.Version)
	}

	// Merge keeps existing values and adds new ones, bumps version.
	updated, _, err := store.UpdateCustomValues(ctx, principal, uuid.NewString(), created.ID, incidentapp.UpdateCustomValues{
		ExpectedVersion: unchanged.Version,
		CustomValues:    map[string]any{definitionID: "PO-2"},
	})
	if err != nil {
		t.Fatalf("merge update: %v", err)
	}
	if updated.Version != unchanged.Version+1 {
		t.Fatalf("merge update version = %d, want %d", updated.Version, unchanged.Version+1)
	}
	if updated.CustomValues[definitionID] != "PO-2" {
		t.Fatalf("merged custom_values = %#v", updated.CustomValues)
	}

	// History rows: one for create, one for the merge (before PO-1, after PO-2).
	var historyCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM custom_value_history
		WHERE incident_id = $1::uuid AND field_definition_id = $2::uuid
	`, created.ID, definitionID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 {
		t.Fatalf("history rows = %d, want 2", historyCount)
	}

	// Custom-field filter on the list endpoint matches only the matching incident.
	matches, _, err := store.List(ctx, principal, incidentapp.Filter{
		CustomFields: map[string]string{definitionID: "PO-2"},
	})
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created.ID {
		t.Fatalf("filtered list = %d items, want 1 matching %s", len(matches), created.ID)
	}
}

func TestIncidentListNumberRangeFilterIntegration(t *testing.T) {
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
			$1::uuid, $2::uuid, $3::uuid, 'P-303', 'Pump P-303', 'PUMP',
			'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4
		)
	`, assetID, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	var definitionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO field_definitions (
			organization_id, entity_type, key, label, field_type, config, status, version, created_at, updated_at
		) VALUES (
			$1::uuid, 'incident', 'temperature', 'Temperature', 'NUMBER',
			'{"min": -40, "max": 200}'::jsonb, 'ACTIVE', 1, $2, $2
		)
		RETURNING id::text
	`, organizationID, now).Scan(&definitionID); err != nil {
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

	values := map[string]float64{"cold": 15, "warm": 25, "hot": 35}
	created := make(map[string]string, len(values))
	for summary, v := range values {
		incident, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
			SiteID: siteID, AssetID: assetID,
			Summary: summary, Priority: "MEDIUM",
			SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
			CustomValues: map[string]any{definitionID: v},
		})
		if err != nil {
			t.Fatalf("create %s: %v", summary, err)
		}
		created[summary] = incident.ID
	}

	// Range 20:30 must match only the 25 value.
	min, max := 20.0, 30.0
	matches, _, err := store.List(ctx, principal, incidentapp.Filter{
		CustomFieldRanges: map[string]incidentapp.CustomFieldRange{definitionID: {Min: &min, Max: &max}},
	})
	if err != nil {
		t.Fatalf("range list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created["warm"] {
		t.Fatalf("range list = %d items, want only warm", len(matches))
	}

	// Open-ended :20 must match only the 15 value.
	openMax := 20.0
	matches, _, err = store.List(ctx, principal, incidentapp.Filter{
		CustomFieldRanges: map[string]incidentapp.CustomFieldRange{definitionID: {Max: &openMax}},
	})
	if err != nil {
		t.Fatalf("open range list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created["cold"] {
		t.Fatalf("open range list = %d items, want only cold", len(matches))
	}

	// Min-only 30: must match only the 35 value.
	minOnly := 30.0
	matches, _, err = store.List(ctx, principal, incidentapp.Filter{
		CustomFieldRanges: map[string]incidentapp.CustomFieldRange{definitionID: {Min: &minOnly}},
	})
	if err != nil {
		t.Fatalf("min-only range list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created["hot"] {
		t.Fatalf("min-only range list = %d items, want only hot", len(matches))
	}
}
