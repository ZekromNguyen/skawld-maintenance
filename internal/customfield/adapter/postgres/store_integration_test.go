package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	ctx := context.Background()
	now := time.Now().UTC()
	var orgID string
	err = pool.QueryRow(ctx, `
		INSERT INTO organizations (id, name, source_of_truth, version, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'OWNED_BY_SKAWLD', 1, $2, $2)
		RETURNING id::text
	`, "test-org-"+now.Format("150405.000000"), now).Scan(&orgID)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	return &Store{Pool: pool}, orgID
}

func TestStoreCRUDAndImmutability(t *testing.T) {
	store, orgID := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	created, err := store.Create(ctx, orgID, domain.Definition{
		OrganizationID: orgID, EntityType: "incident", Key: "po_number",
		Label: "PO Number", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 1,
		Status: domain.StatusActive, Version: 1,
	}, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated id")
	}

	listed, err := store.ListByEntity(ctx, orgID, "incident")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v items=%d", err, len(listed))
	}
	if listed[0].Key != "po_number" {
		t.Fatalf("unexpected definition: %+v", listed[0])
	}

	used, err := store.HasValues(ctx, orgID, created.ID)
	if err != nil {
		t.Fatalf("hasValues: %v", err)
	}
	if used {
		t.Fatal("expected no values yet")
	}

	updated, err := store.Update(ctx, orgID, created.ID, domain.Definition{
		ID: created.ID, OrganizationID: orgID, EntityType: "incident",
		Key: "po_number", Label: "Purchase Order Number", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 2, Version: 2, CreatedAt: created.CreatedAt,
	}, now)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Version != 2 || updated.Label != "Purchase Order Number" {
		t.Fatalf("unexpected updated definition: %+v", updated)
	}

	// Optimistic lock: stale version must conflict. The caller loaded the
	// definition before the first update (version 1), so it sends
	// Version=2 (current+1), which implies expected version 1 — but the
	// store is already at version 2.
	if _, err := store.Update(ctx, orgID, created.ID, domain.Definition{
		ID: created.ID, OrganizationID: orgID, EntityType: "incident",
		Key: "po_number", Label: "Stale", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 2, Version: 2, CreatedAt: created.CreatedAt,
	}, now); err != application.ErrConflict {
		t.Fatalf("stale version update must be ErrConflict, got %v", err)
	}

	retired, err := store.Retire(ctx, orgID, created.ID, now)
	if err != nil {
		t.Fatalf("retire: %v", err)
	}
	if retired.Status != domain.StatusRetired || retired.RetiredAt == nil {
		t.Fatalf("unexpected retired definition: %+v", retired)
	}

	// Double retire conflicts.
	if _, err := store.Retire(ctx, orgID, created.ID, now); err != application.ErrConflict {
		t.Fatalf("double retire must be ErrConflict, got %v", err)
	}

	// Retire keeps history intact: definition still listed.
	listed, err = store.ListByEntity(ctx, orgID, "incident")
	if err != nil || len(listed) != 1 || listed[0].Status != domain.StatusRetired {
		t.Fatalf("retired definition must remain listed: %v items=%d", err, len(listed))
	}

	// Update after retire preserves retired_at (regression for the service fix).
	kept, err := store.Update(ctx, orgID, created.ID, domain.Definition{
		ID: created.ID, OrganizationID: orgID, EntityType: "incident",
		Key: "po_number", Label: "PO (legacy)", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 2, Version: 4,
		CreatedAt: created.CreatedAt, RetiredAt: retired.RetiredAt,
	}, now)
	if err != nil {
		t.Fatalf("update after retire: %v", err)
	}
	if kept.RetiredAt == nil {
		t.Fatal("update after retire must preserve retired_at")
	}
}

func TestCountsReportsIncidentsWithValues(t *testing.T) {
	store, orgID := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	created, err := store.Create(ctx, orgID, domain.Definition{
		OrganizationID: orgID, EntityType: "incident", Key: "po_number",
		Label: "PO Number", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 1,
		Status: domain.StatusActive, Version: 1,
	}, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// A second definition that never holds a value must report 0, which
	// exercises the LEFT JOIN zero-count branch.
	unused, err := store.Create(ctx, orgID, domain.Definition{
		OrganizationID: orgID, EntityType: "incident", Key: "unused_field",
		Label: "Unused Field", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: false}, SortOrder: 2,
		Status: domain.StatusActive, Version: 1,
	}, now)
	if err != nil {
		t.Fatalf("create unused: %v", err)
	}

	var siteID, principalID string
	if err := store.Pool.QueryRow(ctx, `
		INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
		VALUES (gen_random_uuid(), $1::uuid, 'CF', 'CF Site', 'UTC', 'ACTIVE', 1, $2, $2)
		RETURNING id::text
	`, orgID, now).Scan(&siteID); err != nil {
		t.Fatal(err)
	}
	subject := "cf-tester-" + now.Format("150405.000000")
	if err := store.Pool.QueryRow(ctx, `
		INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'CF Tester', 'ACTIVE', $2, $2)
		RETURNING id::text
	`, subject, now).Scan(&principalID); err != nil {
		t.Fatal(err)
	}
	var assetID string
	if err := store.Pool.QueryRow(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class,
			status, source_of_truth, attributes, version, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1::uuid, $2::uuid, 'CF-ASSET', 'CF Asset', 'PUMP',
			'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $3, $3
		)
		RETURNING id::text
	`, orgID, siteID, now).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO incidents (
			id, organization_id, site_id, asset_id, number, summary, priority,
			status, source_of_truth, detected_at, version, created_by, created_at, updated_at,
			custom_values
		) VALUES (
			gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, 'INC-CF-1', 'CF', 'MEDIUM',
			'OPEN', 'OWNED_BY_SKAWLD', $4, 1, $5::uuid, $4, $4,
			jsonb_build_object($6::text, 1)
		)
	`, orgID, siteID, assetID, now, principalID, created.ID); err != nil {
		t.Fatal(err)
	}

	counts, err := store.Counts(ctx, orgID, "incident")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if counts[created.ID] != 1 {
		t.Fatalf("Counts[%s] = %d, want 1", created.ID, counts[created.ID])
	}
	if counts[unused.ID] != 0 {
		t.Fatalf("Counts[%s] = %d, want 0", unused.ID, counts[unused.ID])
	}
}
