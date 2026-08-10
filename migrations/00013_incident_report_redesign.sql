-- +goose Up
-- Incident report redesign: status/priority renames, ticket fields, teams.
ALTER TABLE incidents RENAME COLUMN state TO status;
ALTER TABLE incidents RENAME COLUMN severity TO priority;

-- Rebuild the status check with the extended lifecycle.
ALTER TABLE incidents DROP CONSTRAINT incidents_state_check;
ALTER TABLE incidents ADD CONSTRAINT incidents_status_check CHECK (
    status IN ('OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED', 'REOPENED')
);

ALTER TABLE incidents DROP CONSTRAINT incidents_severity_check;
ALTER TABLE incidents ADD CONSTRAINT incidents_priority_check CHECK (
    priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')
);

-- Resolution invariant now covers CLOSED and REOPENED.
ALTER TABLE incidents DROP CONSTRAINT incidents_resolution_ck;
ALTER TABLE incidents ADD CONSTRAINT incidents_resolution_ck CHECK (
    (status IN ('OPEN', 'IN_PROGRESS', 'REOPENED') AND resolved_at IS NULL AND resolution_summary IS NULL)
    OR
    (status IN ('RESOLVED', 'CLOSED') AND resolved_at IS NOT NULL AND btrim(resolution_summary) <> '')
);

CREATE TABLE teams (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    name text NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, name)
);

CREATE TABLE team_members (
    team_id uuid NOT NULL REFERENCES teams(id),
    principal_id uuid NOT NULL REFERENCES principals(id),
    PRIMARY KEY (team_id, principal_id)
);

ALTER TABLE incidents ADD COLUMN details text;
ALTER TABLE incidents ADD COLUMN assignee_id uuid REFERENCES principals(id);
ALTER TABLE incidents ADD COLUMN reporter_id uuid REFERENCES principals(id);
ALTER TABLE incidents ADD COLUMN team_id uuid REFERENCES teams(id);

-- Backfill: date means "when it occurred"; reporter means who filed it.
UPDATE incidents SET occurred_at = detected_at WHERE occurred_at IS NULL;
UPDATE incidents SET reporter_id = created_by WHERE reporter_id IS NULL;

-- Seed default teams for existing organizations.
INSERT INTO teams (id, organization_id, name, created_at)
SELECT gen_random_uuid(), o.id, t.name, now()
FROM organizations o
CROSS JOIN (VALUES ('Facilities'), ('Electrical'), ('HVAC')) AS t(name)
ON CONFLICT (organization_id, name) DO NOTHING;

-- +goose Down
ALTER TABLE incidents DROP COLUMN IF EXISTS details;
ALTER TABLE incidents DROP COLUMN IF EXISTS assignee_id;
ALTER TABLE incidents DROP COLUMN IF EXISTS reporter_id;
ALTER TABLE incidents DROP COLUMN IF EXISTS team_id;

DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;

ALTER TABLE incidents DROP CONSTRAINT incidents_resolution_ck;
ALTER TABLE incidents ADD CONSTRAINT incidents_resolution_ck CHECK (
    (status <> 'RESOLVED' AND resolved_at IS NULL AND resolution_summary IS NULL)
    OR
    (status = 'RESOLVED' AND resolved_at IS NOT NULL AND btrim(resolution_summary) <> '')
);

ALTER TABLE incidents DROP CONSTRAINT incidents_status_check;
ALTER TABLE incidents ADD CONSTRAINT incidents_state_check CHECK (
    status IN ('OPEN', 'IN_PROGRESS', 'RESOLVED')
);

ALTER TABLE incidents DROP CONSTRAINT incidents_priority_check;
ALTER TABLE incidents ADD CONSTRAINT incidents_severity_check CHECK (
    priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')
);

ALTER TABLE incidents RENAME COLUMN status TO state;
ALTER TABLE incidents RENAME COLUMN priority TO severity;
