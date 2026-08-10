-- +goose Up
CREATE TABLE assets (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    tag text NOT NULL CHECK (btrim(tag) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    asset_class text NOT NULL CHECK (btrim(asset_class) <> ''),
    manufacturer text,
    model text,
    status text NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'OUT_OF_SERVICE', 'RETIRED')),
    source_of_truth text NOT NULL
        CHECK (source_of_truth IN ('OWNED_BY_SKAWLD', 'EXTERNAL_REFERENCE')),
    external_system text,
    external_id text,
    external_version text,
    sync_status text,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT assets_external_source_ck CHECK (
        (source_of_truth = 'OWNED_BY_SKAWLD' AND external_system IS NULL AND external_id IS NULL)
        OR
        (source_of_truth = 'EXTERNAL_REFERENCE' AND external_system IS NOT NULL AND external_id IS NOT NULL)
    ),
    UNIQUE (site_id, tag),
    UNIQUE (organization_id, external_system, external_id)
);

CREATE INDEX assets_scope_class_idx ON assets(organization_id, site_id, asset_class);

CREATE TABLE asset_relationships (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    parent_asset_id uuid NOT NULL REFERENCES assets(id),
    child_asset_id uuid NOT NULL REFERENCES assets(id),
    relationship_type text NOT NULL DEFAULT 'CONTAINS'
        CHECK (relationship_type IN ('CONTAINS', 'SERVES', 'CONNECTED_TO')),
    created_at timestamptz NOT NULL,
    CHECK (parent_asset_id <> child_asset_id),
    UNIQUE (parent_asset_id, child_asset_id, relationship_type)
);

CREATE INDEX asset_relationships_child_idx ON asset_relationships(child_asset_id);

CREATE TABLE asset_components (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    code text NOT NULL CHECK (btrim(code) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    component_type text NOT NULL CHECK (btrim(component_type) <> ''),
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    UNIQUE (asset_id, code)
);

CREATE TABLE asset_criticalities (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    rating text NOT NULL CHECK (rating IN ('A', 'B', 'C')),
    safety_impact smallint NOT NULL CHECK (safety_impact BETWEEN 0 AND 5),
    production_impact smallint NOT NULL CHECK (production_impact BETWEEN 0 AND 5),
    environmental_impact smallint NOT NULL CHECK (environmental_impact BETWEEN 0 AND 5),
    financial_impact smallint NOT NULL CHECK (financial_impact BETWEEN 0 AND 5),
    redundancy text NOT NULL CHECK (redundancy IN ('NONE', 'PARTIAL', 'FULL', 'UNKNOWN')),
    rationale text NOT NULL CHECK (btrim(rationale) <> ''),
    approved_by uuid NOT NULL REFERENCES principals(id),
    approved_at timestamptz NOT NULL,
    superseded_at timestamptz
);

CREATE UNIQUE INDEX asset_criticalities_current_idx
    ON asset_criticalities(asset_id) WHERE superseded_at IS NULL;

CREATE TABLE incidents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    number text NOT NULL,
    summary text NOT NULL CHECK (btrim(summary) <> ''),
    severity text NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    state text NOT NULL CHECK (state IN ('OPEN', 'IN_PROGRESS', 'RESOLVED')),
    source_of_truth text NOT NULL
        CHECK (source_of_truth IN ('OWNED_BY_SKAWLD', 'EXTERNAL_REFERENCE')),
    external_system text,
    external_id text,
    external_version text,
    occurred_at timestamptz,
    detected_at timestamptz NOT NULL,
    resolved_at timestamptz,
    resolution_summary text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT incidents_resolution_ck CHECK (
        (state <> 'RESOLVED' AND resolved_at IS NULL AND resolution_summary IS NULL)
        OR
        (state = 'RESOLVED' AND resolved_at IS NOT NULL AND btrim(resolution_summary) <> '')
    ),
    CONSTRAINT incidents_external_source_ck CHECK (
        (source_of_truth = 'OWNED_BY_SKAWLD' AND external_system IS NULL AND external_id IS NULL)
        OR
        (source_of_truth = 'EXTERNAL_REFERENCE' AND external_system IS NOT NULL AND external_id IS NOT NULL)
    ),
    UNIQUE (organization_id, number),
    UNIQUE (organization_id, external_system, external_id)
);

