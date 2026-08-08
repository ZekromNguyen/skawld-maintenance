package river

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	monitoringpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/adapter/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newCollectorScopes(t *testing.T) (context.Context, monitoringpostgres.Store, string, string, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	organizationID := uuid.NewString()
	siteID := uuid.NewString()
	assetID := uuid.NewString()
	now := time.Now().UTC()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO organizations (id, name, source_of_truth, created_at, updated_at)
			 VALUES ($1::uuid, 'Collector Scope', 'OWNED_BY_SKAWLD', $2, $2)`,
			args: []any{organizationID, now},
		},
		{
			sql: `INSERT INTO sites (id, organization_id, code, name, timezone, created_at, updated_at)
			 VALUES ($2::uuid, $1::uuid, 'COL', 'Collector Site', 'UTC', $3, $3)`,
			args: []any{organizationID, siteID, now},
		},
		{
			sql: `INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
			 VALUES ($1::uuid, 'collector-' || $1::text || '@example.invalid', 'Collector', 'ACTIVE', $2, $2)`,
			args: []any{siteID, now},
		},
		{
			sql: `INSERT INTO assets (id, organization_id, site_id, tag, name, asset_class, source_of_truth, version, attributes, created_at, updated_at)
			 VALUES ($4::uuid, $1::uuid, $2::uuid, 'P-COL', 'Collector Pump', 'CENTRIFUGAL_PUMP', 'OWNED_BY_SKAWLD', 1, '{}'::jsonb, $3, $3)`,
			args: []any{organizationID, siteID, now, assetID},
		},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return ctx, monitoringpostgres.Store{Pool: pool}, organizationID, siteID, assetID
}

func insertIncident(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID, siteID, assetID, severity, state string,
	detectedAt, resolvedAt *time.Time,
) {
	t.Helper()
	var resolved any
	if resolvedAt != nil {
		resolved = *resolvedAt
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO incidents (id, organization_id, site_id, asset_id, number, summary,
			severity, state, source_of_truth, detected_at, resolved_at, version, created_by, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'COL-' || left($1::text, 8), 'Collector incident',
			$5, $6, 'OWNED_BY_SKAWLD', $7, $8, 1, $3::uuid, now(), now())
	`, uuid.NewString(), orgID, siteID, assetID, severity, state, detectedAt, resolved)
	if err != nil {
		t.Fatalf("seed incident: %v", err)
	}
}

