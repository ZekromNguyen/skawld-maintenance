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

// federatedTestRepo builds a Repository with the allowlist gate active.
func federatedTestRepo(t *testing.T, now time.Time) (Repository, *pgxpool.Pool, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// A throwaway organization for federated provisioning assertions.
	orgID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (id, name, source_of_truth, created_at, updated_at)
		VALUES ($1::uuid, 'Federated Test Org', 'OWNED_BY_SKAWLD', $2, $2)
	`, orgID, now); err != nil {
		t.Fatalf("seed federated org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM organizations WHERE id = $1::uuid`, orgID)
	})

	repo := Repository{
		Pool:                 pool,
		IDs:                  id.UUID{},
		Clock:                clock.Fixed{Time: now},
		EmailDomainAllowlist: []string{"acme-industrial.com"},
		FederatedOrgID:       orgID,
	}
	return repo, pool, orgID
}

func federatedClaims(subject, email, role string) IdentityClaims {
	return IdentityClaims{
		Subject:     subject,
		DisplayName: "Federated User",
		Email:       email,
		Role:        role,
	}
}

func TestFederatedPrincipalWithDisallowedDomainIsRejected(t *testing.T) {
	now := time.Now().UTC()
	repo, pool, _ := federatedTestRepo(t, now)
	subject := "federated-blocked-" + uuid.NewString()
	// Ensure cleanup even on failure paths.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM principals WHERE external_subject = $1`, subject)
	})

	_, err := repo.ResolvePrincipal(
		context.Background(),
		federatedClaims(subject, "intruder@other-corp.com", "Technician"),
		nil,
	)
	if err == nil {
		t.Fatal("expected error for disallowed federated email domain, got nil")
	}
}

func TestFederatedPrincipalWithAllowlistedDomainAndRoleIsProvisioned(t *testing.T) {
	now := time.Now().UTC()
	repo, pool, orgID := federatedTestRepo(t, now)
	subject := "federated-allowed-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM principals WHERE external_subject = $1`, subject)
	})

	principal, err := repo.ResolvePrincipal(
		context.Background(),
		federatedClaims(subject, "engineer@acme-industrial.com", "Technician"),
		nil,
	)
	if err != nil {
		t.Fatalf("ResolvePrincipal: %v", err)
	}
	if principal.OrganizationID != orgID {
		t.Fatalf("organization = %q, want %q", principal.OrganizationID, orgID)
	}
	if len(principal.Roles) != 1 || principal.Roles[0] != identitydomain.RoleTechnician {
		t.Fatalf("roles = %v, want [Technician]", principal.Roles)
	}
	if !principal.Has(identitydomain.PermissionExecutionWrite) {
		t.Fatal("technician permission not granted from federated role claim")
	}
	if principal.Has(identitydomain.PermissionWorkflowPublish) {
		t.Fatal("administrator permission leaked from federated role claim")
	}

	// Membership row must exist with a NULL site.
	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM memberships
		WHERE principal_id = $1::uuid AND organization_id = $2::uuid
		  AND site_id IS NULL AND role = 'Technician'
	`, principal.ID, orgID).Scan(&count); err != nil {
		t.Fatalf("count membership: %v", err)
	}
	if count != 1 {
		t.Fatalf("membership count = %d, want 1", count)
	}
}

func TestFederatedProvisioningIsIdempotentAcrossResolutions(t *testing.T) {
	now := time.Now().UTC()
	repo, pool, orgID := federatedTestRepo(t, now)
	subject := "federated-idem-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM principals WHERE external_subject = $1`, subject)
	})
	claims := federatedClaims(subject, "lead@acme-industrial.com", "Senior Technician")

	first, err := repo.ResolvePrincipal(context.Background(), claims, nil)
	if err != nil {
		t.Fatalf("first ResolvePrincipal: %v", err)
	}
	second, err := repo.ResolvePrincipal(context.Background(), claims, nil)
	if err != nil {
		t.Fatalf("second ResolvePrincipal: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("principal id changed across resolutions: %q -> %q", first.ID, second.ID)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM memberships
		WHERE principal_id = $1::uuid AND organization_id = $2::uuid
		  AND site_id IS NULL AND role = 'Senior Technician'
	`, first.ID, orgID).Scan(&count); err != nil {
		t.Fatalf("count membership: %v", err)
	}
	if count != 1 {
		t.Fatalf("membership count = %d, want 1 (idempotent)", count)
	}
	if len(second.Roles) != 1 {
		t.Fatalf("roles = %v, want exactly one role", second.Roles)
	}
}

func TestFederatedPrincipalWithoutRoleAttributeHasNoPermissions(t *testing.T) {
	now := time.Now().UTC()
	repo, _, _ := federatedTestRepo(t, now)
	subject := "federated-norole-" + uuid.NewString()

	principal, err := repo.ResolvePrincipal(
		context.Background(),
		federatedClaims(subject, "guest@acme-industrial.com", ""),
		nil,
	)
	if err != nil {
		t.Fatalf("ResolvePrincipal: %v", err)
	}
	// Allowlisted but no skawld_role: authenticate with no permissions,
	// never with a role invented from nothing.
	if len(principal.Roles) != 0 {
		t.Fatalf("roles = %v, want none", principal.Roles)
	}
}

func TestBootstrapSubjectExemptFromFederatedGate(t *testing.T) {
	now := time.Now().UTC()
	repo, pool, _ := federatedTestRepo(t, now)
	subject := "bootstrap-" + uuid.NewString()
	bootstrap := map[string]struct{}{subject: {}}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM principals WHERE external_subject = $1`, subject)
	})

	// Bootstrap subject with a disallowed domain must still resolve and
	// receive the bootstrap permission.
	principal, err := repo.ResolvePrincipal(
		context.Background(),
		federatedClaims(subject, "admin@not-allowlisted.com", "Administrator"),
		bootstrap,
	)
	if err != nil {
		t.Fatalf("bootstrap subject rejected: %v", err)
	}
	if !principal.Has(identitydomain.PermissionOrganizationCreate) {
		t.Fatal("bootstrap permission not granted")
	}
}