CREATE INDEX incidents_scope_state_idx ON incidents(organization_id, site_id, state, detected_at DESC);
CREATE INDEX incidents_asset_idx ON incidents(asset_id, detected_at DESC);

CREATE TABLE maintenance_executions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    incident_id uuid REFERENCES incidents(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    purpose text NOT NULL CHECK (btrim(purpose) <> ''),
    state text NOT NULL CHECK (state IN ('ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    assigned_to uuid REFERENCES principals(id),
    workflow_id text,
    workflow_version integer,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    started_at timestamptz,
    completed_at timestamptz,
    outcome_summary text,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX maintenance_executions_scope_state_idx
    ON maintenance_executions(organization_id, site_id, state, created_at DESC);

CREATE TABLE execution_steps (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    step_key text NOT NULL,
    sequence integer NOT NULL CHECK (sequence > 0),
    title text NOT NULL,
    state text NOT NULL CHECK (state IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'BLOCKED')),
    risk_level text NOT NULL
        CHECK (risk_level IN ('INFORMATIONAL', 'ADVISORY', 'OPERATIONAL_LOW', 'SAFETY_SIGNIFICANT')),
    required_prerequisite text,
    blocked_reason text,
    started_at timestamptz,
    completed_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (execution_id, step_key),
    UNIQUE (execution_id, sequence)
);

CREATE TABLE prerequisite_verifications (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    prerequisite_type text NOT NULL
        CHECK (prerequisite_type IN ('WORK_PERMIT', 'ENERGY_ISOLATION', 'GAS_TEST', 'SUPERVISOR', 'COMPETENCY')),
    status text NOT NULL CHECK (status IN ('VERIFIED', 'REJECTED', 'EXPIRED')),
    external_reference text NOT NULL CHECK (btrim(external_reference) <> ''),
    verified_by uuid NOT NULL REFERENCES principals(id),
    verified_at timestamptz NOT NULL,
    valid_until timestamptz,
    created_at timestamptz NOT NULL
);

CREATE INDEX prerequisite_current_idx
    ON prerequisite_verifications(execution_id, prerequisite_type, verified_at DESC);

CREATE TABLE measurements (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    component_id uuid REFERENCES asset_components(id),
    measurement_type text NOT NULL CHECK (measurement_type IN ('VIBRATION_VELOCITY', 'TEMPERATURE')),
    value numeric(20, 6) NOT NULL,
    unit text NOT NULL CHECK (unit IN ('MM_PER_S', 'IN_PER_S', 'DEG_C', 'DEG_F')),
    original_value text NOT NULL,
    original_unit text NOT NULL,
    source text NOT NULL CHECK (source IN ('MANUAL', 'INSTRUMENT', 'EXTERNAL_SYSTEM')),
    data_quality text NOT NULL CHECK (data_quality IN ('GOOD', 'QUESTIONABLE', 'BAD', 'UNKNOWN')),
    instrument_reference text,
    verification_status text NOT NULL CHECK (verification_status IN ('UNVERIFIED', 'VERIFIED', 'REJECTED')),
    observed_at timestamptz NOT NULL,
    recorded_by uuid NOT NULL REFERENCES principals(id),
    client_event_id uuid,
    device_id text,
    created_at_device timestamptz,
    received_at_server timestamptz NOT NULL,
    supersedes_id uuid REFERENCES measurements(id),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, client_event_id)
);

CREATE INDEX measurements_execution_time_idx ON measurements(execution_id, observed_at);

CREATE TABLE observations (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    asset_id uuid NOT NULL REFERENCES assets(id),
    component_id uuid REFERENCES asset_components(id),
    property text,
    status text,
    narrative text NOT NULL CHECK (btrim(narrative) <> ''),
    source text NOT NULL CHECK (source IN ('TECHNICIAN', 'VOICE_TRANSCRIPT', 'VISION_CANDIDATE', 'EXTERNAL_SYSTEM')),
    verification_status text NOT NULL CHECK (verification_status IN ('UNVERIFIED', 'VERIFIED', 'REJECTED')),
    observed_at timestamptz NOT NULL,
    recorded_by uuid NOT NULL REFERENCES principals(id),
    client_event_id uuid,
    device_id text,
    created_at_device timestamptz,
    received_at_server timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, client_event_id)
);

