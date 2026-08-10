-- +goose Up
-- Custom fields core: per-tenant field definitions, incident JSONB values, history.
CREATE TABLE field_definitions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    entity_type     text NOT NULL,
    key             text NOT NULL,
    label           text NOT NULL,
    description     text NOT NULL DEFAULT '',
    field_type      text NOT NULL,
    config          jsonb NOT NULL DEFAULT '{}'::jsonb,
    status          text NOT NULL DEFAULT 'ACTIVE',
    sort_order      integer NOT NULL DEFAULT 0,
    version         integer NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    retired_at      timestamptz,
    UNIQUE (organization_id, entity_type, key),
    CHECK (field_type IN ('TEXT', 'NUMBER', 'DATE', 'SELECT', 'MULTI_SELECT')),
    CHECK (status IN ('ACTIVE', 'RETIRED'))
);

ALTER TABLE incidents
    ADD COLUMN custom_values jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE custom_value_history (
    id                 bigserial PRIMARY KEY,
    incident_id        uuid NOT NULL REFERENCES incidents(id),
    field_definition_id uuid NOT NULL REFERENCES field_definitions(id),
    principal_id       uuid NOT NULL REFERENCES principals(id),
    value_before       jsonb,
    value_after        jsonb NOT NULL,
    changed_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX field_definitions_org_idx
    ON field_definitions (organization_id, entity_type, sort_order);
CREATE INDEX incidents_custom_values_gin
    ON incidents USING gin (custom_values);
CREATE INDEX custom_value_history_incident_idx
    ON custom_value_history (incident_id, changed_at DESC);

-- Explicit grants to the API role, following the 00002 convention restored
-- by 00014 for 00013's missing grants.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON field_definitions TO skawld_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON custom_value_history TO skawld_app;
        GRANT USAGE, SELECT ON SEQUENCE custom_value_history_id_seq TO skawld_app;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS custom_value_history;
DROP TABLE IF EXISTS field_definitions;
ALTER TABLE incidents DROP COLUMN IF EXISTS custom_values;
