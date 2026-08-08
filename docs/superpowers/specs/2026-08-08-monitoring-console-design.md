# Skawld Monitoring Console — Design (Brainstorm)

Date: 2026-08-08
Status: Draft for review (pre-implementation)
Scope: Operational KPIs, AI/copilot trends, system/infra health, workflow-learning
  monitors, configurable threshold alerts, new `/monitoring` console page
Deliverable: this design doc (doubles as the stakeholder proposal), then an
  implementation plan via writing-plans

## 0. Executive summary (proposal for stakeholders)

The pilot's value claims — safety, evidence-backed assistance, learning from
normal work — are currently invisible: the dashboard shows only live snapshot
counts and the Quality page shows one static AI summary with no history. No
surface answers "are we getting safer, faster, or more evidence-backed over
time?", and system health (worker, ingestion, backups) has no UI at all even
though backup freshness and worker retry health are named pilot-readiness
gates.

This design adds a **Monitoring Console** (`/monitoring`): one place where
operators, supervisors, and admins see every monitor across four groups —
operational KPIs, AI/copilot trends, system/infra health, and workflow
learning — with **history, configurable thresholds, and state-based alerts**
(OK → WARN → CRIT) that appear in a notification center without extra
infrastructure.

Recommended pilot order: LOTO safety events → incident aging/MTTR → worker &
ingestion health → backup freshness → AI trend charts → learning coverage.
The first four map directly to open pilot-readiness gates; the last two make
the copilot's value visible to stakeholders.

## 1. Design read

The console already has the primitives we need: MetricCard, DataTable,
StatusBadge, pagination/query hooks, i18n (en/vi), theme, role-aware sidebar
gating, and per-page tests. The backend has a mature River worker, scoped
principal query patterns (`organization_id` + `site_ids`), and a proven
postgres adapter style. What is missing is a **metrics layer**: nothing
records history, nothing aggregates across transaction tables on a schedule,
and nothing evaluates thresholds.

Guiding principle: **every number an operator sees must be truthful, dated,
and scoped.** No fabricated baselines, no "100%" from one sample, no
org-wide numbers masquerading as site numbers. Thresholds must be explicit
and adjustable at runtime — a monitor without a configurable threshold is a
decorative number.

## 2. Architecture (approach A: generic metrics store)

```
Transaction tables ──┐
  (incidents, eval,  │  River worker (existing infrastructure)
  executions, ...)   │
                     ▼
  SnapshotJob (5 min)  ─► monitoring_metrics ('latest' rows)   ─┐
  DailyRollupJob (daily) ─► monitoring_metrics ('day' rows)     ─┤
                                                               ▼
                              EvaluateThresholds (in job) ─► monitor_alert_events
                                                               (state-based)
                                                               ▼
  GET /monitoring/metrics | summary | alerts | thresholds (read-only API)
                                                               ▼
  MonitoringPage (/monitoring) + NotificationCenter (topbar bell + toasts)
```

Module layout (mirrors existing module conventions):

```
internal/monitoring/
  domain/               # MetricKey, Granularity, Comparator, AlertState,
                        #   Status (PASS/WARN/CRIT), threshold evaluation
  application/          # Service: CollectSnapshot, CollectDaily, EvaluateAlerts,
                        #   Query (metrics/summary/alerts/thresholds), Configure
  adapter/
    postgres/           # Store: upsert metrics, evaluate + open/resolve alerts,
                        #   thresholds CRUD, queries
    river/              # SnapshotWorker (periodic 5 min),
                        #   DailyRollupWorker (periodic daily)
internal/platform/httpserver/monitoring.go   # routes + handlers
web/src/console/pages/MonitoringPage.tsx     # the new page
web/src/console/components/monitoring/…      # Sparkline, TrendChart, StatusCard,
                                             #   AlertList, ThresholdEditor, CsvExport
```

Non-negotiables:

- **Jobs write, API reads.** The UI never queries transaction tables; it reads
  `monitoring_metrics`/`monitor_alert_events` only. Keeps UI latency bounded
  and the pipeline consistent.
- **Every metric row is scoped.** `organization_id` always set; `site_id`
  null means org-wide. Site switcher filters everything.
- **One migration, one job pipeline.** New monitors are new SQL in a job, not
  new tables or new endpoints.