CREATE TABLE maintenance_actions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    step_id uuid REFERENCES execution_steps(id),
    component_id uuid REFERENCES asset_components(id),
    action_type text NOT NULL CHECK (action_type IN ('INSPECTED', 'CLEANED', 'LUBRICATED', 'ADJUSTED', 'REPLACED', 'OTHER')),
    narrative text NOT NULL CHECK (btrim(narrative) <> ''),
    outcome text,
    performed_by uuid NOT NULL REFERENCES principals(id),
    performed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE attachments (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid NOT NULL REFERENCES sites(id),
    entity_kind text NOT NULL CHECK (entity_kind IN ('EXECUTION', 'OBSERVATION', 'INCIDENT')),
    entity_id uuid NOT NULL,
    object_key text NOT NULL UNIQUE,
    original_filename text NOT NULL,
    declared_mime text NOT NULL,
    verified_mime text,
    size_bytes bigint NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 52428800),
    checksum_sha256 text NOT NULL CHECK (checksum_sha256 ~ '^[a-f0-9]{64}$'),
    state text NOT NULL CHECK (state IN ('PENDING_UPLOAD', 'UPLOADED', 'QUARANTINED', 'AVAILABLE', 'REJECTED')),
    rejection_reason text,
    client_event_id uuid,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (organization_id, client_event_id)
);

CREATE TABLE maintenance_reports (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    revision integer NOT NULL CHECK (revision > 0),
    state text NOT NULL CHECK (state IN ('DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED')),
    structured_content jsonb NOT NULL DEFAULT '{}'::jsonb,
    submitted_by uuid REFERENCES principals(id),
    submitted_at timestamptz,
    approved_by uuid REFERENCES principals(id),
    approved_at timestamptz,
    created_at timestamptz NOT NULL,
    UNIQUE (execution_id, revision)
);

CREATE TABLE domain_events (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL,
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE INDEX domain_events_unpublished_idx ON domain_events(occurred_at) WHERE published_at IS NULL;

CREATE TABLE sync_inbox (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    device_id text NOT NULL,
    client_event_id uuid NOT NULL,
    idempotency_key text NOT NULL,
    request_hash bytea NOT NULL,
    payload_version integer NOT NULL,
    base_server_version bigint,
    created_at_device timestamptz NOT NULL,
    received_at_server timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('RECEIVED', 'APPLIED', 'REJECTED', 'CONFLICT')),
    result jsonb,
    server_version bigint,
    PRIMARY KEY (organization_id, device_id, client_event_id)
);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            assets,
            asset_relationships,
            asset_components,
            asset_criticalities,
            incidents,
            maintenance_executions,
            execution_steps,
            prerequisite_verifications,
            measurements,
            observations,
            maintenance_actions,
            attachments,
            maintenance_reports,
            sync_inbox
        TO skawld_app;
        GRANT SELECT, INSERT, UPDATE ON domain_events TO skawld_app;
        REVOKE DELETE, TRUNCATE ON domain_events FROM skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS sync_inbox;
DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS maintenance_reports;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS maintenance_actions;
DROP TABLE IF EXISTS observations;
DROP TABLE IF EXISTS measurements;
DROP TABLE IF EXISTS prerequisite_verifications;
DROP TABLE IF EXISTS execution_steps;
DROP TABLE IF EXISTS maintenance_executions;
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS asset_criticalities;
DROP TABLE IF EXISTS asset_components;
DROP TABLE IF EXISTS asset_relationships;
DROP TABLE IF EXISTS assets;
