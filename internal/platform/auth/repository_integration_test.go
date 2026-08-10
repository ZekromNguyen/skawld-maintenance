package auth

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestResolvePrincipalDoesNotMergeRolesAcrossOrganizations(t *testing.T) {
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

	now := time.Now().UTC()
	repository := Repository{
		Pool:  pool,
		IDs:   id.UUID{},
		Clock: clock.Fixed{Time: now},
	}
	claims := IdentityClaims{
		Subject:     "multi-organization-" + uuid.NewString(),
		DisplayName: "Multi Organization Test",
	}
	principal, err := repository.ResolvePrincipal(ctx, claims, nil)
	if err != nil {
		t.Fatal(err)
	}
	organizationOne := uuid.NewString()
	organizationTwo := uuid.NewString()
	siteOne := uuid.NewString()
	siteTwo := uuid.NewString()
	for _, value := range []struct {
		organizationID string
		siteID         string
		code           string
	}{
		{organizationOne, siteOne, "ONE"},
		{organizationTwo, siteTwo, "TWO"},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO organizations (
				id, name, source_of_truth, created_at, updated_at
			) VALUES ($1::uuid, $2, 'OWNED_BY_SKAWLD', $3, $3)
		`, value.organizationID, "Organization "+value.code, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO sites (
				id, organization_id, code, name, timezone, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3, $4, 'Asia/Ho_Chi_Minh', $5, $5
			)
		`, value.siteID, value.organizationID, value.code,
			"Organization "+value.code, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO memberships (
			id, principal_id, organization_id, site_id, role, created_at
		) VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'Technician', $5),
			($6::uuid, $2::uuid, $7::uuid, $8::uuid, 'Administrator', $9)
	`, uuid.NewString(), principal.ID, organizationOne, siteOne, now,
		uuid.NewString(), organizationTwo, siteTwo, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := repository.ResolvePrincipal(ctx, claims, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.OrganizationID != organizationOne {
		t.Fatalf("organization = %q, want %q", resolved.OrganizationID, organizationOne)
	}
	if len(resolved.SiteIDs) != 1 || resolved.SiteIDs[0] != siteOne {
		t.Fatalf("sites merged across organizations: %v", resolved.SiteIDs)
	}
	if len(resolved.Roles) != 1 ||
		resolved.Roles[0] != identitydomain.RoleTechnician {
		t.Fatalf("roles merged across organizations: %v", resolved.Roles)
	}
	if !resolved.Has(identitydomain.PermissionExecutionWrite) {
		t.Fatal("technician permission was not resolved")
	}
	if resolved.Has(identitydomain.PermissionWorkflowPublish) {
		t.Fatal("administrator permission leaked from another organization")
	}
}
