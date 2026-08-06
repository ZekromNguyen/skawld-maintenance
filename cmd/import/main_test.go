package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunImportProjectsAndIsIdempotent(t *testing.T) {
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
	subject := "import-admin-" + uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Import CLI Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'CLI', 'Import CLI Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $2, 'Import Administrator', 'ACTIVE', $3, $3)
	`, principalID, subject, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO memberships (
			id, principal_id, organization_id, site_id, role, created_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'Administrator', $5)
	`, uuid.NewString(), principalID, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}

	// The fixture uses tenant-neutral placeholders that runImport rewrites.
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "import-p302.ndjson")
	args := importArgs{
		Snapshot: fixturePath, SiteID: siteID, DatabaseURL: databaseURL,
		Limit: 100, ExternalSystem: "CMMS-X", ExternalSubject: subject,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := runImport(ctx, logger, args); err != nil {
		t.Fatal(err)
	}

	var externalAssets int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM assets
		WHERE organization_id = $1::uuid AND source_of_truth = 'EXTERNAL_REFERENCE'
	`, organizationID).Scan(&externalAssets); err != nil {
		t.Fatal(err)
	}
	if externalAssets != 2 {
		t.Fatalf("imported external assets = %d, want 2", externalAssets)
	}

	// A second run is idempotent: still two assets, versions unchanged.
	if err := runImport(ctx, logger, args); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM assets
		WHERE organization_id = $1::uuid AND source_of_truth = 'EXTERNAL_REFERENCE'
		  AND external_version = 'v1'
	`, organizationID).Scan(&externalAssets); err != nil {
		t.Fatal(err)
	}
	if externalAssets != 2 {
		t.Fatalf("external assets after replay = %d, want 2", externalAssets)
	}
}
