package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDocumentListCursorPagination(t *testing.T) {
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
	seedKnowledgePagination(t, ctx, pool, organizationID, siteID, principalID, now)

	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}
	store := Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{},
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	for index := 0; index < 3; index++ {
		if _, _, err := store.CreateDocument(ctx, principal, uuid.NewString(), knowledgeapp.CreateDocument{
			SiteID: siteID, DocumentType: knowledgedomain.DocumentSOP,
			Title:     "Pump SOP " + string(rune('0'+index)),
			Authority: knowledgedomain.AuthoritySiteApproved,
		}); err != nil {
			t.Fatal(err)
		}
	}
	page1, hasMore, err := store.ListDocuments(ctx, principal, knowledgeapp.Filter{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || !hasMore {
		t.Fatalf("page 1 = %d items, hasMore %v; want 2, true", len(page1), hasMore)
	}
	last := page1[len(page1)-1]
	cursor := last.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
	page2, hasMore2, err := store.ListDocuments(ctx, principal, knowledgeapp.Filter{PageSize: 2, Cursor: cursor})
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
		t.Fatalf("distinct documents across pages = %d, want 3", len(seen))
	}
}

func seedKnowledgePagination(
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
		) VALUES ($1::uuid, 'Knowledge Pagination', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'KPG', 'Knowledge Pagination Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Knowledge Pagination Tester', 'ACTIVE', $2, $2)
	`, principalID, now); err != nil {
		t.Fatal(err)
	}
}
