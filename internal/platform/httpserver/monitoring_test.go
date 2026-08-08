package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	monitoringpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/adapter/postgres"
	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newMonitoringHandlerTestDB(t *testing.T) (context.Context, *pgxpool.Pool, string, string, identitydomain.Principal) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO organizations (id, name, source_of_truth, version, created_at, updated_at)
			 VALUES ($1::uuid, 'Monitoring Handler Org', 'OWNED_BY_SKAWLD', 1, $2, $2)`,
			args: []any{organizationID, now},
		},
		{
			sql: `INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
			 VALUES ($2::uuid, $1::uuid, 'MONH', 'Monitoring Handler Site', 'UTC', 'ACTIVE', 1, $3, $3)`,
			args: []any{organizationID, siteID, now},
		},
		{
			sql: `INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
			 VALUES ($1::uuid, 'monitoring-handler-' || $1::text, 'Monitor Handler', 'ACTIVE', $2, $2)`,
			args: []any{principalID, now},
		},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}
	return ctx, pool, organizationID, siteID, principal
}

func TestMonitoringSummaryRequiresAuthentication(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   rejectingAuth{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestMonitoringSummaryRequiresReadPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs:     []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestMonitoringSummaryHappyPathWithCrit(t *testing.T) {
	ctx, pool, orgID, _, principal := newMonitoringHandlerTestDB(t)
	store := monitoringpostgres.Store{Pool: pool}
	// health_ready = 0 with the default LT 1/1 threshold must be CRIT.
	if err := store.UpsertMetric(ctx, monitoringpostgres.MetricRow{
		OrganizationID: orgID, MetricKey: "sys.health_ready",
		Dimensions: map[string]any{}, Granularity: monitoringdomain.GranularityLatest,
		Value: 0, SampleCount: 1, CollectedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Database:   pool,
		Monitoring: store,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []monitoringSummaryEntry `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var health *monitoringSummaryEntry
	for i := range body.Items {
		if body.Items[i].MetricKey == "sys.health_ready" {
			health = &body.Items[i]
			break
		}
	}
	if health == nil {
		t.Fatalf("summary missing sys.health_ready; got %+v", body.Items)
	}
	if health.Status != "CRIT" || health.Value != 0 || health.Unit != "0/1" {
		t.Fatalf("health entry = %+v, want CRIT/0/0-1", health)
	}
}

func TestMonitoringMetricsRejectsUnknownKey(t *testing.T) {
	_, pool, _, _, principal := newMonitoringHandlerTestDB(t)
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Monitoring: monitoringpostgres.Store{Pool: pool},
	})
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/monitoring/metrics?metric_key=not.a.real.key", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMonitoringMetricsReturnsSeries(t *testing.T) {
	ctx, pool, orgID, _, principal := newMonitoringHandlerTestDB(t)
	store := monitoringpostgres.Store{Pool: pool}
	now := time.Now().UTC()
	day := now.Truncate(24 * time.Hour).Add(-24 * time.Hour)
	if err := store.UpsertMetric(ctx, monitoringpostgres.MetricRow{
		OrganizationID: orgID, MetricKey: "ops.incident_backlog_count",
		Dimensions:  map[string]any{"bucket": "24h"},
		Granularity: monitoringdomain.GranularityDay, Day: &day,
		Value: 3, SampleCount: 1, CollectedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Monitoring: store,
	})
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/monitoring/metrics?metric_key=ops.incident_backlog_count", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Items []monitoringpostgres.MetricRow `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Value != 3 {
		t.Fatalf("items = %+v, want 1 row value 3", body.Items)
	}
}

func TestMonitoringThresholdsUpsertAndSummaryReflectsIt(t *testing.T) {
	ctx, pool, orgID, _, principal := newMonitoringHandlerTestDB(t)
	store := monitoringpostgres.Store{Pool: pool}
	// a metric value that is PASS by default but CRIT under a configured threshold
	if err := store.UpsertMetric(ctx, monitoringpostgres.MetricRow{
		OrganizationID: orgID, MetricKey: "sys.worker_failed_count",
		Dimensions: map[string]any{}, Granularity: monitoringdomain.GranularityLatest,
		Value: 1, SampleCount: 1, CollectedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Monitoring: store,
	})

	// configure: warn at 0.5, crit at 0.8 (value 1 -> CRIT)
	body, _ := json.Marshal(map[string]any{
		"metric_key": "sys.worker_failed_count", "comparator": "GT",
		"warn_value": 0.5, "crit_value": 0.8, "enabled": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/monitoring/thresholds",
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204; body = %s", rec.Code, rec.Body.String())
	}

	// reject a threshold where crit < warn for GT
	bad, _ := json.Marshal(map[string]any{
		"metric_key": "sys.worker_failed_count", "comparator": "GT",
		"warn_value": 2, "crit_value": 1, "enabled": true,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/monitoring/thresholds",
		bytes.NewReader(bad))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT bad threshold status = %d, want 400", rec.Code)
	}

	// summary should now report CRIT for worker_failed_count
	req = httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/summary", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var summary struct {
		Items []monitoringSummaryEntry `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, entry := range summary.Items {
		if entry.MetricKey == "sys.worker_failed_count" {
			if entry.Status != "CRIT" {
				t.Fatalf("worker_failed_count status = %s, want CRIT", entry.Status)
			}
			return
		}
	}
	t.Fatalf("summary missing sys.worker_failed_count")
}

func TestMonitoringThresholdsRequiresConfigurePermission(t *testing.T) {
	permissions := map[identitydomain.Permission]struct{}{
		identitydomain.PermissionMonitoringRead: {},
	}
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs:     []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: permissions,
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
	})
	body, _ := json.Marshal(map[string]any{
		"metric_key": "sys.health_ready", "comparator": "LT",
		"warn_value": 1, "crit_value": 1, "enabled": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/monitoring/thresholds",
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
