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