- **Thresholds are runtime data.** The job reads them each cycle; changing a
  threshold never requires a deploy.
- **Alerts are state-based.** Events open only on OK→WARN/CRIT transitions and
  auto-resolve on return to OK. No per-cycle spam.

## 3. Data model (migration `00013_monitoring.sql`, goose)

### `monitoring_metrics`
```sql
CREATE TABLE monitoring_metrics (
    id              bigserial PRIMARY KEY,
    organization_id uuid NOT NULL,
    site_id         uuid,                       -- NULL = org-wide
    metric_key      text NOT NULL,              -- e.g. 'ops.mttr_hours'
    dimensions      jsonb NOT NULL DEFAULT '{}'::jsonb,  -- {"severity":"HIGH","model":"..."}
    granularity     text NOT NULL DEFAULT 'day',          -- 'day' | 'latest'
    day             date,                       -- for 'day' rows (metric day)
    value           numeric NOT NULL,
    sample_count    bigint NOT NULL DEFAULT 1,  -- denominator for rates/avgs
    collected_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, site_id, metric_key, dimensions, granularity, day)
);
-- partial unique so 'latest' rows have one row per key (day IS NULL)
CREATE UNIQUE INDEX monitoring_metrics_latest_key
    ON monitoring_metrics (organization_id, site_id, metric_key, dimensions)
    WHERE granularity = 'latest';
CREATE INDEX monitoring_metrics_lookup
    ON monitoring_metrics (organization_id, metric_key, day DESC);
```

`granularity = 'latest'` is upserted each snapshot cycle (worker health,
backup freshness, API error rate now). `granularity = 'day'` is upserted once
per calendar day (MTTR, rates, costs). `dimensions` carries the breakdown
axis (severity, state, queue, model, prompt_version, endpoint).

### `monitor_thresholds`
```sql
CREATE TABLE monitor_thresholds (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    site_id         uuid,                       -- NULL = org-wide default
    metric_key      text NOT NULL,
    comparator      text NOT NULL,              -- 'GT' | 'GTE' | 'LT' | 'LTE'
    warn_value      numeric NOT NULL,
    crit_value      numeric NOT NULL,
    enabled         boolean NOT NULL DEFAULT true,
    updated_by      uuid NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, site_id, metric_key)
);
```

Evaluation: if `crit_value` comparator is true → CRIT; else if `warn_value`
comparator is true → WARN; else PASS. An org-wide row (`site_id NULL`) is the
default; a site row overrides it. `metric_key` values are a fixed catalog in
the domain layer — the UI offers a picker, not free text.

### `monitor_alert_events`
```sql
CREATE TABLE monitor_alert_events (
    id              bigserial PRIMARY KEY,
    organization_id uuid NOT NULL,
    site_id         uuid,
    metric_key      text NOT NULL,
    dimensions      jsonb NOT NULL DEFAULT '{}'::jsonb,
    state           text NOT NULL,              -- 'WARN' | 'CRIT'
    message         text NOT NULL,              -- human-readable, i18n key + params
    value           numeric NOT NULL,           -- value that triggered it
    opened_at       timestamptz NOT NULL DEFAULT now(),
    resolved_at     timestamptz
);
-- one active event per metric+state at a time
CREATE UNIQUE INDEX monitor_alert_events_active
    ON monitor_alert_events (organization_id, site_id, metric_key, dimensions, state)
    WHERE resolved_at IS NULL;
```

State machine per (site, metric_key, dimensions): PASS → open WARN and/or CRIT
event (transition); WARN/CRIT → PASS resolves all open events; severity
escalation WARN→CRIT opens a CRIT event (WARN stays until PASS). Dismissed
alerts are out of scope for this wave (YAGNI — auto-resolve plus thresholds is
enough for pilot).

### Supporting write tables (produced outside the jobs)

Two small write paths exist today that the monitors need:

