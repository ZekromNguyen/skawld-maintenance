-- +goose Up
-- Repair databases where migration 00013 was recorded as applied but its DDL
-- never landed, leaving incidents on the pre-redesign schema (state/severity,
-- no details/assignee/reporter/team columns, no teams tables). The API queries
-- the redesigned columns, so every incident read failed with 500.
--
-- The whole block is gated on the pre-redesign "state" column still being
-- present. On databases where 00013 applied correctly the gate is false and
-- this migration is a no-op; on drifted databases it converges to the same end
-- state as 00013.

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'incidents' AND column_name = 'state'
  ) THEN
    ALTER TABLE incidents RENAME COLUMN state TO status;
    ALTER TABLE incidents RENAME COLUMN severity TO priority;

    ALTER TABLE incidents DROP CONSTRAINT IF EXISTS incidents_state_check;
    ALTER TABLE incidents DROP CONSTRAINT IF EXISTS incidents_severity_check;

    ALTER TABLE incidents DROP CONSTRAINT IF EXISTS incidents_resolution_ck;
    ALTER TABLE incidents ADD CONSTRAINT incidents_resolution_ck CHECK (
        (status IN ('OPEN', 'IN_PROGRESS', 'REOPENED') AND resolved_at IS NULL AND resolution_summary IS NULL)
        OR
        (status IN ('RESOLVED', 'CLOSED') AND resolved_at IS NOT NULL AND btrim(resolution_summary) <> '')
    );

    ALTER TABLE incidents ADD CONSTRAINT incidents_status_check CHECK (
        status IN ('OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED', 'REOPENED')
    );
    ALTER TABLE incidents ADD CONSTRAINT incidents_priority_check CHECK (
        priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')
    );

    CREATE TABLE IF NOT EXISTS teams (
        id uuid PRIMARY KEY,
        organization_id uuid NOT NULL REFERENCES organizations(id),
        name text NOT NULL,
        created_at timestamptz NOT NULL,
        UNIQUE (organization_id, name)
    );

    CREATE TABLE IF NOT EXISTS team_members (
        team_id uuid NOT NULL REFERENCES teams(id),
        principal_id uuid NOT NULL REFERENCES principals(id),
        PRIMARY KEY (team_id, principal_id)
    );

    ALTER TABLE incidents ADD COLUMN IF NOT EXISTS details text;
    ALTER TABLE incidents ADD COLUMN IF NOT EXISTS assignee_id uuid REFERENCES principals(id);
    ALTER TABLE incidents ADD COLUMN IF NOT EXISTS reporter_id uuid REFERENCES principals(id);
    ALTER TABLE incidents ADD COLUMN IF NOT EXISTS team_id uuid REFERENCES teams(id);

    -- Backfill: date means "when it occurred"; reporter means who filed it.
    UPDATE incidents SET occurred_at = detected_at WHERE occurred_at IS NULL;
    UPDATE incidents SET reporter_id = created_by WHERE reporter_id IS NULL;

    -- Seed default teams for existing organizations.
    INSERT INTO teams (id, organization_id, name, created_at)
    SELECT gen_random_uuid(), o.id, t.name, now()
    FROM organizations o
    CROSS JOIN (VALUES ('Facilities'), ('Electrical'), ('HVAC')) AS t(name)
    ON CONFLICT (organization_id, name) DO NOTHING;
  END IF;
END $$;
-- +goose StatementEnd

-- Migration 00013 created teams/team_members but never granted them to the API
-- role, so every query joining teams failed with permission denied even on
-- correctly migrated databases. Restore the convention used by every other
-- migration: explicit grants to skawld_app.

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'skawld_app') THEN
    GRANT SELECT, INSERT, UPDATE, DELETE ON teams, team_members TO skawld_app;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- The repair only ever converges drifted databases to the 00013 end state, so
-- it cannot be reversed independently; rollback is handled by 00013's Down.
