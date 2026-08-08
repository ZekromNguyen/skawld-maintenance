package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// emptyUUID is the sentinel used in COALESCE-based unique indexes so that
// org-wide (site_id NULL) rows still participate in uniqueness.
const emptyUUID = "00000000-0000-0000-0000-000000000000"

type Store struct {
	Pool *pgxpool.Pool
}

type MetricRow struct {
	OrganizationID string
	SiteID         string
	MetricKey      string
	Dimensions     map[string]any
	Granularity    monitoringdomain.Granularity
	Day            *time.Time
	Value          float64
	SampleCount    int64
	CollectedAt    time.Time
}

func (s Store) UpsertMetric(ctx context.Context, row MetricRow) error {
	dimensions, err := json.Marshal(row.Dimensions)
	if err != nil {
		return fmt.Errorf("marshal dimensions: %w", err)
	}
	var day *time.Time
	if row.Granularity == monitoringdomain.GranularityDay && row.Day != nil {
		d := row.Day.UTC().Truncate(24 * time.Hour)
		day = &d
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO monitoring_metrics
			(organization_id, site_id, metric_key, dimensions, granularity, day, value, sample_count, collected_at)
		VALUES ($1::uuid, nullif($2, '')::uuid, $3, $4::jsonb, $5, $6::date, $7, $8, $9)
		ON CONFLICT (
			organization_id,
			(COALESCE(site_id, '00000000-0000-0000-0000-000000000000'::uuid)),
			metric_key, dimensions, granularity,
			(COALESCE(day, '1970-01-01'::date))
		)
		DO UPDATE SET value = EXCLUDED.value,
		              sample_count = EXCLUDED.sample_count,
		              collected_at = EXCLUDED.collected_at
	`, row.OrganizationID, row.SiteID, row.MetricKey, dimensions,
		row.Granularity, day, row.Value, row.SampleCount, row.CollectedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert metric %s: %w", row.MetricKey, err)
	}
	return nil
}

func (s Store) MetricsForRange(
	ctx context.Context,
	orgID, siteID, key string,
	from, to time.Time,
) ([]MetricRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT organization_id::text, coalesce(site_id::text, ''), metric_key,
		       dimensions, granularity, day, value::float8, sample_count, collected_at
		FROM monitoring_metrics
		WHERE organization_id = $1::uuid
		  AND metric_key = $2
		  AND (nullif($3, '') IS NULL OR site_id = $3::uuid)
		  AND collected_at >= $4
		  AND collected_at < $5
		ORDER BY day NULLS LAST, collected_at
	`, orgID, key, siteID, from.UTC(), to.UTC())
	if err != nil {
		return nil, fmt.Errorf("query metrics %s: %w", key, err)
	}
	defer rows.Close()
	var result []MetricRow
	for rows.Next() {
		var row MetricRow
		var dimensions []byte
		var granularity string
		if err := rows.Scan(
			&row.OrganizationID, &row.SiteID, &row.MetricKey, &dimensions,
			&granularity, &row.Day, &row.Value, &row.SampleCount, &row.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		row.Granularity = monitoringdomain.Granularity(granularity)
		if err := json.Unmarshal(dimensions, &row.Dimensions); err != nil {
			return nil, fmt.Errorf("unmarshal dimensions: %w", err)
		}
		if row.Dimensions == nil {
			row.Dimensions = map[string]any{}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

type Threshold struct {
	ID             string
	OrganizationID string
	SiteID         string
	MetricKey      string
	Comparator     string
	WarnValue      float64
	CritValue      float64
	Enabled        bool
	UpdatedAt      time.Time
}

func (s Store) ListThresholds(ctx context.Context, orgID string) ([]Threshold, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, organization_id::text, coalesce(site_id::text, ''),
		       metric_key, comparator, warn_value::float8, crit_value::float8,
		       enabled, updated_at
		FROM monitor_thresholds
		WHERE organization_id = $1::uuid
		ORDER BY metric_key, site_id NULLS FIRST
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list thresholds: %w", err)
	}
	defer rows.Close()
	var result []Threshold
	for rows.Next() {
		var value Threshold
		if err := rows.Scan(
			&value.ID, &value.OrganizationID, &value.SiteID, &value.MetricKey,
			&value.Comparator, &value.WarnValue, &value.CritValue,
			&value.Enabled, &value.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan threshold: %w", err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) UpsertThreshold(ctx context.Context, value Threshold) error {
	// value.ID is the acting principal's ID, required by updated_by.
	if value.ID == "" {
		return fmt.Errorf("upsert threshold %s: updated_by principal required", value.MetricKey)
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO monitor_thresholds
			(id, organization_id, site_id, metric_key, comparator, warn_value, crit_value, enabled, updated_by, updated_at)
		VALUES ($1::uuid, $2::uuid, nullif($3, '')::uuid, $4, $5, $6, $7, $8, $9::uuid, now())
		ON CONFLICT (
			organization_id,
			(COALESCE(site_id, '00000000-0000-0000-0000-000000000000'::uuid)),
			metric_key
		)
		DO UPDATE SET comparator = EXCLUDED.comparator,
		              warn_value = EXCLUDED.warn_value,
		              crit_value = EXCLUDED.crit_value,
		              enabled = EXCLUDED.enabled,
		              updated_by = EXCLUDED.updated_by,
		              updated_at = now()
	`, uuid.NewString(), value.OrganizationID, value.SiteID, value.MetricKey,
		value.Comparator, value.WarnValue, value.CritValue, value.Enabled, value.ID)
	if err != nil {
		return fmt.Errorf("upsert threshold %s: %w", value.MetricKey, err)
	}
	return nil
}

type AlertEvent struct {
	ID             int64
	OrganizationID string
	SiteID         string
	MetricKey      string
	Dimensions     map[string]any
	State          monitoringdomain.Status
	Message        string
	Value          float64
	OpenedAt       time.Time
	ResolvedAt     *time.Time
}

func (s Store) OpenAlert(
	ctx context.Context,
	orgID, siteID, key string,
	dimensions map[string]any,
	state monitoringdomain.Status,
	message string,
	value float64,
) error {
	encoded, err := json.Marshal(dimensions)
	if err != nil {
		return fmt.Errorf("marshal alert dimensions: %w", err)
	}
	// ON CONFLICT DO NOTHING relies on the partial unique index on active
	// events; a duplicate open alert is a no-op, making the open idempotent.
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO monitor_alert_events
			(organization_id, site_id, metric_key, dimensions, state, message, value)
		VALUES ($1::uuid, nullif($2, '')::uuid, $3, $4::jsonb, $5, $6, $7)
		ON CONFLICT DO NOTHING
	`, orgID, siteID, key, encoded, state, message, value)
	if err != nil {
		return fmt.Errorf("open alert %s: %w", key, err)
	}
	return nil
}

func (s Store) ResolveAlerts(
	ctx context.Context,
	orgID, siteID, key string,
	dimensions map[string]any,
) error {
	encoded, err := json.Marshal(dimensions)
	if err != nil {
		return fmt.Errorf("marshal alert dimensions: %w", err)
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE monitor_alert_events
		SET resolved_at = now()
		WHERE organization_id = $1::uuid
		  AND metric_key = $2
		  AND dimensions = $3::jsonb
		  AND resolved_at IS NULL
		  AND (nullif($4, '') IS NULL OR site_id = $4::uuid)
	`, orgID, key, encoded, siteID)
	if err != nil {
		return fmt.Errorf("resolve alerts %s: %w", key, err)
	}
	return nil
}

func (s Store) ListAlerts(ctx context.Context, orgID string, openOnly bool) ([]AlertEvent, error) {
	query := `
		SELECT id, organization_id::text, coalesce(site_id::text, ''), metric_key,
		       dimensions, state, message, value::float8, opened_at, resolved_at
		FROM monitor_alert_events
		WHERE organization_id = $1::uuid
	`
	if openOnly {
		query += " AND resolved_at IS NULL"
	}
	query += " ORDER BY opened_at DESC"
	rows, err := s.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()
	var result []AlertEvent
	for rows.Next() {
		var event AlertEvent
		var dimensions []byte
		if err := rows.Scan(
			&event.ID, &event.OrganizationID, &event.SiteID, &event.MetricKey,
			&dimensions, &event.State, &event.Message, &event.Value,
			&event.OpenedAt, &event.ResolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		if err := json.Unmarshal(dimensions, &event.Dimensions); err != nil {
			return nil, fmt.Errorf("unmarshal alert dimensions: %w", err)
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
