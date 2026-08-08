# Monitoring Console (Phase 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/monitoring` console page plus a backend metrics pipeline so operators can see operational KPIs and system health with configurable thresholds and state-based alerts.

**Architecture:** A generic metrics store (`monitoring_metrics`) written by two River periodic jobs (5-minute snapshot for system health, daily rollup for operational KPIs); thresholds live in `monitor_thresholds` and are evaluated in the jobs against `monitor_alert_events`; the web console reads only the monitoring tables through four read-only endpoints, and shows alerts via a topbar bell.

**Tech Stack:** Go (chi, pgx, River), PostgreSQL (goose migrations), React 19 + Vite + Vitest, i18n en/vi.

## Global Constraints

- Module paths: `github.com/ZekromNguyen/skawld-maintenance/...`.
- Migrations are goose format (`-- +goose Up` / `-- +goose Down`) registered via `//go:embed` in `migrations/embed.go`; new file number is `00013_monitoring.sql`.
- River periodic jobs use `river.NewPeriodicJob(...)` and are registered through `jobs.NewWithWorkers` (see `internal/platform/jobs/client.go:318-348`).
- Database role is `skawld_app`; every new table needs `GRANT SELECT, INSERT, UPDATE ... TO skawld_app` and `REVOKE DELETE, TRUNCATE` (follow `00002_core_maintenance.sql:328-329`).
- Web API calls go through `web/src/api.ts` `request()` helper (prefix `/api/v1`, `credentials: "include"`, 401 → redirect `/auth/login`); mutations use `command()` with an Idempotency-Key.
- Every new i18n string needs both `en` and `vi` entries in `web/src/i18n/messages.ts` (`const en`, then `const vi: Record<MessageKey, string>`).
- No new frontend chart library: sparklines are inline SVG only.
- `internal/monitoring` must not import `internal/platform/httpserver` or any `web` package (enforced by `internal/architecture/imports_test.go`).
- Postgres-backed tests are `*_integration_test.go` gated by `TEST_DATABASE_URL` (`t.Skip` when unset), per `internal/evaluation/adapter/postgres/store_integration_test.go`. Run them with `TEST_DATABASE_URL` set and migrations applied.
- All timestamps UTC; store `numeric` values for metrics.

---

### Task 1: Monitoring tables migration

**Files:**
- Create: `migrations/00013_monitoring.sql`
- Modify: `migrations/embed.go` (embed the new file — verify it globs `*.sql`; if it lists files explicitly, add it)

**Interfaces:**
- Consumes: existing `organizations`, `sites`, `principals` tables.
- Produces: tables `monitoring_metrics`, `monitor_thresholds`, `monitor_alert_events`, `monitoring_backup_runs`, `monitoring_request_counters` used by Tasks 3, 4, 6, 7.

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE monitoring_metrics (
    id              bigserial PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id         uuid REFERENCES sites(id),
    metric_key      text NOT NULL,
    dimensions      jsonb NOT NULL DEFAULT '{}'::jsonb,
    granularity     text NOT NULL DEFAULT 'day',
    day             date,
    value           numeric NOT NULL,
    sample_count    bigint NOT NULL DEFAULT 1,
    collected_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, site_id, metric_key, dimensions, granularity, day)
);

CREATE UNIQUE INDEX monitoring_metrics_latest_key
    ON monitoring_metrics (organization_id, site_id, metric_key, dimensions)
    WHERE granularity = 'latest';

CREATE INDEX monitoring_metrics_lookup
    ON monitoring_metrics (organization_id, metric_key, day DESC);

CREATE TABLE monitor_thresholds (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id         uuid REFERENCES sites(id),
    metric_key      text NOT NULL,
    comparator      text NOT NULL CHECK (comparator IN ('GT','GTE','LT','LTE')),
    warn_value      numeric NOT NULL,
    crit_value      numeric NOT NULL,
    enabled         boolean NOT NULL DEFAULT true,
    updated_by      uuid NOT NULL REFERENCES principals(id),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, site_id, metric_key)
);

CREATE TABLE monitor_alert_events (
    id              bigserial PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id         uuid REFERENCES sites(id),
    metric_key      text NOT NULL,
    dimensions      jsonb NOT NULL DEFAULT '{}'::jsonb,
    state           text NOT NULL CHECK (state IN ('WARN','CRIT')),
    message         text NOT NULL,
    value           numeric NOT NULL,
    opened_at       timestamptz NOT NULL DEFAULT now(),
    resolved_at     timestamptz
);

CREATE UNIQUE INDEX monitor_alert_events_active
    ON monitor_alert_events (organization_id, site_id, metric_key, dimensions, state)
    WHERE resolved_at IS NULL;

CREATE TABLE monitoring_backup_runs (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id         uuid REFERENCES sites(id),
    started_at      timestamptz NOT NULL,
    completed_at    timestamptz,
    status          text NOT NULL CHECK (status IN ('RUNNING','SUCCESS','FAILED')),
    size_bytes      bigint
);

CREATE TABLE monitoring_request_counters (
    id           bigserial PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id      uuid REFERENCES sites(id),
    endpoint     text NOT NULL,
    status_class text NOT NULL,
    window_start timestamptz NOT NULL,
    count        bigint NOT NULL DEFAULT 0,
    UNIQUE (organization_id, site_id, endpoint, status_class, window_start)
);

GRANT SELECT, INSERT, UPDATE ON monitoring_metrics, monitor_thresholds,
    monitor_alert_events, monitoring_backup_runs, monitoring_request_counters
    TO skawld_app;
GRANT USAGE, SELECT ON SEQUENCE monitoring_metrics_id_seq,
    monitor_alert_events_id_seq, monitoring_request_counters_id_seq
    TO skawld_app;
