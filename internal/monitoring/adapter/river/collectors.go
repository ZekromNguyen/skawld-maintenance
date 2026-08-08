package river

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	monitoringpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/adapter/postgres"
	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/jackc/pgx/v5"
)

// sample is one metric value to persist, scoped to a site ("" = org-wide).
type sample struct {
	SiteID     string
	Dimensions map[string]any
	Value      float64
	Count      int64
}

// emit persists one metric row per sample and evaluates thresholds for it.
func emit(
	ctx context.Context,
	store monitoringpostgres.Store,
	orgID, key string,
	granularity monitoringdomain.Granularity,
	day *time.Time,
	now time.Time,
	samples []sample,
) error {
	for _, s := range samples {
		if err := store.UpsertMetric(ctx, monitoringpostgres.MetricRow{
			OrganizationID: orgID, SiteID: s.SiteID,
			MetricKey: key, Dimensions: s.Dimensions,
			Granularity: granularity, Day: day,
			Value: s.Value, SampleCount: s.Count, CollectedAt: now,
		}); err != nil {
			return err
		}
		if err := EvaluateAndAlert(ctx, store, orgID, s.SiteID, key, s.Dimensions, s.Value); err != nil {
			return err
		}
	}
	return nil
}

// CollectSnapshot gathers the system/infra metrics (group C) for one
// organization and evaluates their thresholds. Worker-health metrics are
// global infra (river jobs carry no org scope); in the single-organization
// pilot they are persisted org-wide per organization.
func CollectSnapshot(
	ctx context.Context,
	store monitoringpostgres.Store,
	orgID string,
	_ []string,
	now time.Time,
) error {
	collectors := []func(context.Context, monitoringpostgres.Store, string, time.Time) error{
		collectWorkerRetries,
		collectWorkerFailed,
		collectWorkerQueueDepth,
		collectIngestionFailures,
		collectBackupFreshness,
		collectHealthReady,
	}
	for _, collect := range collectors {
		if err := collect(ctx, store, orgID, now); err != nil {
			return fmt.Errorf("snapshot collector: %w", err)
		}
	}
	return nil
}

