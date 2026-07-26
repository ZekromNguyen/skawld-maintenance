-- +goose Up
CREATE TABLE execution_decisions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    execution_id uuid NOT NULL REFERENCES maintenance_executions(id),
    step_id uuid REFERENCES execution_steps(id),
    component_id uuid REFERENCES asset_components(id),
    decision text NOT NULL CHECK (btrim(decision) <> ''),
    rationale text NOT NULL CHECK (btrim(rationale) <> ''),
    alternatives jsonb NOT NULL DEFAULT '[]'::jsonb,
    decided_by uuid NOT NULL REFERENCES principals(id),
    decided_at timestamptz NOT NULL,
    client_event_id uuid,
    device_id text,
    created_at_device timestamptz,
    received_at_server timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, client_event_id)
);

CREATE INDEX execution_decisions_execution_time_idx
    ON execution_decisions(execution_id, decided_at);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON execution_decisions TO skawld_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS execution_decisions;