REVOKE DELETE, TRUNCATE ON monitoring_metrics, monitor_thresholds,
    monitor_alert_events, monitoring_backup_runs, monitoring_request_counters
    FROM skawld_app;

-- +goose Down
DROP TABLE IF EXISTS monitoring_request_counters;
DROP TABLE IF EXISTS monitoring_backup_runs;
DROP TABLE IF EXISTS monitor_alert_events;
DROP TABLE IF EXISTS monitor_thresholds;
DROP TABLE IF EXISTS monitoring_metrics;
```

- [ ] **Step 2: Update `migrations/embed.go`** — confirm the `//go:embed` glob picks up `00013_monitoring.sql`; if the file enumerates migrations, add it.

- [ ] **Step 3: Run migration against the test database**

Run: `make migrate-up` (or the project's migration command from `docs/runbooks/local-development.md`)
Expected: migration applies cleanly.

- [ ] **Step 4: Commit**

```bash
git add migrations/00013_monitoring.sql migrations/embed.go
git commit -m "feat: add monitoring metrics, thresholds, and alert tables"
```

---

### Task 2: Monitoring domain — catalog, status, threshold evaluation

**Files:**
- Create: `internal/monitoring/domain/catalog.go`
- Create: `internal/monitoring/domain/status.go`
- Create: `internal/monitoring/domain/status_test.go`

**Interfaces:**
- Consumes: nothing (pure Go).
- Produces (used by Tasks 3-9):
  - `type Granularity string` with `GranularityLatest Granularity = "latest"` and `GranularityDay Granularity = "day"`.
  - `type Status string` with `StatusPass`, `StatusWarn`, `StatusCrit`.
  - `type Comparator string` with `ComparatorGT`, `ComparatorGTE`, `ComparatorLT`, `ComparatorLTE`.
  - `func (c Comparator) Compare(metricValue, thresholdValue float64) bool`
  - `func Evaluate(metricValue float64, warnValue, critValue float64, comparator Comparator) Status`
  - `type CatalogEntry struct { Key, Group, DisplayName, Unit, Direction string; Granularity Granularity }`
  - `var Catalog map[string]CatalogEntry` — keys from the spec: group A `ops.loto_blocked_count`, `ops.loto_verified_count`, `ops.incident_max_age_hours`, `ops.incident_backlog_count`, `ops.mttr_hours`; group C `sys.worker_queue_depth`, `sys.worker_retry_count`, `sys.worker_failed_count`, `sys.ingestion_failed_count`, `sys.backup_freshness_hours`, `sys.health_ready`. (Groups B/D rollups are Phase 2 — keep the catalog map only for Phase 1 keys plus a placeholder group constant.)
  - `func InCatalog(key string) bool`
  - `func DefaultThresholds(key string) (warnValue, critValue float64, comparator Comparator, enabled, ok bool)` — defaults: `sys.worker_failed_count` GT 0/0 enabled, `sys.backup_freshness_hours` GT 24/48 enabled, `sys.health_ready` LT 1/1 enabled, `sys.worker_retry_count` GT 5/20 enabled, `sys.ingestion_failed_count` GT 0/3 enabled, `ops.incident_max_age_hours` GT 24/72 enabled, `ops.incident_backlog_count` GT 0/0 **disabled** (informational — only alerts once an operator configures a threshold row), `ops.mttr_hours` GT 0/0 **disabled**, `ops.loto_blocked_count`/`ops.loto_verified_count`/`sys.worker_queue_depth` GT 0/0 **disabled**. `enabled=false` means `EvaluateAndAlert` (Task 4) never opens alerts from the default — a row in `monitor_thresholds` with `enabled=true` overrides.

- [ ] **Step 1: Write the failing tests** (`internal/monitoring/domain/status_test.go`)

```go
package domain

import "testing"

func TestEvaluatePrecedence(t *testing.T) {
	// crit first, then warn, then pass
	if got := Evaluate(10, 5, 7, ComparatorGT); got != StatusCrit {
		t.Fatalf("expected CRIT, got %s", got)
	}
	if got := Evaluate(6, 5, 7, ComparatorGT); got != StatusWarn {
		t.Fatalf("expected WARN, got %s", got)
	}
	if got := Evaluate(1, 5, 7, ComparatorGT); got != StatusPass {
		t.Fatalf("expected PASS, got %s", got)
	}
}

func TestEvaluateLT(t *testing.T) {
	if got := Evaluate(0, 1, 1, ComparatorLT); got != StatusCrit {
		t.Fatalf("expected CRIT, got %s", got)
	}
	if got := Evaluate(1, 1, 1, ComparatorLT); got != StatusPass {
		t.Fatalf("expected PASS for equal value, got %s", got)
	}
}

func TestCatalogContainsPhaseOneKeys(t *testing.T) {
	for _, key := range []string{
		"sys.worker_queue_depth", "sys.worker_retry_count",
		"sys.worker_failed_count", "sys.ingestion_failed_count",
		"sys.backup_freshness_hours", "sys.health_ready",
		"ops.loto_blocked_count", "ops.loto_verified_count",
		"ops.incident_max_age_hours", "ops.incident_backlog_count",
		"ops.mttr_hours",
	} {
		if !InCatalog(key) {
			t.Errorf("catalog missing %q", key)
		}
	}
}

func TestDefaultThresholdsShape(t *testing.T) {
	for key := range Catalog {
		warn, crit, cmp, _, ok := DefaultThresholds(key)
		if !ok {
			t.Errorf("no default threshold for %q", key)
		}
		if cmp == "" {
			t.Errorf("no comparator for %q", key)
		}
		if cmp == ComparatorGT && crit < warn {
			t.Errorf("key %q crit %v < warn %v", key, crit, warn)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/monitoring/domain/ -run 'TestEvaluate|TestCatalog|TestDefault' -v`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Write the implementation** (`catalog.go`, `status.go`)

```go
// status.go
package domain

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusCrit Status = "CRIT"
)

type Comparator string

const (
	ComparatorGT  Comparator = "GT"
	ComparatorGTE Comparator = "GTE"
	ComparatorLT  Comparator = "LT"
	ComparatorLTE Comparator = "LTE"
)

func (c Comparator) Compare(value, threshold float64) bool {
	switch c {
	case ComparatorGT:
		return value > threshold
	case ComparatorGTE:
		return value >= threshold
	case ComparatorLT:
		return value < threshold
	case ComparatorLTE:
		return value <= threshold
	default:
		return false
	}
}

func Evaluate(value, warn, crit float64, comparator Comparator) Status {
	if comparator.Compare(value, crit) {
		return StatusCrit
	}
	if comparator.Compare(value, warn) {
		return StatusWarn
	}
	return StatusPass
}
```

```go
// catalog.go
package domain

type Granularity string

const (
	GranularityLatest Granularity = "latest"
	GranularityDay    Granularity = "day"
)

type CatalogEntry struct {
	Key         string
	Group       string
	DisplayName string
	Unit        string
	Direction   string // "down_is_good" | "up_is_good"
	Granularity Granularity
}

var Catalog = map[string]CatalogEntry{
	"sys.worker_queue_depth":      {Key: "sys.worker_queue_depth", Group: "system", DisplayName: "Worker queue depth", Unit: "jobs", Direction: "down_is_good", Granularity: GranularityLatest},
	"sys.worker_retry_count":      {Key: "sys.worker_retry_count", Group: "system", DisplayName: "Worker retries", Unit: "jobs", Direction: "down_is_good", Granularity: GranularityLatest},
	"sys.worker_failed_count":     {Key: "sys.worker_failed_count", Group: "system", DisplayName: "Worker failed jobs", Unit: "jobs", Direction: "down_is_good", Granularity: GranularityLatest},
	"sys.ingestion_failed_count":  {Key: "sys.ingestion_failed_count", Group: "system", DisplayName: "Ingestion failures", Unit: "docs", Direction: "down_is_good", Granularity: GranularityLatest},
	"sys.backup_freshness_hours":  {Key: "sys.backup_freshness_hours", Group: "system", DisplayName: "Backup freshness", Unit: "hours", Direction: "down_is_good", Granularity: GranularityLatest},
	"sys.health_ready":            {Key: "sys.health_ready", Group: "system", DisplayName: "Backend ready", Unit: "0/1", Direction: "up_is_good", Granularity: GranularityLatest},
	"ops.loto_blocked_count":      {Key: "ops.loto_blocked_count", Group: "ops", DisplayName: "LOTO blocks", Unit: "events", Direction: "down_is_good", Granularity: GranularityDay},
	"ops.loto_verified_count":     {Key: "ops.loto_verified_count", Group: "ops", DisplayName: "LOTO verifications", Unit: "events", Direction: "up_is_good", Granularity: GranularityDay},
	"ops.incident_max_age_hours":  {Key: "ops.incident_max_age_hours", Group: "ops", DisplayName: "Oldest open incident", Unit: "hours", Direction: "down_is_good", Granularity: GranularityDay},
	"ops.incident_backlog_count":  {Key: "ops.incident_backlog_count", Group: "ops", DisplayName: "Aged incident backlog", Unit: "incidents", Direction: "down_is_good", Granularity: GranularityDay},
	"ops.mttr_hours":              {Key: "ops.mttr_hours", Group: "ops", DisplayName: "MTTR", Unit: "hours", Direction: "down_is_good", Granularity: GranularityDay},
}

func InCatalog(key string) bool {
	_, ok := Catalog[key]
	return ok
}

func DefaultThresholds(key string) (warn, crit float64, comparator Comparator, enabled, ok bool) {
	switch key {
	case "sys.worker_failed_count", "sys.ingestion_failed_count":
		return 0, 0, ComparatorGT, true, true // any failure above 0 is CRIT
	case "sys.worker_retry_count":
		return 5, 20, ComparatorGT, true, true
	case "sys.backup_freshness_hours":
		return 24, 48, ComparatorGT, true, true
	case "sys.health_ready":
		return 1, 1, ComparatorLT, true, true // value 0 (not ready) is CRIT
	case "ops.incident_max_age_hours":
		return 24, 72, ComparatorGT, true, true
	case "ops.incident_backlog_count", "ops.mttr_hours",
		"ops.loto_blocked_count", "ops.loto_verified_count",
		"sys.worker_queue_depth":
		return 0, 0, ComparatorGT, false, true // informational: no alerts until configured
	default:
		return 0, 0, "", false, false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/monitoring/domain/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/monitoring/domain/
git commit -m "feat: add monitoring domain with threshold evaluation and catalog"
```

---

### Task 3: Monitoring postgres store — metrics, thresholds, alerts

**Files:**
- Create: `internal/monitoring/adapter/postgres/store.go`
- Create: `internal/monitoring/adapter/postgres/store_integration_test.go` (integration test gated by the `TEST_DATABASE_URL` env var, mirroring `internal/evaluation/adapter/postgres/store_integration_test.go`: `t.Skip("TEST_DATABASE_URL is not set")` when unset, same-package `_test`, live pool from `pgxpool.New`)

**Interfaces:**
- Consumes: `internal/monitoring/domain` (Task 2).
- Produces (used by Tasks 4, 7):
  - `type MetricRow struct { OrganizationID, SiteID, MetricKey string; Dimensions map[string]any; Granularity domain.Granularity; Day *time.Time; Value float64; SampleCount int64; CollectedAt time.Time }`
  - `func (s Store) UpsertMetric(ctx context.Context, row MetricRow) error`
  - `func (s Store) MetricsForRange(ctx context.Context, orgID, siteID, key string, from, to time.Time) ([]MetricRow, error)`
  - `type Threshold struct { ID, OrganizationID, SiteID, MetricKey, Comparator string; WarnValue, CritValue float64; Enabled bool; UpdatedAt time.Time }`
  - `func (s Store) ListThresholds(ctx context.Context, orgID string) ([]Threshold, error)`
  - `func (s Store) UpsertThreshold(ctx context.Context, value Threshold) error`
  - `type AlertEvent struct { ID int64; OrganizationID, SiteID, MetricKey string; Dimensions map[string]any; State domain.Status; Message string; Value float64; OpenedAt, ResolvedAt *time.Time }`
  - `func (s Store) OpenAlert(ctx context.Context, orgID, siteID, key string, dimensions map[string]any, state domain.Status, message string, value float64) error` (idempotent via the partial unique index; returns nil if already open)
  - `func (s Store) ResolveAlerts(ctx context.Context, orgID, siteID, key string, dimensions map[string]any) error` (resolves open WARN+CRIT for that key)
  - `func (s Store) ListAlerts(ctx context.Context, orgID string, openOnly bool) ([]AlertEvent, error)`

- [ ] **Step 1: Write the failing integration test** (`store_integration_test.go`) — gate on `TEST_DATABASE_URL`, create a fresh org + site, then assert: upserting the same (org, site, key, dimensions, granularity, day) twice keeps one row with the newer value; `OpenAlert` twice leaves one open row; `ResolveAlerts` sets `resolved_at`; `ListThresholds` returns org rows and site overrides.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/monitoring/adapter/postgres/ -run TestStore -v`
Expected: FAIL — store not implemented (compile error or missing methods). Without `TEST_DATABASE_URL` the integration test skips, so set it from the compose-up Postgres (CI sets `postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable` after `go run ./cmd/migrate up`).

- [ ] **Step 3: Write the implementation** (`store.go`) — skeleton with exact SQL:

```go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	if row.Granularity == monitoringdomain.GranularityDay {
		d := row.Day.UTC().Truncate(24 * time.Hour)
		day = &d
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO monitoring_metrics
			(organization_id, site_id, metric_key, dimensions, granularity, day, value, sample_count, collected_at)
		VALUES ($1::uuid, nullif($2, '')::uuid, $3, $4::jsonb, $5, $6::date, $7, $8, $9)
		ON CONFLICT (organization_id, site_id, metric_key, dimensions, granularity, day)
		DO UPDATE SET value = EXCLUDED.value,
		              sample_count = EXCLUDED.sample_count,
		              collected_at = EXCLUDED.collected_at
	`, row.OrganizationID, row.SiteID, row.MetricKey, dimensions,
		row.Granularity, day, row.Value, row.SampleCount, row.CollectedAt.UTC())
	return err
}
```

Implement the remaining methods with straightforward SQL:
- `MetricsForRange`: `SELECT organization_id::text, coalesce(site_id::text,''), metric_key, dimensions, granularity, day, value, sample_count, collected_at FROM monitoring_metrics WHERE organization_id = $1::uuid AND metric_key = $2 AND ($3::uuid[] IS NULL OR cardinality($3::uuid[]) = 0 OR site_id = ANY($3::uuid[])) AND (day >= $4::date OR (granularity = 'latest' AND collected_at >= $4)) AND collected_at < $5 ORDER BY day, collected_at` — parameterize org, key, site array, from, to.
- `ListThresholds`: org rows plus site rows: `SELECT ... WHERE organization_id = $1::uuid AND enabled ORDER BY metric_key`.
- `UpsertThreshold`: `INSERT ... ON CONFLICT (organization_id, site_id, metric_key) DO UPDATE SET comparator = EXCLUDED.comparator, warn_value = EXCLUDED.warn_value, crit_value = EXCLUDED.crit_value, enabled = EXCLUDED.enabled, updated_by = EXCLUDED.updated_by, updated_at = now()`.
- `OpenAlert`: `INSERT INTO monitor_alert_events (organization_id, site_id, metric_key, dimensions, state, message, value) VALUES (...) ON CONFLICT DO NOTHING` (relies on the partial unique index).
- `ResolveAlerts`: `UPDATE monitor_alert_events SET resolved_at = now() WHERE organization_id = $1::uuid AND metric_key = $2 AND resolved_at IS NULL AND (site_id = $3::uuid OR ($3 = '' AND site_id IS NULL))`.
- `ListAlerts`: org-scoped, optional `resolved_at IS NULL` filter, `ORDER BY opened_at DESC`, join nothing.

- [ ] **Step 4: Run to verify they pass**

Run: `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test ./internal/monitoring/adapter/postgres/ -v` (Postgres from `make compose-up`, migrations applied)
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/monitoring/adapter/postgres/
git commit -m "feat: add monitoring store with metrics, thresholds, and alerts"
```

---

### Task 4: River collectors and workers

**Files:**
- Create: `internal/monitoring/adapter/river/collectors.go`
- Create: `internal/monitoring/adapter/river/workers.go`
- Create: `internal/monitoring/adapter/river/collectors_integration_test.go`

**Interfaces:**
- Consumes: `Store` (Task 3), `monitoringdomain` (Task 2).
- Produces (used by Task 5):
  - `type Collector func(ctx context.Context, orgID string, siteIDs []string, now time.Time) error` — one per metric group; each writes rows via `Store.UpsertMetric` then evaluates thresholds.
  - `func CollectSnapshot(ctx context.Context, store Store, orgID string, siteIDs []string, now time.Time) error`
  - `func CollectDaily(ctx context.Context, store Store, orgID string, siteIDs []string, now time.Time) error`
  - `func EvaluateAndAlert(ctx context.Context, store Store, orgID, siteID string, key string, dimensions map[string]any, value float64, threshold Threshold, message string) error` — pure transition logic: compute status with `domain.Evaluate`; if CRIT/WARN and no matching open alert → `OpenAlert`; if PASS → `ResolveAlerts`.
  - `type SnapshotArgs struct{}` with `Kind() string` → `"monitoring.snapshot"`.
  - `type DailyRollupArgs struct{}` with `Kind() string` → `"monitoring.daily_rollup"`.

- [ ] **Step 1: Write failing tests** (`collectors_integration_test.go`) gated on `TEST_DATABASE_URL` with a seeded pool: create an org with one site; insert one incident `detected_at` 3 days ago and `resolved_at` 1 day ago (for `ops.mttr_hours` = 48) and one open incident 2 days old (for `ops.incident_max_age_hours` = 48, backlog bucket 24h = 1); insert a `river.job` row with state `retryable` in queue `foundation` and one `cancelled` (the `river` schema exists after `cmd/migrate` applies River migrations, as CI does); insert `monitoring_backup_runs` SUCCESS `completed_at` 12 hours ago; insert a `domain_events` row with `event_type = 'execution.step.blocked'` (columns per `events.Append` in `internal/platform/events/outbox.go`). Assert after `CollectDaily`: `ops.mttr_hours` row exists ≈ 48; `ops.incident_max_age_hours` ≥ 48; `ops.loto_blocked_count` = 1. After `CollectSnapshot`: `sys.worker_retry_count` = 1, `sys.backup_freshness_hours` ≈ 12, `sys.health_ready` = 1. Also assert a CRIT alert opens when a `sys.worker_failed_count` row is collected with value 1.

- [ ] **Step 2: Run to verify it fails**

Run: `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test ./internal/monitoring/adapter/river/ -run TestCollect -v`
Expected: FAIL (compile error or missing methods).

- [ ] **Step 3: Write the implementation**

Collector SQL (exact queries to embed):

```sql
-- sys.worker_retry_count (per queue dimension), river schema
SELECT queue, count(*) FROM river.job
WHERE state IN ('retryable') AND (attempted_at IS NULL OR attempted_at >= now() - interval '24 hours')
GROUP BY queue;
-- sys.worker_failed_count: state = 'cancelled' in last 24h
SELECT count(*) FROM river.job WHERE state = 'cancelled' AND attempted_at >= now() - interval '24 hours';
-- sys.worker_queue_depth: active states per queue
SELECT queue, count(*) FROM river.job
WHERE state IN ('available','scheduled','retryable','running')
GROUP BY queue;
-- sys.backup_freshness_hours: latest SUCCESS run per org (site NULL)
SELECT extract(epoch FROM (now() - completed_at)) / 3600
FROM monitoring_backup_runs
WHERE organization_id = $1::uuid AND status = 'SUCCESS'
ORDER BY completed_at DESC LIMIT 1;
-- sys.health_ready: 1 if DB ping ok (use pool.Ping)
-- sys.ingestion_failed_count: document_revisions.ingestion_state = 'FAILED' in last 24h
--   (column is ingestion_state on document_revisions, see 00004_copilot.sql:41;
--    count WHERE ingestion_state = 'FAILED' AND updated_at >= now() - interval '24 hours')
-- ops.loto_blocked_count / ops.loto_verified_count (dimension = site):
SELECT site_id::text, count(*) FROM domain_events
WHERE organization_id = $1::uuid AND event_type = 'execution.step.blocked'
  AND occurred_at >= $2::date GROUP BY site_id;
-- verified: event_type = 'execution.prerequisite.recorded' AND payload->'value'->>'status' = 'VERIFIED'
-- ops.incident_max_age_hours (per state dimension):
SELECT state, max(extract(epoch FROM (now() - detected_at)) / 3600)
FROM incidents WHERE organization_id = $1::uuid AND state IN ('OPEN','IN_PROGRESS')
GROUP BY state;
-- ops.incident_backlog_count (bucket dimension: '24h'):
SELECT count(*) FROM incidents
WHERE organization_id = $1::uuid AND state IN ('OPEN','IN_PROGRESS')
  AND detected_at < now() - interval '24 hours';
-- ops.mttr_hours (severity dimension):
SELECT severity, avg(extract(epoch FROM (resolved_at - detected_at)) / 3600)
FROM incidents WHERE organization_id = $1::uuid AND resolved_at IS NOT NULL
GROUP BY severity;
```

`CollectSnapshot` runs each collector, logging-and-skipping individual failures; `EvaluateAndAlert` applies org-wide + site-specific thresholds from `ListThresholds`, falling back to `DefaultThresholds` — and **skips alerting entirely when the effective threshold has `enabled=false`** (defaults) unless a `monitor_thresholds` row with `enabled=true` overrides it. Workers:

```go
type SnapshotWorker struct {
	river.WorkerDefaults[SnapshotArgs]
	Store Store
}

func (w *SnapshotWorker) Work(ctx context.Context, _ *river.Job[SnapshotArgs]) error {
	return CollectSnapshot(ctx, w.Store, "", nil, time.Now())
}

func (*SnapshotWorker) Timeout(*river.Job[SnapshotArgs]) time.Duration { return 2 * time.Minute }
```

(Org/site iteration comes from a `SELECT id FROM organizations` plus per-org site list; implement a small `loadScopes(ctx, pool)` helper.)

- [ ] **Step 4: Run to verify they pass**

Run: `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test ./internal/monitoring/adapter/river/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/monitoring/adapter/river/
git commit -m "feat: add monitoring collectors and periodic workers"
```

---

### Task 5: Register monitoring workers

**Files:**
- Modify: `internal/platform/jobs/client.go` (add to `WorkerSet` + registration + periodic jobs)
- Modify: `cmd/worker/main.go` (construct `monitoring` store + collectors and pass into `WorkerSet`)

**Interfaces:**
- Consumes: `SnapshotWorker`, `DailyRollupWorker` (Task 4).
- Produces: workers run on `QueueFoundation`; snapshot every 5 minutes, daily at 00:10 UTC.

- [ ] **Step 1: Extend `WorkerSet` and `NewWithWorkers`**

Add `Snapshotter monitoring.Snapshotter` and `DailyRolluper monitoring.DailyRolluper` (define small interfaces in the jobs package to avoid an import cycle) to `WorkerSet`; in `NewWithWorkers`, when set, `AddWorkerSafely` both workers and append two `river.NewPeriodicJob`s:

```go
river.NewPeriodicJob(
	river.PeriodicInterval(5*time.Minute),
	func() (river.JobArgs, *river.InsertOpts) {
		return SnapshotArgs{}, &river.InsertOpts{
			Queue: QueueFoundation, MaxAttempts: 1,
			UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Minute},
		}
	},
	&river.PeriodicJobOpts{RunOnStart: true},
),
// daily: river.NewPeriodicJob(river.PeriodicCron("@daily"), ...) with UniqueOpts{ByPeriod: 24 * time.Hour}
```

- [ ] **Step 2: Wire in `cmd/worker/main.go`**

Construct `monitoringstore := monitoringpostgres.Store{Pool: pool}`, pass it into `jobs.NewWithWorkers(..., jobs.WorkerSet{ ..., Snapshotter: monitoringstore, DailyRolluper: monitoringstore })` (add the methods to the store if the interfaces require them; alternatively pass the collectors directly).

- [ ] **Step 3: Run worker + tests**

Run: `go build ./cmd/worker && go test ./internal/platform/jobs/ ./cmd/worker/ -v`
Expected: build passes; worker starts with both periodic jobs registered (check log line on `RunOnStart`).

- [ ] **Step 4: Commit**

```bash
git add internal/platform/jobs/client.go cmd/worker/main.go
git commit -m "feat: register monitoring snapshot and daily rollup workers"
```

---

### Task 6: Identity permissions and role matrix

**Files:**
- Modify: `internal/identity/domain/authorization.go` (two new permissions + role grants)
- Modify: `cmd/seed/main.go` (grant to seed-admin — find the grant INSERT and add both)
- Modify: `web/src/console/permissions.ts` (add the two keys to `PermissionKey`)

**Interfaces:**
- Consumes: nothing new.
- Produces: `identitydomain.PermissionMonitoringRead = "monitoring:read"`, `PermissionMonitoringConfigure = "monitoring:configure"`; `web` `PermissionKey` adds `"monitoring:read" | "monitoring:configure"`.

- [ ] **Step 1: Add permissions + grants**

In `authorization.go` add the two `Permission` constants and append to `RoleAdministrator` (both) and `RoleMaintenanceSupervisor` + `RoleManager` (`monitoring:read` only). Keep unknown-role fail-closed tests passing (`PermissionsForUnknownRoleFailsClosed`).

- [ ] **Step 2: Grant to seed admin**

In `cmd/seed/main.go`, find where `seed-admin` gets its permission grants (a grant INSERT with `approval_authorities` / role membership) and ensure the Administrator role grant covers the new permissions (the role matrix is the single source — verify the seed relies on `PermissionsForRole`; if it does, no seed change is needed beyond the matrix, and note that in the commit).

- [ ] **Step 3: Update web permission keys**

Add `"monitoring:read"` and `"monitoring:configure"` to `PermissionKey` in `web/src/console/permissions.ts`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/identity/... -v && npm test -- --run web/src/console/permissions.test.ts`
Expected: PASS (existing role-matrix tests still green).

- [ ] **Step 5: Commit**

```bash
git add internal/identity/domain/authorization.go cmd/seed/main.go web/src/console/permissions.ts
git commit -m "feat: add monitoring read and configure permissions"
```

---

### Task 7: Monitoring HTTP endpoints

**Files:**
- Create: `internal/platform/httpserver/monitoring.go`
- Create: `internal/platform/httpserver/monitoring_test.go`
- Modify: `internal/platform/httpserver/router.go` (mount + add monitoring Service to `Dependencies`)

**Interfaces:**
- Consumes: `Store` (Task 3) via a small `monitoringapp` service; `identitydomain` permissions (Task 6).
- Produces: routes below; the API client consumes these in Task 8.

| Method | Path | Body/Query | Response |
|---|---|---|---|
| GET | `/monitoring/summary` | `site_id` optional | `[{ "metric_key", "status", "value", "unit", "group", "site_id", "collected_at", "open_alerts" }]` |
| GET | `/monitoring/metrics` | `metric_key`, `site_id`, `from`, `to`, `granularity` | `{ "items": [{ "day", "value", "sample_count", "dimensions", "site_id" }] }` |
| GET | `/monitoring/alerts` | `state=OPEN\|RESOLVED`, `site_id` | `{ "items": [{ "id", "metric_key", "site_id", "state", "message", "value", "opened_at", "resolved_at" }] }` |
| PUT | `/monitoring/thresholds` | `{ metric_key, site_id?, comparator, warn_value, crit_value, enabled }` | 204 |

Authorization: GET endpoints require `monitoring:read`; PUT requires `monitoring:configure`. Site filter via principal `site_ids` like `summary.go`.

- [ ] **Step 1: Write failing handler tests** (`monitoring_test.go`) mirroring `summary_test.go`: 401 without auth, 403 without `monitoring:read`, 200 happy path for `/monitoring/summary` with a seeded metric row and threshold, PUT upsert round-trip, site filter narrowing rows.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/platform/httpserver/ -run TestMonitoring -v`
Expected: FAIL.

- [ ] **Step 3: Implement the handlers**

Follow the `summary.go` / `evaluations.go` handler style (principal check, `validUUIDParam`, `writeJSON`/`writeProblem`). `GET /monitoring/summary` iterates `domain.Catalog`, reads the latest `latest` (or last `day`) row per key from the store, computes `domain.Evaluate` against resolved thresholds, and counts open alerts per key. `PUT /monitoring/thresholds` validates `domain.InCatalog(metric_key)`, comparator ∈ {GT,GTE,LT,LTE}, `crit >= warn` for GT/GTE, `crit <= warn` for LT/LTE, then `UpsertThreshold` with `updated_by` from the principal.

- [ ] **Step 4: Mount in router.go**

Add `Monitoring monitoringapp.Service` to `Dependencies` and `mountMonitoringRoutes(router, deps.Monitoring)` next to the other mounts.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/platform/httpserver/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/platform/httpserver/monitoring.go internal/platform/httpserver/monitoring_test.go internal/platform/httpserver/router.go
git commit -m "feat: add monitoring summary, metrics, alerts, and thresholds endpoints"
```

---

### Task 8: Web API client and types

**Files:**
- Modify: `web/src/types.ts` (add `MonitoringMetric`, `MonitoringSummaryEntry`, `MonitoringAlert`, `MonitorThreshold` types)
- Modify: `web/src/api.ts` (add five methods to `api`)
- Modify: `web/src/api.test.ts` (cover the new methods)

**Interfaces:**
- Consumes: HTTP endpoints (Task 7).
- Produces (used by Task 9):

```ts
export type MonitoringMetric = {
  day?: string; value: number; sample_count: number;
  dimensions: Record<string, unknown>; site_id?: string;
};
export type MonitoringSummaryEntry = {
  metric_key: string; status: "PASS" | "WARN" | "CRIT";
  value: number; unit: string; group: string; site_id?: string;
  collected_at?: string; open_alerts: number;
};
export type MonitoringAlert = {
  id: number; metric_key: string; site_id?: string;
  state: "WARN" | "CRIT"; message: string; value: number;
  opened_at: string; resolved_at?: string;
};
export type MonitorThreshold = {
  metric_key: string; site_id?: string; comparator: "GT" | "GTE" | "LT" | "LTE";
  warn_value: number; crit_value: number; enabled: boolean;
};
```

`api` methods: `monitoringSummary(siteID?: string)`, `monitoringMetrics(key: string, opts?: { site_id?: string; from?: string; to?: string })`, `monitoringAlerts(opts?: { state?: "OPEN" | "RESOLVED"; site_id?: string })`, `monitoringThresholds()`, `setMonitoringThreshold(value: MonitorThreshold)` (PUT via `request` with method PUT).

- [ ] **Step 1: Add types + API methods + tests** (test the URL/params via mocked `fetch`, following the existing `api.test.ts` style).

- [ ] **Step 2: Run tests**

Run: `npm test -- --run web/src/api.test.ts web/src/types.ts`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.test.ts
git commit -m "feat: add monitoring API client and types"
```

---

### Task 9: Monitoring page UI

**Files:**
- Create: `web/src/console/pages/MonitoringPage.tsx`
- Create: `web/src/console/pages/MonitoringPage.test.tsx`
- Create: `web/src/console/components/monitoring/Sparkline.tsx` (inline SVG)
- Create: `web/src/console/components/monitoring/StatusCard.tsx`
- Create: `web/src/console/components/monitoring/ThresholdEditor.tsx`
- Modify: `web/src/App.tsx` (route `/monitoring`)
- Modify: `web/src/console/layout/Sidebar.tsx` (new section item, Gauge icon, gated on `monitoring:read`)
- Modify: `web/src/i18n/messages.ts` (all new strings, en + vi)
- Modify: `web/src/console/permissions.ts` only if a focus/label helper needs the keys (Task 6 already added the keys)

**Interfaces:**
- Consumes: `api` methods (Task 8), `can()`/`PermissionKey` (Task 6), `useCommand` toast pattern.
- Produces: the `/monitoring` route and sidebar entry; the NotificationBell in Task 10 reuses `MonitoringSummaryEntry`.

- [ ] **Step 1: Add i18n strings** (en + vi) — `nav.monitoring`, `monitoring.overview`, `monitoring.system`, `monitoring.alerts`, `monitoring.thresholds`, `monitoring.statusPass`, `monitoring.statusWarn`, `monitoring.statusCrit`, `monitoring.group.system`, `monitoring.group.ops`, `monitoring.refresh`, `monitoring.noData`, `monitoring.configure`, `monitoring.thresholdSaved`, `monitoring.exportCsv`.

- [ ] **Step 2: Implement `Sparkline`** — takes `values: number[]`, renders an inline `<svg>` polyline with a 12×4 viewBox scaled to min/max; export a `toCSV(rows, columns)` helper in the same file for the export buttons.

- [ ] **Step 3: Implement `StatusCard`** — props `{ entry: MonitoringSummaryEntry; history: MonitoringMetric[]; to?: string }`; renders MetricCard-style body with `StatusBadge` tone map `PASS→"success"|"info", WARN→"warning", CRIT→"danger"` (match the tones used in `labels.ts`), latest value, sparkline, and open-alert count.

- [ ] **Step 4: Implement `MonitoringPage`** — tabs via the existing `Tabs` component: Overview (grouped StatusCards by `group`), System (grid of `sys.*` cards), Alerts (`DataTable` of `MonitoringAlert`, filter by state, CSV export via `toCSV`), Thresholds (`DataTable` + `ThresholdEditor` dialog using the existing `Dialog`/`FormField` components, PUT via `useCommand` with success toast `monitoring.thresholdSaved`, edit affordance gated on `monitoring:configure`). Query with `useQuery` + `useSite` site scoping; render `Skeleton` while loading and `ErrorState` with retry on error; `PageTrailProvider` with `[{ label: t("nav.monitoring") }]`.

- [ ] **Step 5: Add route + sidebar entry** — in `App.tsx` add `<Route path="/monitoring" element={<MonitoringPage />} />`; in `Sidebar.tsx` add `{ to: "/monitoring", key: "nav.monitoring", icon: Gauge, permission: "monitoring:read" }` to a new `sidebar.monitoring` section after the quality section.

- [ ] **Step 6: Write page tests** — mock `api` like `QualityPage.test.tsx`: summary returns a CRIT `sys.backup_freshness_hours` entry → assert CRIT badge renders; alerts list renders rows; threshold editor PUT calls `setMonitoringThreshold`; sidebar item hidden when principal lacks `monitoring:read`.

- [ ] **Step 7: Run tests + build**

Run: `npm test -- --run web/src/console/pages/MonitoringPage.test.tsx && npm run build`
Expected: PASS, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add web/src/console/pages/MonitoringPage.tsx web/src/console/pages/MonitoringPage.test.tsx web/src/console/components/monitoring/ web/src/App.tsx web/src/console/layout/Sidebar.tsx web/src/i18n/messages.ts
git commit -m "feat: add monitoring console page with overview, system, alerts, and thresholds"
```

---

### Task 10: Notification bell

**Files:**
- Create: `web/src/console/components/monitoring/NotificationBell.tsx`
- Create: `web/src/console/components/monitoring/NotificationBell.test.tsx`
- Modify: `web/src/console/layout/ConsoleLayout.tsx` (render the bell in a top bar area; if there is no topbar slot, render it above the `Outlet` in the main column, aligned right, gated on `monitoring:read`)

**Interfaces:**
- Consumes: `api.monitoringSummary` (Task 8); `can()` (Task 6).
- Produces: a bell with an open-alert count badge and a dropdown list of open CRIT/WARN alerts; polls every 60s while mounted.

- [ ] **Step 1: Implement `NotificationBell`** — `useEffect` interval 60s calling `api.monitoringSummary()`; count entries where `status !== "PASS"` summed via `open_alerts`; render a button with a badge; dropdown lists `metric_key`, `status` badge, `value`, `unit`; use `Phosphor` `Bell`/`BellRinging` icons; close on outside click.

- [ ] **Step 2: Render in `ConsoleLayout`** — `{can(principal, "monitoring:read") ? <NotificationBell principal={principal} /> : null}` placed in a header row above `<Outlet />`.

- [ ] **Step 3: Write tests** — bell shows count from mocked summary, dropdown lists a CRIT alert, hidden without permission.

- [ ] **Step 4: Run tests + build**

Run: `npm test -- --run web/src/console/components/monitoring/NotificationBell.test.tsx && npm run build`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/console/components/monitoring/NotificationBell.tsx web/src/console/components/monitoring/NotificationBell.test.tsx web/src/console/layout/ConsoleLayout.tsx
git commit -m "feat: add monitoring notification bell to the console shell"
```

---

### Task 11: Backup script marker (backup freshness)

**Files:**
- Modify: `scripts/backup-postgres.sh`
- Modify: `docs/runbooks/backup-restore.md`

**Interfaces:**
- Consumes: `monitoring_backup_runs` table (Task 1).
- Produces: real `sys.backup_freshness_hours` values (Task 4).

- [ ] **Step 1: Update the script** — after `pg_dump` succeeds, before the `mv`, insert/update a marker row. The script already has `BACKUP_DATABASE_URL`; add an optional `SKAWLD_ORGANIZATION_ID`/`SKAWLD_SITE_ID` (empty default) and run:

```bash
backup_run_id="$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)"
psql "${BACKUP_DATABASE_URL}" -v ON_ERROR_STOP=1 -c \
  "INSERT INTO monitoring_backup_runs (id, organization_id, site_id, started_at, status)
   VALUES ('${backup_run_id}', COALESCE(NULLIF('${SKAWLD_ORGANIZATION_ID:-}', '')::uuid, (SELECT id FROM organizations ORDER BY created_at LIMIT 1)), NULLIF('${SKAWLD_SITE_ID:-}', '')::uuid, now(), 'SUCCESS')
   ON CONFLICT (id) DO NOTHING;"
```

On failure, insert a `FAILED` row (wrap the existing `trap` logic so the failure path records `FAILED` before exiting).

- [ ] **Step 2: Update the runbook** — document the two new env vars and the marker table.

- [ ] **Step 3: Shell-check the script**

Run: `bash -n scripts/backup-postgres.sh`
Expected: no syntax errors.

- [ ] **Step 4: Commit**

```bash
git add scripts/backup-postgres.sh docs/runbooks/backup-restore.md
git commit -m "feat: record backup runs for freshness monitoring"
```

---

### Task 12: End-to-end verification

**Files:**
- Modify: `internal/architecture/imports_test.go` (add `internal/monitoring` import rule if the test uses an allowlist; otherwise verify it passes)

**Interfaces:**
- Consumes: everything above.

- [ ] **Step 1: Run the full backend suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 2: Run the full web suite + build**

Run: `npm test && npm run build`
Expected: PASS, build succeeds.

- [ ] **Step 3: Manual smoke test**

With `make compose-up` + `make migrate-up` + API + worker running: `make seed`, open the console, verify the `/monitoring` page renders with system cards, the bell shows no alerts (seed data has no failures), and `PUT /monitoring/thresholds` for `sys.health_ready` with `warn 0 crit 0` flips the card to CRIT (or verify a CRIT card appears after setting a low threshold on `sys.backup_freshness_hours`).

- [ ] **Step 4: Commit any cleanup**

```bash
git add -A
git commit -m "chore: verify monitoring pipeline end to end"
```