func insertDomainEvent(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID, siteID, eventType string,
	payload map[string]any,
	occurredAt time.Time,
) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO domain_events (id, organization_id, site_id, event_type,
			aggregate_type, aggregate_id, aggregate_version, payload, occurred_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'maintenance_execution', $1::uuid, 1, $5::jsonb, $6)
	`, uuid.NewString(), orgID, siteID, eventType, payload, occurredAt)
	if err != nil {
		t.Fatalf("seed domain event: %v", err)
	}
}

func TestCollectDailyWritesOperationalMetrics(t *testing.T) {
	ctx, store, orgID, siteID, assetID := newCollectorScopes(t)
	now := time.Now().UTC()
	targetStart, _ := dayWindow(now)

	// one incident resolved 2 days after detection, resolved inside the window
	resolved := targetStart.Add(2 * time.Hour)
	detected := resolved.Add(-48 * time.Hour)
	insertIncident(t, ctx, store.Pool, orgID, siteID, assetID, "HIGH", "RESOLVED", &detected, &resolved)
	// one open incident detected 3 days ago
	openDetected := now.Add(-72 * time.Hour)
	insertIncident(t, ctx, store.Pool, orgID, siteID, assetID, "MEDIUM", "OPEN", &openDetected, nil)

	// one LOTO block inside the window
	insertDomainEvent(t, ctx, store.Pool, orgID, siteID, "execution.step.blocked",
		map[string]any{"value": map[string]any{"key": "intrusive_bearing_inspection"}},
		targetStart.Add(30*time.Minute))

	if err := CollectDaily(ctx, store, orgID, []string{siteID}, now); err != nil {
		t.Fatal(err)
	}

	assertMetric(t, ctx, store, orgID, siteID, "ops.loto_blocked_count", 1)  // per-site row
	assertMetric(t, ctx, store, orgID, "", "ops.incident_max_age_hours", 72) // org-wide row: max = 3-day-old open incident
	assertMetric(t, ctx, store, orgID, "", "ops.incident_backlog_count", 1)  // org-wide row

	// mttr: one resolved HIGH incident of exactly 48h
	var mttr float64
	var count int64
	err := store.Pool.QueryRow(ctx, `
		SELECT value::float8, sample_count FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'ops.mttr_hours'
		  AND dimensions->>'severity' = 'HIGH'
	`, orgID).Scan(&mttr, &count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || math.Abs(mttr-48) > 1 {
		t.Fatalf("expected mttr 48h (count 1), got value=%v count=%d", mttr, count)
	}
}

func TestCollectSnapshotWritesSystemMetrics(t *testing.T) {
	ctx, store, orgID, _, _ := newCollectorScopes(t)
	now := time.Now().UTC()

	// a retryable river job in the foundation queue
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO river.river_job (state, queue, kind, args, max_attempts)
		VALUES ('retryable', 'foundation', 'monitoring.snapshot', '{}'::jsonb, 3)
	`); err != nil {
		t.Fatal(err)
	}
	// a successful backup 12 hours ago
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO monitoring_backup_runs (id, organization_id, started_at, completed_at, status, size_bytes)
		VALUES ($1::uuid, $2::uuid, $3, $3, 'SUCCESS', 1024)
	`, uuid.NewString(), orgID, now.Add(-12*time.Hour)); err != nil {
		t.Fatal(err)
	}

	if err := CollectSnapshot(ctx, store, orgID, nil, now); err != nil {
		t.Fatal(err)
	}

	var retries int64
	if err := store.Pool.QueryRow(ctx, `
		SELECT count(*) FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'sys.worker_retry_count'
		  AND dimensions->>'queue' = 'foundation'
	`, orgID).Scan(&retries); err != nil {
		t.Fatal(err)
	}
	if retries != 1 {
		t.Fatalf("expected 1 retry metric row, got %d", retries)
	}
	var freshness float64
	err := store.Pool.QueryRow(ctx, `
		SELECT value::float8 FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'sys.backup_freshness_hours'
	`, orgID).Scan(&freshness)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(freshness-12) > 0.5 {
		t.Fatalf("expected freshness ~12h, got %v", freshness)
	}
	var ready float64
	if err := store.Pool.QueryRow(ctx, `
		SELECT value::float8 FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'sys.health_ready'
	`, orgID).Scan(&ready); err != nil {
		t.Fatal(err)
	}
	if ready != 1 {
		t.Fatalf("expected health_ready 1, got %v", ready)
	}
}

func TestCollectSnapshotOpensCritAlertForWorkerFailures(t *testing.T) {
	ctx, store, orgID, _, _ := newCollectorScopes(t)
	now := time.Now().UTC()
	// worker_failed_count default is enabled and any value > 0 is CRIT

	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO river.river_job (state, queue, kind, args, max_attempts, finalized_at)
		VALUES ('cancelled', 'foundation', 'monitoring.snapshot', '{}'::jsonb, 3, now())
	`); err != nil {
		t.Fatal(err)
	}

	if err := CollectSnapshot(ctx, store, orgID, nil, now); err != nil {
		t.Fatal(err)
	}

	var open int
	err := store.Pool.QueryRow(ctx, `
		SELECT count(*) FROM monitor_alert_events
		WHERE organization_id = $1::uuid AND metric_key = 'sys.worker_failed_count'
		  AND state = 'CRIT' AND resolved_at IS NULL
	`, orgID).Scan(&open)
	if err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("expected 1 open CRIT alert, got %d", open)
	}

	// second collection does not duplicate the alert (idempotent)
	if err := CollectSnapshot(ctx, store, orgID, nil, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Pool.QueryRow(ctx, `
		SELECT count(*) FROM monitor_alert_events
		WHERE organization_id = $1::uuid AND metric_key = 'sys.worker_failed_count'
		  AND state = 'CRIT' AND resolved_at IS NULL
	`, orgID).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("expected alert to stay single, got %d", open)
	}
}

func TestListScopesReturnsOrgWithSites(t *testing.T) {
	ctx, store, orgID, siteID, _ := newCollectorScopes(t)
	scopes, err := store.ListScopes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, scope := range scopes {
		if scope.OrganizationID == orgID {
			found = true
			if len(scope.SiteIDs) != 1 || scope.SiteIDs[0] != siteID {
				t.Fatalf("expected site %s, got %v", siteID, scope.SiteIDs)
			}
		}
	}
	if !found {
		t.Fatalf("scope for org %s not found", orgID)
	}
}

func assertMetric(t *testing.T, ctx context.Context, store monitoringpostgres.Store, orgID, siteID, key string, expected float64) {
	t.Helper()
	var value float64
	err := store.Pool.QueryRow(ctx, `
		SELECT value::float8 FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = $2
		  AND (nullif($3, '') IS NULL OR site_id = $3::uuid)
	`, orgID, key, siteID).Scan(&value)
	if err != nil {
		t.Fatalf("metric %s: %v", key, err)
	}
	if math.Abs(value-expected) > 0.01 {
		t.Fatalf("metric %s = %v, want %v", key, value, expected)
	}
}
