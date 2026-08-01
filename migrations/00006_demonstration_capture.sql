-- +goose Up
ALTER TABLE domain_events
    ADD COLUMN site_id uuid REFERENCES sites(id),
    ADD COLUMN actor_id uuid REFERENCES principals(id),
    ADD COLUMN subject_kind text,
    ADD COLUMN subject_id uuid,
    ADD CONSTRAINT domain_events_subject_ck CHECK (
        (subject_kind IS NULL AND subject_id IS NULL)
        OR
        (subject_kind IN ('EXECUTION', 'HANDOVER') AND subject_id IS NOT NULL)
    );

CREATE INDEX domain_events_subject_time_idx
    ON domain_events(organization_id, subject_kind, subject_id, occurred_at);

CREATE TABLE demonstrations (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    subject_kind text NOT NULL CHECK (subject_kind IN ('EXECUTION', 'HANDOVER')),
    subject_id uuid NOT NULL,
    workflow_key text NOT NULL CHECK (btrim(workflow_key) <> ''),
    sdk_schema_version text NOT NULL,
    sdk_session_id uuid NOT NULL UNIQUE,
    principal_snapshot jsonb NOT NULL,
    status text NOT NULL CHECK (status IN ('recording', 'completed', 'rejected')),
    review_status text NOT NULL DEFAULT 'PENDING'
        CHECK (review_status IN ('PENDING', 'APPROVED', 'REJECTED', 'REDACTION_REQUIRED')),
    initial_context jsonb NOT NULL DEFAULT '{}'::jsonb,
    final_result jsonb,
    started_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (
        (status = 'recording' AND completed_at IS NULL)
        OR
        (status <> 'recording' AND completed_at IS NOT NULL)
    )
);

CREATE INDEX demonstrations_scope_time_idx
    ON demonstrations(organization_id, site_id, started_at DESC);
CREATE INDEX demonstrations_subject_idx
    ON demonstrations(organization_id, subject_kind, subject_id, started_at DESC);

CREATE TABLE demonstration_events (
    id uuid PRIMARY KEY,
    demonstration_id uuid NOT NULL REFERENCES demonstrations(id),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    schema_version text NOT NULL,
    session_id uuid NOT NULL,
    principal_snapshot jsonb NOT NULL,
    occurred_at timestamptz NOT NULL,
    source text NOT NULL,
    trust text NOT NULL,
    sensitivity text NOT NULL,
    application text,
    action text NOT NULL CHECK (btrim(action) <> ''),
    intent text,
    entity jsonb,
    input_value jsonb,
    output_value jsonb,
    context_value jsonb,
    decision_value jsonb,
    result_value jsonb,
    error_value text,
    correction_of uuid,
    approval_id text,
    domain_event_id uuid REFERENCES domain_events(id),
    provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    UNIQUE (demonstration_id, ordinal),
    UNIQUE (demonstration_id, domain_event_id),
    FOREIGN KEY (correction_of) REFERENCES demonstration_events(id)
);

CREATE INDEX demonstration_events_timeline_idx
    ON demonstration_events(demonstration_id, ordinal);

CREATE TABLE demonstration_capture_deliveries (
    demonstration_id uuid NOT NULL REFERENCES demonstrations(id),
    domain_event_id uuid NOT NULL REFERENCES domain_events(id),
    status text NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'APPLIED', 'FAILED')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at timestamptz NOT NULL,
    last_error text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (demonstration_id, domain_event_id)
);

CREATE INDEX demonstration_capture_due_idx
    ON demonstration_capture_deliveries(next_attempt_at, created_at)
    WHERE status IN ('PENDING', 'FAILED');

CREATE TABLE demonstration_reviews (
    id uuid PRIMARY KEY,
    demonstration_id uuid NOT NULL REFERENCES demonstrations(id),
    decision text NOT NULL
        CHECK (decision IN ('APPROVED', 'REJECTED', 'REDACTION_REQUIRED')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    reviewed_by uuid NOT NULL REFERENCES principals(id),
    reviewed_at timestamptz NOT NULL
);

CREATE INDEX demonstration_reviews_timeline_idx
    ON demonstration_reviews(demonstration_id, reviewed_at);

CREATE TABLE demonstration_event_redactions (
    id uuid PRIMARY KEY,
    demonstration_id uuid NOT NULL REFERENCES demonstrations(id),
    event_id uuid NOT NULL REFERENCES demonstration_events(id),
    json_path text NOT NULL CHECK (btrim(json_path) <> ''),
    action text NOT NULL CHECK (action IN ('MASK', 'DROP')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    requested_by uuid NOT NULL REFERENCES principals(id),
    requested_at timestamptz NOT NULL,
    UNIQUE (event_id, json_path)
);

-- Capture delivery creation is part of the committed domain-event outbox.
-- SDK persistence and mapping happen later in the worker and cannot roll back
-- the authoritative maintenance transaction.
-- +goose StatementBegin
CREATE FUNCTION enqueue_demonstration_capture() RETURNS trigger AS $$
BEGIN
    IF NEW.subject_kind IS NULL OR NEW.subject_id IS NULL THEN
        RETURN NEW;
    END IF;

    INSERT INTO demonstration_capture_deliveries (
        demonstration_id, domain_event_id, status, attempts,
        next_attempt_at, created_at, updated_at
    )
    SELECT d.id, NEW.id, 'PENDING', 0, NEW.occurred_at, NEW.occurred_at, NEW.occurred_at
    FROM demonstrations d
    WHERE d.organization_id = NEW.organization_id
      AND d.subject_kind = NEW.subject_kind
      AND d.subject_id = NEW.subject_id
      AND d.status = 'recording'
      AND d.started_at <= NEW.occurred_at
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER domain_events_demonstration_capture
AFTER INSERT ON domain_events
FOR EACH ROW EXECUTE FUNCTION enqueue_demonstration_capture();

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE ON domain_events TO skawld_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            demonstrations,
            demonstration_events,
            demonstration_capture_deliveries,
            demonstration_reviews,
            demonstration_event_redactions
        TO skawld_app;
        REVOKE DELETE, TRUNCATE ON
            demonstrations,
            demonstration_events,
            demonstration_reviews,
            demonstration_event_redactions
        FROM skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS domain_events_demonstration_capture ON domain_events;
DROP FUNCTION IF EXISTS enqueue_demonstration_capture();
DROP TABLE IF EXISTS demonstration_event_redactions;
DROP TABLE IF EXISTS demonstration_reviews;
DROP TABLE IF EXISTS demonstration_capture_deliveries;
DROP TABLE IF EXISTS demonstration_events;
DROP TABLE IF EXISTS demonstrations;
DROP INDEX IF EXISTS domain_events_subject_time_idx;
ALTER TABLE domain_events
    DROP CONSTRAINT IF EXISTS domain_events_subject_ck,
    DROP COLUMN IF EXISTS subject_id,
    DROP COLUMN IF EXISTS subject_kind,
    DROP COLUMN IF EXISTS actor_id,
    DROP COLUMN IF EXISTS site_id;
