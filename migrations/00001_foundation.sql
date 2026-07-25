-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;
CREATE SCHEMA IF NOT EXISTS river;

CREATE TABLE organizations (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> ''),
    source_of_truth text NOT NULL CHECK (source_of_truth IN ('OWNED_BY_SKAWLD', 'EXTERNAL_REFERENCE')),
    external_system text,
    external_id text,
    external_version text,
    sync_status text,
    last_synced_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT organizations_external_source_ck CHECK (
        (source_of_truth = 'OWNED_BY_SKAWLD' AND external_system IS NULL AND external_id IS NULL)
        OR
        (source_of_truth = 'EXTERNAL_REFERENCE' AND external_system IS NOT NULL AND external_id IS NOT NULL)
    ),
    UNIQUE (external_system, external_id)
);

CREATE TABLE sites (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    code text NOT NULL CHECK (btrim(code) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    timezone text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (organization_id, code)
);

CREATE TABLE principals (
    id uuid PRIMARY KEY,
    external_subject text NOT NULL UNIQUE,
    display_name text NOT NULL,
    email text,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE memberships (
    id uuid PRIMARY KEY,
    principal_id uuid NOT NULL REFERENCES principals(id),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid REFERENCES sites(id),
    role text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (principal_id, organization_id, site_id, role)
);

CREATE INDEX memberships_principal_idx ON memberships(principal_id);
CREATE INDEX memberships_scope_idx ON memberships(organization_id, site_id);

CREATE TABLE approval_authorities (
    id uuid PRIMARY KEY,
    subject_id uuid NOT NULL REFERENCES principals(id),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    site_id uuid REFERENCES sites(id),
    scope_kind text,
    scope_id text,
    competency text,
    maximum_risk smallint NOT NULL CHECK (maximum_risk BETWEEN 0 AND 4),
    valid_from timestamptz,
    valid_until timestamptz,
    revoked_at timestamptz,
    delegated_by_subject_id uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL,
    CONSTRAINT approval_authority_validity_ck CHECK (
        valid_until IS NULL OR valid_from IS NULL OR valid_until > valid_from
    )
);

CREATE INDEX approval_authorities_subject_scope_idx
    ON approval_authorities(subject_id, organization_id, site_id);

CREATE TABLE auth_flows (
    state_hash bytea PRIMARY KEY,
    nonce text NOT NULL,
    pkce_verifier text NOT NULL,
    return_to text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX auth_flows_expiry_idx ON auth_flows(expires_at);

CREATE TABLE web_sessions (
    token_hash bytea PRIMARY KEY,
    principal_id uuid NOT NULL REFERENCES principals(id),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX web_sessions_principal_idx ON web_sessions(principal_id);
CREATE INDEX web_sessions_expiry_idx ON web_sessions(expires_at);

CREATE TABLE idempotency_keys (
    principal_id uuid NOT NULL,
    scope text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash bytea NOT NULL,
    status text NOT NULL CHECK (status IN ('PROCESSING', 'COMPLETED')),
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    PRIMARY KEY (principal_id, scope, idempotency_key)
);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY,
    organization_id uuid,
    site_id uuid,
    actor_id uuid,
    action text NOT NULL,
    entity_kind text NOT NULL,
    entity_id text NOT NULL,
    reason text,
    request_id text,
    execution_id text,
    workflow_id text,
    approval_id text,
    ai_involvement jsonb,
    before_value jsonb,
    after_value jsonb,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL
);

CREATE INDEX audit_events_scope_time_idx
    ON audit_events(organization_id, site_id, occurred_at DESC);
CREATE INDEX audit_events_entity_time_idx
    ON audit_events(entity_kind, entity_id, occurred_at DESC);
CREATE INDEX audit_events_actor_time_idx
    ON audit_events(actor_id, occurred_at DESC);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT USAGE ON SCHEMA public TO skawld_app;
        GRANT USAGE ON SCHEMA river TO skawld_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA river
            GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO skawld_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA river
            GRANT USAGE, SELECT ON SEQUENCES TO skawld_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            organizations,
            sites,
            principals,
            memberships,
            approval_authorities,
            auth_flows,
            web_sessions,
            idempotency_keys
        TO skawld_app;
        GRANT SELECT, INSERT ON audit_events TO skawld_app;
        REVOKE UPDATE, DELETE, TRUNCATE ON audit_events FROM skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS auth_flows;
DROP TABLE IF EXISTS approval_authorities;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS principals;
DROP TABLE IF EXISTS sites;
DROP TABLE IF EXISTS organizations;
DROP SCHEMA IF EXISTS river CASCADE;