func collectWorkerRetries(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	rows, err := store.Pool.Query(ctx, `
		SELECT queue, count(*) FROM river.river_job
		WHERE state::text = 'retryable'
		GROUP BY queue
	`)
	if err != nil {
		return fmt.Errorf("worker retries: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var queue string
		var count int64
		if err := rows.Scan(&queue, &count); err != nil {
			return fmt.Errorf("scan worker retries: %w", err)
		}
		samples = append(samples, sample{
			SiteID: "", Dimensions: map[string]any{"queue": queue},
			Value: float64(count), Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "sys.worker_retry_count",
		monitoringdomain.GranularityLatest, nil, now, samples)
}

func collectWorkerFailed(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	var count int64
	err := store.Pool.QueryRow(ctx, `
		SELECT count(*) FROM river.river_job
		WHERE state::text = 'cancelled'
		  AND finalized_at >= now() - interval '24 hours'
	`).Scan(&count)
	if err != nil {
		return fmt.Errorf("worker failed: %w", err)
	}
	return emit(ctx, store, orgID, "sys.worker_failed_count",
		monitoringdomain.GranularityLatest, nil, now,
		[]sample{{SiteID: "", Dimensions: map[string]any{}, Value: float64(count), Count: count}})
}

func collectWorkerQueueDepth(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	rows, err := store.Pool.Query(ctx, `
		SELECT queue, count(*) FROM river.river_job
		WHERE state::text IN ('available', 'scheduled', 'retryable', 'running')
		GROUP BY queue
	`)
	if err != nil {
		return fmt.Errorf("worker queue depth: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var queue string
		var count int64
		if err := rows.Scan(&queue, &count); err != nil {
			return fmt.Errorf("scan worker queue depth: %w", err)
		}
		samples = append(samples, sample{
			SiteID: "", Dimensions: map[string]any{"queue": queue},
			Value: float64(count), Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "sys.worker_queue_depth",
		monitoringdomain.GranularityLatest, nil, now, samples)
}

func collectIngestionFailures(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	var count int64
	err := store.Pool.QueryRow(ctx, `
		SELECT count(*) FROM document_revisions
		WHERE organization_id = $1::uuid
		  AND ingestion_state = 'FAILED'
		  AND updated_at >= now() - interval '24 hours'
	`, orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("ingestion failures: %w", err)
	}
	return emit(ctx, store, orgID, "sys.ingestion_failed_count",
		monitoringdomain.GranularityLatest, nil, now,
		[]sample{{SiteID: "", Dimensions: map[string]any{}, Value: float64(count), Count: count}})
}

func collectBackupFreshness(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	var hours float64
	err := store.Pool.QueryRow(ctx, `
		SELECT extract(epoch FROM (now() - completed_at)) / 3600
		FROM monitoring_backup_runs
		WHERE organization_id = $1::uuid AND status = 'SUCCESS'
		ORDER BY completed_at DESC LIMIT 1
	`, orgID).Scan(&hours)
	if err != nil {
		// no successful backup recorded yet: nothing to report
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("backup freshness: %w", err)
	}
	return emit(ctx, store, orgID, "sys.backup_freshness_hours",
		monitoringdomain.GranularityLatest, nil, now,
		[]sample{{SiteID: "", Dimensions: map[string]any{}, Value: hours, Count: 1}})
}

func collectHealthReady(ctx context.Context, store monitoringpostgres.Store, orgID string, now time.Time) error {
	ready := 0.0
	if err := store.Pool.Ping(ctx); err == nil {
		ready = 1
	}
	return emit(ctx, store, orgID, "sys.health_ready",
		monitoringdomain.GranularityLatest, nil, now,
		[]sample{{SiteID: "", Dimensions: map[string]any{}, Value: ready, Count: 1}})
}

// CollectDaily gathers the operational KPI metrics (group A) for one
// organization. Windowed counts use the previous UTC day so the daily job
// (which runs just after midnight) covers a complete day.
func CollectDaily(
	ctx context.Context,
	store monitoringpostgres.Store,
	orgID string,
	siteIDs []string,
	now time.Time,
) error {
	collectors := []func(context.Context, monitoringpostgres.Store, string, []string, time.Time) error{
		collectLOTOBlocks,
		collectLOTOVerified,
		collectMTTR,
		collectIncidentAge,
		collectIncidentBacklog,
	}
	for _, collect := range collectors {
		if err := collect(ctx, store, orgID, siteIDs, now); err != nil {
			return fmt.Errorf("daily collector: %w", err)
		}
	}
	return nil
}

// dayWindow returns the target day (previous UTC day) and its [start, end).
func dayWindow(now time.Time) (time.Time, time.Time) {
	target := now.UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	return target, target.Add(24 * time.Hour)
}

func collectLOTOBlocks(ctx context.Context, store monitoringpostgres.Store, orgID string, siteIDs []string, now time.Time) error {
	start, end := dayWindow(now)
	rows, err := store.Pool.Query(ctx, `
		SELECT coalesce(site_id::text, ''), count(*)
		FROM domain_events
		WHERE organization_id = $1::uuid
		  AND event_type = 'execution.step.blocked'
		  AND occurred_at >= $2 AND occurred_at < $3
		  AND (cardinality($4::uuid[]) = 0 OR site_id = ANY($4::uuid[]))
		GROUP BY site_id
	`, orgID, start, end, siteIDs)
	if err != nil {
		return fmt.Errorf("loto blocks: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var siteID string
		var count int64
		if err := rows.Scan(&siteID, &count); err != nil {
			return fmt.Errorf("scan loto blocks: %w", err)
		}
		samples = append(samples, sample{
			SiteID: siteID, Dimensions: map[string]any{},
			Value: float64(count), Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "ops.loto_blocked_count",
		monitoringdomain.GranularityDay, &start, now, samples)
}

func collectLOTOVerified(ctx context.Context, store monitoringpostgres.Store, orgID string, siteIDs []string, now time.Time) error {
	start, end := dayWindow(now)
	rows, err := store.Pool.Query(ctx, `
		SELECT coalesce(site_id::text, ''), count(*)
		FROM domain_events
		WHERE organization_id = $1::uuid
		  AND event_type = 'execution.prerequisite.recorded'
		  AND payload->'value'->>'status' = 'VERIFIED'
		  AND occurred_at >= $2 AND occurred_at < $3
		  AND (cardinality($4::uuid[]) = 0 OR site_id = ANY($4::uuid[]))
		GROUP BY site_id
	`, orgID, start, end, siteIDs)
	if err != nil {
		return fmt.Errorf("loto verified: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var siteID string
		var count int64
		if err := rows.Scan(&siteID, &count); err != nil {
			return fmt.Errorf("scan loto verified: %w", err)
		}
		samples = append(samples, sample{
			SiteID: siteID, Dimensions: map[string]any{},
			Value: float64(count), Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "ops.loto_verified_count",
		monitoringdomain.GranularityDay, &start, now, samples)
}

func collectMTTR(ctx context.Context, store monitoringpostgres.Store, orgID string, siteIDs []string, now time.Time) error {
	start, end := dayWindow(now)
	rows, err := store.Pool.Query(ctx, `
		SELECT severity, avg(extract(epoch FROM (resolved_at - detected_at)) / 3600)::float8, count(*)
		FROM incidents
		WHERE organization_id = $1::uuid
		  AND resolved_at IS NOT NULL
		  AND resolved_at >= $2 AND resolved_at < $3
		  AND (cardinality($4::uuid[]) = 0 OR site_id = ANY($4::uuid[]))
		GROUP BY severity
	`, orgID, start, end, siteIDs)
	if err != nil {
		return fmt.Errorf("mttr: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var severity string
		var avg float64
		var count int64
		if err := rows.Scan(&severity, &avg, &count); err != nil {
			return fmt.Errorf("scan mttr: %w", err)
		}
		samples = append(samples, sample{
			SiteID: "", Dimensions: map[string]any{"severity": severity},
			Value: avg, Count: count,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "ops.mttr_hours",
		monitoringdomain.GranularityDay, &start, now, samples)
}

func collectIncidentAge(ctx context.Context, store monitoringpostgres.Store, orgID string, siteIDs []string, now time.Time) error {
	start, _ := dayWindow(now)
	rows, err := store.Pool.Query(ctx, `
		SELECT state, max(extract(epoch FROM (now() - detected_at)) / 3600)::float8
		FROM incidents
		WHERE organization_id = $1::uuid
		  AND state IN ('OPEN', 'IN_PROGRESS')
		  AND (cardinality($2::uuid[]) = 0 OR site_id = ANY($2::uuid[]))
		GROUP BY state
	`, orgID, siteIDs)
	if err != nil {
		return fmt.Errorf("incident age: %w", err)
	}
	defer rows.Close()
	var samples []sample
	for rows.Next() {
		var state string
		var maxAge float64
		if err := rows.Scan(&state, &maxAge); err != nil {
			return fmt.Errorf("scan incident age: %w", err)
		}
		samples = append(samples, sample{
			SiteID: "", Dimensions: map[string]any{"state": state},
			Value: maxAge, Count: 1,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return emit(ctx, store, orgID, "ops.incident_max_age_hours",
		monitoringdomain.GranularityDay, &start, now, samples)
}

func collectIncidentBacklog(ctx context.Context, store monitoringpostgres.Store, orgID string, siteIDs []string, now time.Time) error {
	start, _ := dayWindow(now)
	var samples []sample
	for _, bucket := range []struct {
		name  string
		hours int
	}{
		{name: "24h", hours: 24},
		{name: "7d", hours: 7 * 24},
	} {
		var count int64
		err := store.Pool.QueryRow(ctx, `
			SELECT count(*) FROM incidents
			WHERE organization_id = $1::uuid
			  AND state IN ('OPEN', 'IN_PROGRESS')
			  AND detected_at < now() - make_interval(hours => $2)
			  AND (cardinality($3::uuid[]) = 0 OR site_id = ANY($3::uuid[]))
		`, orgID, bucket.hours, siteIDs).Scan(&count)
		if err != nil {
			return fmt.Errorf("incident backlog %s: %w", bucket.name, err)
		}
		samples = append(samples, sample{
			SiteID: "", Dimensions: map[string]any{"bucket": bucket.name},
			Value: float64(count), Count: count,
		})
	}
	return emit(ctx, store, orgID, "ops.incident_backlog_count",
		monitoringdomain.GranularityDay, &start, now, samples)
}

// EvaluateAndAlert applies the effective threshold for a metric value and
// opens or resolves alert events. Effective threshold resolution: a site row
// in monitor_thresholds overrides the org-wide row; without any row the
// domain defaults apply, and a default with enabled=false never alerts.
func EvaluateAndAlert(
	ctx context.Context,
	store monitoringpostgres.Store,
	orgID, siteID, key string,
	dimensions map[string]any,
	value float64,
) error {
	thresholds, err := store.ListThresholds(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list thresholds: %w", err)
	}
	var warn, crit float64
	var comparator monitoringdomain.Comparator
	enabled := false
	configured := false
	for _, row := range thresholds {
		if row.MetricKey != key {
			continue
		}
		if row.SiteID == siteID && siteID != "" {
			warn, crit = row.WarnValue, row.CritValue
			comparator = monitoringdomain.Comparator(row.Comparator)
			enabled, configured = row.Enabled, true
			break
		}
		if row.SiteID == "" && !configured {
			warn, crit = row.WarnValue, row.CritValue
			comparator = monitoringdomain.Comparator(row.Comparator)
			enabled, configured = row.Enabled, true
		}
	}
	if !configured {
		var ok bool
		warn, crit, comparator, enabled, ok = monitoringdomain.DefaultThresholds(key)
		if !ok {
			return nil // key not in catalog: never alerts
		}
	}
	if !enabled {
		return nil
	}
	status := monitoringdomain.Evaluate(value, warn, crit, comparator)
	switch status {
	case monitoringdomain.StatusPass:
		return store.ResolveAlerts(ctx, orgID, siteID, key, dimensions)
	case monitoringdomain.StatusWarn:
		return store.OpenAlert(ctx, orgID, siteID, key, dimensions,
			monitoringdomain.StatusWarn, alertMessage(key, value), value)
	case monitoringdomain.StatusCrit:
		return store.OpenAlert(ctx, orgID, siteID, key, dimensions,
			monitoringdomain.StatusCrit, alertMessage(key, value), value)
	default:
		return nil
	}
}

func alertMessage(key string, value float64) string {
	encoded, _ := json.Marshal(map[string]any{"metric_key": key, "value": value})
	return string(encoded)
}