```sql
CREATE TABLE monitoring_backup_runs (
    id           uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    site_id      uuid,                 -- NULL = org-wide
    started_at   timestamptz NOT NULL,
    completed_at timestamptz,
    status       text NOT NULL,        -- 'RUNNING' | 'SUCCESS' | 'FAILED'
    size_bytes   bigint
);

CREATE TABLE monitoring_request_counters (
    id           bigserial PRIMARY KEY,
    organization_id uuid NOT NULL,
    site_id      uuid,
    endpoint     text NOT NULL,        -- e.g. '/incidents'
    status_class text NOT NULL,        -- '2xx' | '4xx' | '5xx'
    window_start timestamptz NOT NULL, -- 5-minute bucket
    count        bigint NOT NULL DEFAULT 0,
    UNIQUE (organization_id, site_id, endpoint, status_class, window_start)
);
```

- `monitoring_backup_runs` is written by the backup script (runbook +
  `scripts/backup-postgres.sh` change, Phase 1): insert RUNNING at start,
  flip to SUCCESS/FAILED with size at end. `sys.backup_freshness_hours`
  derives from the latest SUCCESS row.
- `monitoring_request_counters` is incremented by the existing `accessLog`
  middleware (route → status class → 5-min bucket, one upsert per request is
  too hot; instead batch via a buffered in-memory counter flushed every 5 min,
  or accept a single upsert per request at pilot scale — decided in planning).
  `sys.api_error_rate_5xx` and `sys.auth_failure_count` aggregate this table.
  Auth failures increment a synthetic `/auth/*` endpoint from the callback
  handlers.

## 4. Monitor catalog

Keys below are the fixed catalog. Group A/B/D are `day` rollups; group C is
`latest` snapshots.

### Group A — Operational KPIs (daily rollup)
| Key | Metric | Dimensions |
|---|---|---|
| `ops.mttr_hours` | mean time to resolve (resolved_at − detected_at) | severity |
| `ops.incident_max_age_hours` | max age of OPEN/IN_PROGRESS incidents | state |
| `ops.incident_backlog_count` | open older than 24h / 7d | bucket (24h, 7d) |
| `ops.execution_cycle_hours` | avg ASSIGNED→COMPLETED duration | — |
| `ops.execution_overdue_count` | executions past SLA (baseline 48h default, configurable) | — |
| `ops.loto_blocked_count` | intrusive step blocks without energy isolation | — |
| `ops.loto_verified_count` | energy-isolation verifications performed | — |
| `ops.handover_gap_count` | shifts with no handover record | — |
| `ops.handover_accept_latency_hours` | avg submitted→accepted | — |
| `ops.report_cycle_hours` | avg DRAFT→APPROVED | — |
| `ops.report_pending_count` | reports stuck in DRAFT/SUBMITTED | state |
| `ops.assets_without_criticality_count` | assets with no approved criticality | — |
| `ops.documents_ingest_failed_count` | revision ingestion failures | — |
| `ops.documents_stale_count` | no approved revision in N months (config 12) | — |

LOTO counts come from execution domain events (already appended for
demonstrations); the daily rollup aggregates event types, no new capture
surface.

### Group B — AI/copilot trends (daily rollup)
Reuses the existing `evaluation/adapter/postgres/store.go` count SQL but
grouped by day, with dimensions `model` / `prompt_version` for drift:
| Key | Metric |
|---|---|
| `ai.acceptance_rate` | accepted ÷ reviewed |
| `ai.unsafe_rate` | unsafe ÷ recommendations |
| `ai.corrected_rate` | corrected ÷ reviewed (learning signal) |
| `ai.insufficient_evidence_rate` | INSUFFICIENT_EVIDENCE outcomes ÷ recommendations |
| `ai.evidence_coverage` | supported claims ÷ material claims |
| `ai.retrieval_precision` | relevant ÷ retrieved evidence |
| `ai.llm_latency_p95_ms` | p95 of recommendation latency (p50 also stored) |
| `ai.llm_cost_usd` | estimated cost (micros) per day |
| `ai.llm_calls` | calls per day |

### Group C — System/infra (5-minute `latest` snapshots + daily where useful)
| Key | Metric | Source |
|---|---|---|
| `sys.worker_queue_depth` | River queue depth | `river_job` table, per queue |
| `sys.worker_retry_count` | jobs in retry state | `river_job` |
| `sys.worker_failed_count` | jobs failed (last 24h, daily too) | `river_job` |
| `sys.ingestion_failed_count` | document ingestion failures (24h) | knowledge revisions |
| `sys.objectstore_upload_failed_count` | attachment upload/checksum failures (24h) | attachments |
| `sys.backup_freshness_hours` | hours since last successful backup | `monitoring_backup_runs` |
| `sys.api_error_rate_5xx` | 5xx per 5-min window, per endpoint | access log → new counter table |
| `sys.health_ready` | 0/1 (DB ping) | readiness check |
| `sys.auth_failure_count` | OIDC/session failures (5-min window) | new counter table |

