package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestScopes(t *testing.T) (context.Context, *pgxpool.Pool, string, string) {
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
	organizationID := uuid.NewString()
	siteID := uuid.NewString()
	now := time.Now().UTC()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO organizations (id, name, source_of_truth, created_at, updated_at)
			 VALUES ($1::uuid, 'Monitoring Scope', 'OWNED_BY_SKAWLD', $2, $2)`,
			args: []any{organizationID, now},
		},
		{
			sql: `INSERT INTO sites (id, organization_id, code, name, timezone, created_at, updated_at)
			 VALUES ($2::uuid, $1::uuid, 'MON', 'Monitoring Site', 'UTC', $3, $3)`,
			args: []any{organizationID, siteID, now},
		},
		{
			sql: `INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
			 VALUES ($1::uuid, 'monitor-' || $1::text || '@example.invalid', 'Monitor', 'ACTIVE', $2, $2)`,
			args: []any{siteID, now},
		},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return ctx, pool, organizationID, siteID
}

func TestUpsertMetricIsIdempotent(t *testing.T) {
	ctx, pool, orgID, siteID := newTestScopes(t)
	store := Store{Pool: pool}
	row := MetricRow{
		OrganizationID: orgID, SiteID: siteID,
		MetricKey: "sys.worker_failed_count", Dimensions: map[string]any{},
		Granularity: monitoringdomain.GranularityLatest,
		Value: 1, SampleCount: 1, CollectedAt: time.Now().UTC(),
	}
	if err := store.UpsertMetric(ctx, row); err != nil {
		t.Fatal(err)
	}
	row.Value = 2
	if err := store.UpsertMetric(ctx, row); err != nil {
		t.Fatal(err)
	}
	var count int
	var value float64
	err := pool.QueryRow(ctx, `
		SELECT count(*), max(value)::float8 FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'sys.worker_failed_count'
	`, orgID).Scan(&count, &value)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || value != 2 {
		t.Fatalf("expected 1 row with value 2, got count=%d value=%v", count, value)
	}
}

func TestUpsertOrgWideMetricIsIdempotent(t *testing.T) {
	ctx, pool, orgID, _ := newTestScopes(t)
	store := Store{Pool: pool}
	row := MetricRow{
		OrganizationID: orgID, // SiteID empty = org-wide
		MetricKey:      "sys.backup_freshness_hours", Dimensions: map[string]any{},
		Granularity: monitoringdomain.GranularityLatest,
		Value: 10, SampleCount: 1, CollectedAt: time.Now().UTC(),
	}
	if err := store.UpsertMetric(ctx, row); err != nil {
		t.Fatal(err)
	}
	row.Value = 12
	if err := store.UpsertMetric(ctx, row); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = 'sys.backup_freshness_hours'
	`, orgID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 org-wide row, got %d", count)
	}
}

func TestMetricsForRangeFiltersBySite(t *testing.T) {
	ctx, pool, orgID, siteID := newTestScopes(t)
	store := Store{Pool: pool}
	day := time.Now().UTC().Truncate(24 * time.Hour)
	for _, value := range []float64{1, 2} {
		if err := store.UpsertMetric(ctx, MetricRow{
			OrganizationID: orgID, SiteID: siteID,
			MetricKey: "ops.mttr_hours", Dimensions: map[string]any{},
			Granularity: monitoringdomain.GranularityDay, Day: &day,
			Value: value, SampleCount: 1, CollectedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.MetricsForRange(ctx, orgID, siteID, "ops.mttr_hours",
		time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// different site sees nothing
	rows, err = store.MetricsForRange(ctx, orgID, uuid.NewString(), "ops.mttr_hours",
		time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for other site, got %d", len(rows))
	}
}

func TestOpenAlertIsIdempotentAndResolveCloses(t *testing.T) {
	ctx, pool, orgID, siteID := newTestScopes(t)
	store := Store{Pool: pool}
	dimensions := map[string]any{"queue": "foundation"}
	for i := 0; i < 2; i++ {
		if err := store.OpenAlert(ctx, orgID, siteID, "sys.worker_failed_count",
			dimensions, monitoringdomain.StatusCrit, "worker failed", 1); err != nil {
			t.Fatal(err)
		}
	}
	var open int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM monitor_alert_events
		WHERE organization_id = $1::uuid AND resolved_at IS NULL
	`, orgID).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("expected 1 open alert, got %d", open)
	}
	if err := store.ResolveAlerts(ctx, orgID, siteID, "sys.worker_failed_count", dimensions); err != nil {
		t.Fatal(err)
	}
	var resolved int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM monitor_alert_events
		WHERE organization_id = $1::uuid AND resolved_at IS NOT NULL
	`, orgID).Scan(&resolved); err != nil {
		t.Fatal(err)
	}
	if resolved != 1 {
		t.Fatalf("expected 1 resolved alert, got %d", resolved)
	}
}

func TestThresholdUpsertAndListWithSiteOverride(t *testing.T) {
	ctx, pool, orgID, siteID := newTestScopes(t)
	store := Store{Pool: pool}
	orgThreshold := Threshold{
		ID: siteID, OrganizationID: orgID, MetricKey: "sys.worker_failed_count",
		Comparator: "GT", WarnValue: 0, CritValue: 0, Enabled: true,
	}
	if err := store.UpsertThreshold(ctx, orgThreshold); err != nil {
		t.Fatal(err)
	}
	siteThreshold := Threshold{
		ID: siteID, OrganizationID: orgID, SiteID: siteID, MetricKey: "sys.worker_failed_count",
		Comparator: "GT", WarnValue: 1, CritValue: 2, Enabled: true,
	}
	if err := store.UpsertThreshold(ctx, siteThreshold); err != nil {
		t.Fatal(err)
	}
	// upsert same key again keeps one row
	if err := store.UpsertThreshold(ctx, siteThreshold); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListThresholds(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 thresholds (org + site), got %d", len(rows))
	}
	bySite := map[string]Threshold{}
	for _, row := range rows {
		bySite[row.SiteID] = row
	}
	if bySite[siteID].CritValue != 2 {
		t.Fatalf("expected site override crit 2, got %+v", bySite[siteID])
	}
	if bySite[""].CritValue != 0 {
		t.Fatalf("expected org default crit 0, got %+v", bySite[""])
	}
}

func TestListAlertsReturnsOpenFirst(t *testing.T) {
	ctx, pool, orgID, siteID := newTestScopes(t)
	store := Store{Pool: pool}
	if err := store.OpenAlert(ctx, orgID, siteID, "sys.health_ready",
		map[string]any{}, monitoringdomain.StatusCrit, "backend not ready", 0); err != nil {
		t.Fatal(err)
	}
	openOnly, err := store.ListAlerts(ctx, orgID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(openOnly) != 1 || openOnly[0].MetricKey != "sys.health_ready" {
		t.Fatalf("expected 1 open alert, got %+v", openOnly)
	}
	if err := store.ResolveAlerts(ctx, orgID, siteID, "sys.health_ready", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	all, err := store.ListAlerts(ctx, orgID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ResolvedAt == nil {
		t.Fatalf("expected 1 resolved alert, got %+v", all)
	}
}
