package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rejectingAuth never injects a principal, exercising the 401 path.
type rejectingAuth struct{ fakeAuth }

func (rejectingAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "principal is missing")
	})
}

func TestSummaryRequiresAuthentication(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   rejectingAuth{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSummaryRequiresIncidentReadPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs:     []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestSummaryHappyPath(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Summary Handler Organization', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($2::uuid, $1::uuid, 'SUMMARY', 'Summary Handler Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, organizationID, siteID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Summary Handler Admin', 'ACTIVE', $2, $2)
	`, principalID, now); err != nil {
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
	// One open HIGH-severity incident so counts are non-zero.
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO incidents (
			id, organization_id, site_id, asset_id, number, summary, severity,
			state, source_of_truth, occurred_at, detected_at, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, 'IN-1', 'High vibration', 'HIGH',
			'OPEN', 'OWNED_BY_SKAWLD', $5, $5, 1, $5, $5
		)
	`, uuid.NewString(), organizationID, siteID, assetID, now); err != nil {
		t.Fatal(err)
	}

	handler := New(Dependencies{
		Logger:   slog.New(slog.DiscardHandler),
		Auth:     fakeAuth{principal: principal},
		Database: pool,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["open_incidents"] != float64(1) {
		t.Fatalf("open_incidents = %v, want 1", body["open_incidents"])
	}
	bySeverity, ok := body["by_severity"].(map[string]any)
	if !ok {
		t.Fatalf("by_severity missing or wrong type: %v", body["by_severity"])
	}
	for _, key := range []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"} {
		if _, present := bySeverity[key]; !present {
			t.Fatalf("by_severity missing key %s: %v", key, bySeverity)
		}
	}
	if bySeverity["HIGH"] != float64(1) {
		t.Fatalf("by_severity.HIGH = %v, want 1", bySeverity["HIGH"])
	}
}
