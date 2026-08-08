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
