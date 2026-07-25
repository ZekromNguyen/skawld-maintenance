package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationCreateIsAtomicAndIdempotent(t *testing.T) {
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

	principalID := uuid.NewString()
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, created_at, updated_at
		) VALUES ($1::uuid, $2, 'Integration Test', $3, $3)
	`, principalID, "integration-"+principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	principal := domain.Principal{
		ID: principalID,
		Permissions: map[domain.Permission]struct{}{
			domain.PermissionOrganizationCreate: {},
		},
	}
	store := OrganizationStore{
		Pool:        pool,
		IDs:         id.UUID{},
		Clock:       clock.System{},
		Idempotency: idempotency.Store{},
		Audit:       audit.Sink{},
	}
	service := application.OrganizationService{Store: store}
	command := application.CreateOrganization{Name: "Concurrent Test " + principalID}
	key := "organization-" + principalID

	type outcome struct {
		result application.CreateOrganizationResult
		replay bool
		err    error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, replay, err := service.Create(ctx, principal, key, command)
			results <- outcome{result: result, replay: replay, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(results)

	var first application.CreateOrganizationResult
	replays := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if first.ID == "" {
			first = result.result
		}
		if result.result != first {
			t.Fatalf("idempotent result mismatch: %#v != %#v", result.result, first)
		}
		if result.replay {
			replays++
		}
	}
	if replays != 1 {
		t.Fatalf("replays = %d, want 1", replays)
	}

	_, _, err = service.Create(ctx, principal, key, application.CreateOrganization{
		Name: command.Name + " changed",
	})
	if !errors.Is(err, idempotency.ErrKeyConflict) {
		t.Fatalf("changed payload error = %v, want idempotency conflict", err)
	}

	var organizationCount, auditCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM organizations WHERE id = $1::uuid`, first.ID,
	).Scan(&organizationCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_events
		WHERE entity_kind = 'organization' AND entity_id = $1
	`, first.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if organizationCount != 1 || auditCount != 1 {
		t.Fatalf("organization count = %d, audit count = %d; want 1, 1",
			organizationCount, auditCount)
	}

	allowedSiteID := uuid.NewString()
	wrongSiteID := uuid.NewString()
	for siteID, code := range map[string]string{
		allowedSiteID: "ALLOWED",
		wrongSiteID:   "WRONG",
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO sites (
				id, organization_id, code, name, timezone, created_at, updated_at
			) VALUES ($1::uuid, $2::uuid, $3, $3, 'Asia/Ho_Chi_Minh', $4, $4)
		`, siteID, first.ID, code, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	reader := SiteReader{Pool: pool}
	sitePrincipal := principal
	sitePrincipal.OrganizationID = first.ID
	sitePrincipal.SiteIDs = []string{allowedSiteID}
	if _, err := reader.Get(ctx, sitePrincipal, allowedSiteID); err != nil {
		t.Fatalf("allowed site read failed: %v", err)
	}
	if _, err := reader.Get(ctx, sitePrincipal, wrongSiteID); !errors.Is(err, application.ErrSiteNotFound) {
		t.Fatalf("wrong site error = %v, want not found", err)
	}
}