Backup freshness needs a write path that does not exist yet: the backup
runbook (`docs/runbooks/backup-restore.md`) must gain a step that inserts into
a new small table `monitoring_backup_runs` (started_at, completed_at, status,
size_bytes) after each dump. This is a runbook + script change, not an agent
change. API error rate and auth failures need lightweight counter writes in
the existing `accessLog` middleware and auth callback — two small additions in
existing code, then the snapshot job aggregates them.

### Group D — Workflow learning (daily rollup)
| Key | Metric |
|---|---|
| `learn.learning_coverage_rate` | completed demonstrations ÷ total executions |
| `learn.demo_review_backlog_count` | demonstrations awaiting review |
| `learn.demo_review_latency_hours` | avg capture→review decision |
| `learn.redaction_count` | redactions per day (ambiguity surfaced) |
| `learn.workflow_guided_rate` | executions with an applicable published workflow ÷ total |
| `learn.workflow_review_due_count` | workflows within 30 days of `review_at` |

## 5. Jobs (River)

Both workers live in `internal/monitoring/adapter/river` and register on the
existing `QueueFoundation` (no new queue; small periodic jobs).

- **SnapshotWorker** — `river.PeriodicJob` every 5 minutes. Collects group C
  into `latest` rows, then evaluates thresholds and opens/resolves alert
  events. Each metric collection runs in its own transaction and a single
  metric failure is logged and skipped, never failing the whole cycle.
- **DailyRollupWorker** — `river.PeriodicJob` daily at 00:10 UTC. Collects
  groups A, B, D into `day` rows (upsert by (org, site, key, dimensions, day)
  so replays are safe), then evaluates thresholds for daily metrics.
- Threshold evaluation is shared (`domain`) and unit-tested; both workers call
  the same `EvaluateAlerts` so behavior is identical.
- Both are idempotent by construction: upserts + unique partial index on
  active alert events.
- Config knobs (`MONITORING_SNAPSHOT_INTERVAL`, `MONITORING_DAILY_CRON`) with
  sane defaults, read in `cmd/worker/main.go` alongside existing job wiring.

## 6. API (`internal/platform/httpserver/monitoring.go`)

All endpoints require auth; read endpoints gate on `monitoring:read`,
threshold writes gate on `monitoring:configure`. Site scoping uses the same
`site_id` query pattern as `summary.go` (principal `site_ids` filter).

| Method | Path | Purpose |
|---|---|---|
| GET | `/monitoring/metrics` | series for one or more keys, `site_id`, `from`, `to`, `granularity` |
| GET | `/monitoring/summary` | latest value + derived status for every enabled monitor key |
| GET | `/monitoring/alerts` | alert events, filter `state=OPEN|RESOLVED`, `site_id` |
| GET | `/monitoring/thresholds` | list thresholds for org (+site override) |
| PUT | `/monitoring/thresholds` | upsert a threshold (idempotent) |

Response shapes follow existing `writeJSON`/`ListPage` conventions; metrics
come back ordered by day with `value` and `sample_count`.

New identity permissions: `monitoring:read` and `monitoring:configure`
(identity permissions migration + seed + role matrix — Administrator gets
both, read extends to roles that already hold incident read; exact matrix
decided during planning).

## 7. UI (`/monitoring`)

New sidebar section "Monitoring" (Gauge icon) placed after the Quality
section, gated on `monitoring:read` like Reports is gated on `report:write`.

- **Overview tab** — status summary: one StatusCard per monitor group with
  PASS/WARN/CRIT badges (existing StatusBadge tones), latest values, sparkline
  of the last 30 days, and a count of open alerts per group. Cards deep-link
  to the Trends tab pre-filtered.
- **Trends tab** — metric picker (grouped dropdown of the catalog), range
  selector (7/30/90 days), site from the site switcher. Renders lightweight
  inline-SVG Sparkline/BarSeries components (no chart dependency added to
  `package.json` — consistent with the current minimal-dependency frontend).
  Dimensions (severity/model/queue) render as a small legend; each series has
  a CSV export button.
