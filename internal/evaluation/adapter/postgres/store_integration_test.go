package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCountsReturnsEmptyScopedPilotSummary(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	organizationID := uuid.NewString()
	siteID := uuid.NewString()
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, created_at, updated_at
		) VALUES ($1::uuid, 'Evaluation Scope', 'OWNED_BY_SKAWLD', $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, created_at, updated_at
		) VALUES (
			$2::uuid, $1::uuid, 'EVAL', 'Evaluation Site', 'UTC', $3, $3
		)
	`, organizationID, siteID, now)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := (Store{Pool: pool}).Counts(
		ctx,
		identitydomain.Principal{
			OrganizationID: organizationID,
			SiteIDs:        []string{siteID},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Recommendations != 0 || counts.AICalls != 0 ||
		counts.WorkflowEvaluations != 0 {
		t.Fatalf("counts = %+v", counts)
	}
}