- **System tab** — group C cards in a grid: worker queue depths, ingestion
  failures, backup freshness (with a direct "RPO breach" CRIT threshold),
  API error rate, auth failures, `/health/ready` status.
- **Alerts tab** — DataTable of alert events (metric, site, state badge,
  value, opened/resolved), filter by state, CSV export.
- **Thresholds tab** — DataTable editor of the catalog: metric key, site
  scope, comparator, warn/crit values, enabled toggle. Writes via PUT with
  the existing `useCommand` toast pattern; only `monitoring:configure`
  holders see the edit affordance.
- **NotificationCenter** — topbar bell showing the count of open CRIT/WARN
  alerts (polls the summary endpoint), a dropdown list of open events, and a
  toast when a new CRIT alert opens (polls every 60s while on the console).

All strings via the existing i18n messages (en + vi), theme-aware, reusing
MetricCard/Skeleton/EmptyState/ErrorState/DataTable; page tests follow the
existing per-page test pattern.

## 8. Error handling & idempotency

- Metrics upsert is `ON CONFLICT DO UPDATE` on the unique keys; replays and
  duplicate job executions are safe.
- Alert events: unique partial index on active events makes open/resolve
  idempotent; a job crash mid-cycle leaves no partial state beyond the
  already-committed metric rows, which is fine for monitoring.
- Job failures use the existing `boundedRetryAt` pattern and are logged; a
  permanently failing collector surfaces as a `sys.*` gap (metrics go stale)
  rather than a crash loop.
- Threshold evaluation is pure domain logic (no I/O) and unit-tested across
  every comparator × severity combination.
- API handlers reuse `validUUIDParam`/`writeProblem` patterns; `monitoring:read`
  without it returns 403 like other permission gates.

## 9. Testing

- **Domain**: threshold evaluation (comparators, WARN/CRIT precedence,
  transition edges, escalate/downgrade), status derivation, catalog validity
  (every key has a display name + unit + direction).
- **Store (postgres)**: upsert idempotency for both granularities, alert
  open/resolve transitions, site scoping, org isolation.
- **Workers**: contract tests that the snapshot/rollup collectors produce the
  right rows from seeded transaction data; periodic registration smoke test.
- **HTTP**: auth required, permission 403, site filter, happy path for all
  four endpoints.
- **Web**: MonitoringPage render + status badges, Trends chart series,
  ThresholdEditor PUT flow, AlertList filtering, i18n key coverage (existing
  `messages.test.ts` pattern), NotificationCenter count.
- **Architecture**: extend `internal/architecture/imports_test.go` so
  `internal/monitoring` never imports `httpserver` or `web`.

## 10. Phasing

- **Phase 1 (pilot cut)** — migration + domain + snapshot job (group C:
  worker health, ingestion failures, backup freshness, health_ready) + daily
  rollup (group A: LOTO safety events, incident aging/MTTR) + thresholds +
  alert events + MonitoringPage Overview/System/Alerts/Thresholds tabs +
  sidebar + permission + notification bell. This is the safety/ops core.
- **Phase 2** — group B AI trend charts + group D learning coverage, Trends
  tab with CSV export, site breakdown legend, configurable thresholds surface
  completed.
- **Phase 3** — remaining group A KPIs (report pipeline, handover coverage,
  assets without criticality, stale docs), API error rate + auth failure
  counters in middleware, per-endpoint breakdowns, runbook/script changes for
  backup marker.

## 11. Open decisions

- **Backup marker**: backup script writes `monitoring_backup_runs` rows
  (doc + script change in Phase 1 so backup freshness is real from day one).
- **New permissions** `monitoring:read`/`monitoring:configure` added to the
  identity role matrix; exact role spread set during planning from the seed
  roles.
- **Worker scheduling**: `river.PeriodicJob` with cron; verify the codebase's
  periodic-job registration pattern during planning (worker currently
  registers one-shot workers only).
- **Threshold defaults**: per-metric defaults seeded in the migration (e.g.
  worker failed > 0 = CRIT, backup older than RPO hours = CRIT, incident age
  > 24h = WARN); all editable at runtime.
