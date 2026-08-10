# Incident Report Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the incident record into a ticket-style incident report with status, video/images, summary, details, assignee, priority, date, team, and reporter fields, end to end.

**Architecture:** Extend the existing incident module in place: one migration renames `state`→`status` and `severity`→`priority`, adds `details`/`assignee_id`/`reporter_id`/`team_id`, and creates org-scoped `teams`/`team_members` tables. A small new `internal/team` module serves `GET /api/v1/teams` and `GET /api/v1/people` for the form dropdowns. The existing attachment subsystem (S3) gains video MIME types and a list-by-entity query so incident detail can show a media gallery. The web console's create form, queue, board, and detail page are updated to the new field set.

**Tech Stack:** Go (chi, pgx, goose migrations), PostgreSQL 17 (pgvector), S3-compatible object store, React 19 + TypeScript + Vite + Vitest, OpenAPI 3.

## Global Constraints

- Migration files: goose format (`-- +goose Up` / `-- +goose Down`), placed in `migrations/`, auto-embedded by `go:embed *.sql`.
- DB is PostgreSQL 17 (`pgvector/pgvector:0.8.2-pg17-trixie`); `gen_random_uuid()` is available.
- Multi-tenancy is `organization_id` + `site_id` on every table, derived server-side from the OIDC `Principal`; never client-supplied.
- Renames are field-level only: Go identifiers `Severity`→`Priority`, `State`→`Status` inside `internal/incident`. Wire JSON: `severity`→`priority`, `state`→`status`. Same on the web side.
- Status enum (extended): `OPEN, IN_PROGRESS, RESOLVED, CLOSED, REOPENED`. Priority enum (renamed): `LOW, MEDIUM, HIGH, CRITICAL`.
- Reporter defaults to the logged-in principal (`principal.ID`) when not provided; `occurred_at` defaults to `detected_at` when not provided.
- Every task's requirements implicitly include this section.

---

### Task 1: Database migration

**Files:**
- Create: `migrations/00013_incident_report_redesign.sql`

**Interfaces:**
- Produces: `incidents` columns `status`, `priority`, `details`, `assignee_id`, `reporter_id`, `team_id`; tables `teams(id, organization_id, name, created_at)` and `team_members(team_id, principal_id)`; backfilled `occurred_at`/`reporter_id`; default teams (Facilities, Electrical, HVAC) seeded per existing org.

- [ ] **Step 1: Write the migration**

```sql
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
```

- [ ] **Step 2: Apply the migration**

Run: `MIGRATION_DATABASE_URL="postgres://postgres:postgres_dev_only@localhost:5432/postgres" go run ./cmd/migrate up`
Expected: goose applies `00013` with no error. (If the compose stack is not running: `make compose-up` first.)

- [ ] **Step 3: Verify schema**

Run: `MIGRATION_DATABASE_URL="postgres://postgres:postgres_dev_only@localhost:5432/postgres" go run ./cmd/migrate status`
Expected: `00013_incident_report_redesign.sql` shows as applied. Then spot-check columns:
`psql "$TEST_DATABASE_URL" -c "\d incidents"` shows `status`, `priority`, `details`, `assignee_id`, `reporter_id`, `team_id`, and `\d teams` shows the teams table.

- [ ] **Step 4: Seed default teams for new orgs**

Modify: `internal/identity/adapter/postgres/organization_store.go` — inside `Create`, after the bootstrap Administrator membership insert (around line 86), insert the three default teams in the same transaction:

```go
for _, teamName := range []string{"Facilities", "Electrical", "HVAC"} {
    if _, err := tx.Exec(ctx, `
        INSERT INTO teams (id, organization_id, name, created_at)
        VALUES ($1::uuid, $2::uuid, $3, $4)
        ON CONFLICT (organization_id, name) DO NOTHING
    `, s.IDs.New(), organization.ID, teamName, now); err != nil {
        return txResult{}, fmt.Errorf("seed default team %q: %w", teamName, err)
    }
}
```

- [ ] **Step 5: Run Go tests**

Run: `go test ./internal/identity/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add migrations/00013_incident_report_redesign.sql internal/identity/adapter/postgres/organization_store.go
git commit -m "feat: add incident report ticket fields and teams tables"
```

---

### Task 2: Incident domain model

**Files:**
- Modify: `internal/incident/domain/incident.go`
- Test: `internal/incident/domain/incident_test.go`

**Interfaces:**
- Consumes: migration column names (`status`, `priority`, `details`, `assignee_id`, `reporter_id`, `team_id`).
- Produces: `type Status string` with constants `StatusOpen, StatusInProgress, StatusResolved, StatusClosed, StatusReopened`; `type Priority string` with `PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical`; `Incident` fields `Priority, Status, Details, AssigneeID, ReporterID, TeamID`; methods `Close(at time.Time) error`, `Reopen() error`; `New` validates the priority enum and status set.

- [ ] **Step 1: Write the failing tests**

Add to `internal/incident/domain/incident_test.go`:

```go
func TestNewValidatesPriorityAndDefaultsStatus(t *testing.T) {
	t.Parallel()
	_, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: "UNKNOWN",
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("unsupported priority must fail")
	}
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusOpen {
		t.Fatalf("status = %q, want OPEN", incident.Status)
	}
}

func TestResolveCloseReopenLifecycle(t *testing.T) {
	t.Parallel()
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := incident.Resolve("Bearing replaced", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := incident.Close(time.Now()); err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusClosed {
		t.Fatalf("status = %q, want CLOSED", incident.Status)
	}
	if err := incident.Reopen(); err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusReopened {
		t.Fatalf("status = %q, want REOPENED", incident.Status)
	}
	if incident.ResolvedAt != nil || incident.ResolutionSummary != "" {
		t.Fatal("reopen must clear resolution")
	}
	if err := incident.Close(time.Now()); err == nil {
		t.Fatal("closing a reopened incident directly must fail")
	}
}

func TestStartAllowsReopened(t *testing.T) {
	t.Parallel()
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := incident.Resolve("Bearing replaced", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := incident.Reopen(); err != nil {
		t.Fatal(err)
	}
	if err := incident.Start(); err != nil {
		t.Fatalf("reopened incident must start: %v", err)
	}
	if incident.Status != StatusInProgress {
		t.Fatalf("status = %q, want IN_PROGRESS", incident.Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/incident/domain/ -run 'TestNewValidatesPriority|TestResolveCloseReopenLifecycle|TestStartAllowsReopened' -v`
Expected: FAIL (unknown identifiers `Priority`, `Status`, etc.).

- [ ] **Step 3: Implement the domain changes**

In `internal/incident/domain/incident.go`:

```go
type Status string
type Priority string

const (
	StatusOpen       Status = "OPEN"
	StatusInProgress Status = "IN_PROGRESS"
	StatusResolved   Status = "RESOLVED"
	StatusClosed     Status = "CLOSED"
	StatusReopened   Status = "REOPENED"

	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)
```

Struct: replace `Severity Severity` with `Priority Priority`, `State State` with `Status Status`, and add `Details string`, `AssigneeID string`, `ReporterID string`, `TeamID string`.

`New`: replace the severity switch with a priority switch; after `value.DetectedAt` validation, keep `value.Status = StatusOpen` only when empty:

```go
switch value.Priority {
case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
default:
	return Incident{}, errors.New("unsupported incident priority")
}
...
if value.Status == "" {
	value.Status = StatusOpen
}
```

Update `Start`, `Resolve`, and add:

```go
func (i *Incident) Close(at time.Time) error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.Status != StatusResolved {
		return errors.New("only a resolved incident can be closed")
	}
	if at.IsZero() {
		return errors.New("resolution time is required")
	}
	closedAt := at.UTC()
	if i.ResolvedAt == nil {
		i.ResolvedAt = &closedAt
	}
	i.Status = StatusClosed
	i.Version++
	return nil
}

func (i *Incident) Reopen() error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.Status != StatusResolved {
		return errors.New("only a resolved incident can be reopened")
	}
	i.Status = StatusReopened
	i.ResolutionSummary = ""
	i.ResolvedAt = nil
	i.Version++
	return nil
}
```

`Start`: change the guard to `if i.Status != StatusOpen && i.Status != StatusReopened { return errors.New("only an open or reopened incident can move to in progress") }`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/incident/domain/`
Expected: PASS (existing `TestResolveRequiresNativeIncidentAndSummary` uses `Severity: SeverityHigh` and `State: StateResolved` — update those references to `Priority: PriorityHigh` and `Status: StatusResolved`).

- [ ] **Step 5: Commit**

```bash
git add internal/incident/domain/incident.go internal/incident/domain/incident_test.go
git commit -m "feat: extend incident domain with priority, status lifecycle, and ticket fields"
```

---

### Task 3: Incident application service

**Files:**
- Modify: `internal/incident/application/service.go`

**Interfaces:**
- Consumes: domain `Status`/`Priority`/`Details`/`AssigneeID`/`ReporterID`/`TeamID`, domain `Close`/`Reopen`.
- Produces: `CreateIncident{... Priority string, Status string, Details string, AssigneeID string, ReporterID string, TeamID string}`; `Incident` DTO with `priority`, `status`, `details`, `assignee_id`, `reporter_id`, `team_id`, `assignee_name`, `reporter_name`, `team_name`, `occurred_at`, `time_to_complete_seconds`; `CloseIncident{ExpectedVersion int64}`; `ReopenIncident{ExpectedVersion int64}`; Store interface gains `Close` and `Reopen`.

- [ ] **Step 1: Update the DTOs**

In `internal/incident/application/service.go`:

```go
type CreateIncident struct {
	SiteID          string     `json:"site_id"`
	AssetID         string     `json:"asset_id"`
	Summary         string     `json:"summary"`
	Details         string     `json:"details,omitempty"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status,omitempty"`
	AssigneeID      string     `json:"assignee_id,omitempty"`
	ReporterID      string     `json:"reporter_id,omitempty"`
	TeamID          string     `json:"team_id,omitempty"`
	SourceOfTruth   string     `json:"source_of_truth"`
	ExternalSystem  string     `json:"external_system,omitempty"`
	ExternalID      string     `json:"external_id,omitempty"`
	ExternalVersion string     `json:"external_version,omitempty"`
	OccurredAt      *time.Time `json:"occurred_at,omitempty"`
	DetectedAt      time.Time  `json:"detected_at"`
}

type CloseIncident struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type ReopenIncident struct {
	ExpectedVersion int64 `json:"expected_version"`
}
```

`Incident` DTO: replace `Severity string json:"severity"` with `Priority string json:"priority"`, `State string json:"state"` with `Status string json:"status"`, and add:

```go
Details            string  `json:"details,omitempty"`
AssigneeID         string  `json:"assignee_id,omitempty"`
AssigneeName       string  `json:"assignee_name,omitempty"`
ReporterID         string  `json:"reporter_id,omitempty"`
ReporterName       string  `json:"reporter_name,omitempty"`
TeamID             string  `json:"team_id,omitempty"`
TeamName           string  `json:"team_name,omitempty"`
TimeToCompleteSeconds *int64 `json:"time_to_complete_seconds,omitempty"`
```

`Filter`: rename `State` field to `Status` (JSON tag `status`). Update `Service.Create` validation to check `command.Priority` is one of the four values (reuse a small helper), and `Service.Resolve` stays. Add service methods:

```go
func (s Service) Close(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command CloseIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentResolve) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Close(ctx, principal, key, incidentID, command)
}

func (s Service) Reopen(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command ReopenIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentResolve) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Reopen(ctx, principal, key, incidentID, command)
}
```

Store interface gains `Close(context.Context, identitydomain.Principal, string, string, CloseIncident) (Incident, bool, error)` and `Reopen(context.Context, identitydomain.Principal, string, string, ReopenIncident) (Incident, bool, error)`.

- [ ] **Step 2: Build**

Run: `go build ./internal/incident/... ./internal/platform/httpserver/... ./cmd/...`
Expected: compile errors in callers (`cmd/seed/main.go`, httpserver tests, store) — fix iteratively: `cmd/seed/main.go:142` `Severity: "HIGH"` → `Priority: "HIGH"`; `internal/platform/httpserver/incidents_list_test.go` fake store gains `Close`/`Reopen` stubs; `internal/incident/adapter/postgres/cursor_pagination_integration_test.go` any `state`/`severity` references.

- [ ] **Step 3: Update the list filter call sites**

In `internal/platform/httpserver/incidents.go` `listIncidents`: `State: r.URL.Query().Get("state")` → `Status: r.URL.Query().Get("status")`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/incident/... ./internal/platform/httpserver/ -run 'TestListIncidentsEnvelope' -v`
Expected: PASS (test updates: `Filter{Status: "OPEN"}`).

- [ ] **Step 5: Commit**

```bash
git add internal/incident/application/service.go internal/platform/httpserver/incidents.go cmd/seed/main.go internal/platform/httpserver/incidents_list_test.go internal/incident/adapter/postgres/cursor_pagination_integration_test.go
git commit -m "feat: add incident report ticket fields to the application service"
```

---

### Task 4: Incident postgres store

**Files:**
- Modify: `internal/incident/adapter/postgres/store.go`
- Test: `internal/incident/adapter/postgres/cursor_pagination_integration_test.go`

**Interfaces:**
- Consumes: app `CreateIncident`/`CloseIncident`/`ReopenIncident`, domain model.
- Produces: `Create` writes new columns and defaults reporter/occurred_at; `Close`/`Reopen` execute the lifecycle UPDATEs; `incidentSelect` joins principals and teams; `scanIncident`/`mapIncident` carry the new fields; `time_to_complete_seconds` computed when resolved.

- [ ] **Step 1: Update `Create`**

In `internal/incident/adapter/postgres/store.go` `Create`:

```go
reporterID := command.ReporterID
if reporterID == "" {
	reporterID = principal.ID
}
occurredAt := command.OccurredAt
if occurredAt == nil {
	occurredAt = &command.DetectedAt
}
```

Pass `Details: command.Details, AssigneeID: command.AssigneeID, ReporterID: reporterID, TeamID: command.TeamID` into the `incidentdomain.Incident` literal, and `Priority: incidentdomain.Priority(command.Priority), Status: incidentdomain.Status(command.Status)`.

Replace the INSERT with:

```sql
INSERT INTO incidents (
	id, organization_id, site_id, asset_id, number, summary, details, priority,
	status, source_of_truth, external_system, external_id, external_version,
	occurred_at, detected_at, version, created_by, assignee_id, reporter_id, team_id,
	created_at, updated_at
) VALUES (
	$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, nullif($7, ''), $8,
	$9, $10, nullif($11, ''), nullif($12, ''), nullif($13, ''),
	$14, $15, $16, $17::uuid, nullif($18, '')::uuid, $19::uuid, nullif($20, '')::uuid,
	$21, $21
)
```

with args `incident.ID, incident.OrganizationID, incident.SiteID, incident.AssetID, incident.Number, incident.Summary, incident.Details, incident.Priority, incident.Status, incident.SourceOfTruth, incident.ExternalSystem, incident.ExternalID, incident.ExternalVersion, incident.OccurredAt, incident.DetectedAt, incident.Version, principal.ID, incident.AssigneeID, incident.ReporterID, incident.TeamID, now`.

- [ ] **Step 2: Update the select, scan, and mapping**

Also update the `List` filter SQL: `filter.State` → `filter.Status` and the column reference `i.state = $5` → `i.status = $5` (args array already uses the renamed field).

`incidentSelect`:

```sql
SELECT
	i.id::text, i.organization_id::text, i.site_id::text, i.asset_id::text,
	a.tag, i.number, i.summary, i.priority, i.status, i.source_of_truth,
	coalesce(i.external_system, ''), coalesce(i.external_id, ''),
	coalesce(i.external_version, ''), i.occurred_at, i.detected_at,
	i.resolved_at, coalesce(i.resolution_summary, ''), i.version,
	coalesce(i.details, ''),
	coalesce(i.assignee_id::text, ''), coalesce(assignee.display_name, ''),
	coalesce(i.reporter_id::text, ''), coalesce(reporter.display_name, ''),
	coalesce(i.team_id::text, ''), coalesce(team.name, '')
FROM incidents i
JOIN assets a ON a.id = i.asset_id
LEFT JOIN principals assignee ON assignee.id = i.assignee_id
LEFT JOIN principals reporter ON reporter.id = i.reporter_id
LEFT JOIN teams team ON team.id = i.team_id
```

`scanIncident` and the manual scan in `loadIncidentForUpdate`: scan the five new `coalesce` columns into `&value.Details, &value.AssigneeID, &value.AssigneeName, &value.ReporterID, &value.ReporterName, &value.TeamID, &value.TeamName`.

`mapIncident`:

```go
var timeToComplete *int64
if value.ResolvedAt != nil {
	seconds := int64(value.ResolvedAt.Sub(value.DetectedAt).Seconds())
	timeToComplete = &seconds
}
return incidentapp.Incident{
	...
	Priority: string(value.Priority), Status: string(value.Status),
	Details: value.Details, AssigneeID: value.AssigneeID, ReporterID: value.ReporterID,
	TeamID: value.TeamID, TimeToCompleteSeconds: timeToComplete,
}
```

`loadIncidentForUpdate` returns `Priority: incidentdomain.Priority(value.Priority), Status: incidentdomain.Status(value.Status)` and the new fields.

- [ ] **Step 3: Add `Close` and `Reopen`**

Both follow the `Resolve` transaction shape: idempotency begin (scope `"incident.close.v1:"+incidentID` / `"incident.reopen.v1:"+incidentID`), `loadIncidentForUpdate`, version check, domain method, UPDATE, `appendEvent`, audit append, idempotency complete. Full `Close`:

```go
func (s Store) Close(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.CloseIncident,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.close.v1:" + incidentID
	type outcome struct {
		Value  incidentapp.Incident
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay incidentapp.Incident
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, err
			}
			return outcome{Value: replay, Replay: true}, nil
		}
		incident, assetTag, err := loadIncidentForUpdate(ctx, tx, principal, incidentID)
		if err != nil {
			return outcome{}, err
		}
		if incident.Version != command.ExpectedVersion {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		before := mapIncident(incident, assetTag)
		if err := incident.Close(now); err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET status = $2, resolved_at = $3, version = $4, updated_at = $5
			WHERE id = $1::uuid AND version = $6
		`, incident.ID, incident.Status, incident.ResolvedAt,
			incident.Version, now, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentClosed", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.closed", EntityKind: "incident",
			EntityID: incident.ID, Before: before, After: response, OccurredAt: now,
		}); err != nil {
			return outcome{}, err
		}
		body, _ := json.Marshal(response)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return outcome{}, err
		}
		return outcome{Value: response}, nil
	})
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	return result.Value, result.Replay, nil
}
```

`Reopen` is identical except: scope `"incident.reopen.v1:"+incidentID`, event type `"IncidentReopened"`, action `"incident.reopened"`, domain call `incident.Reopen()`, and UPDATE:

```sql
UPDATE incidents
SET status = $2, resolved_at = NULL, resolution_summary = NULL, version = $3, updated_at = $4
WHERE id = $1::uuid AND version = $5
```

with args `incident.ID, incident.Status, incident.Version, now, command.ExpectedVersion`.

- [ ] **Step 4: Update the integration test fixture**

In `cursor_pagination_integration_test.go`, any INSERT/assertions referencing `state`/`severity` columns become `status`/`priority`. Add an integration assertion: after `Create`, verify `ReporterID == principalID` and `TimeToCompleteSeconds` is nil; after resolving, it is non-nil.

- [ ] **Step 5: Run tests**

Run: `go build ./internal/incident/... && TEST_DATABASE_URL="postgres://postgres:postgres_dev_only@localhost:5432/postgres" go test -count=1 ./internal/incident/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/incident/adapter/postgres/store.go internal/incident/adapter/postgres/cursor_pagination_integration_test.go
git commit -m "feat: persist incident report ticket fields and lifecycle transitions"
```

---

### Task 5: Teams and people module

**Files:**
- Create: `internal/team/application/service.go`
- Create: `internal/team/application/service_test.go`
- Create: `internal/team/adapter/postgres/store.go`

**Interfaces:**
- Produces: `Team{ID, Name}`, `Person{ID, DisplayName}`, `Service.ListTeams(ctx, principal) ([]Team, error)`, `Service.ListPeople(ctx, principal) ([]Person, error)`, `Store` interface with `ListTeams`/`ListPeople`, errors `ErrForbidden`/`ErrInvalid`.

- [ ] **Step 1: Write the failing tests**

`internal/team/application/service_test.go`:

```go
package application

import (
	"context"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type fakeStore struct{}

func (*fakeStore) ListTeams(context.Context, identitydomain.Principal) ([]Team, error) {
	return []Team{{ID: "t1", Name: "Facilities"}}, nil
}
func (*fakeStore) ListPeople(context.Context, identitydomain.Principal) ([]Person, error) {
	return []Person{{ID: "p1", DisplayName: "Ada"}}, nil
}

func TestListTeamsRequiresIncidentCreate(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org",
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	if _, err := service.ListTeams(context.Background(), principal); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	principal.Permissions[identitydomain.PermissionIncidentCreate] = struct{}{}
	teams, err := service.ListTeams(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].Name != "Facilities" {
		t.Fatalf("unexpected teams: %#v", teams)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/team/...`
Expected: FAIL (package does not exist).

- [ ] **Step 3: Implement the service**

`internal/team/application/service.go`:

```go
package application

import (
	"context"
	"errors"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrForbidden = errors.New("team operation forbidden")
	ErrInvalid   = errors.New("invalid team command")
)

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Person struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Store interface {
	ListTeams(context.Context, identitydomain.Principal) ([]Team, error)
	ListPeople(context.Context, identitydomain.Principal) ([]Person, error)
}

type Service struct {
	Store Store
}

func (s Service) ListTeams(ctx context.Context, principal identitydomain.Principal) ([]Team, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return nil, ErrForbidden
	}
	return s.Store.ListTeams(ctx, principal)
}

func (s Service) ListPeople(ctx context.Context, principal identitydomain.Principal) ([]Person, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return nil, ErrForbidden
	}
	return s.Store.ListPeople(ctx, principal)
}
```

`internal/team/adapter/postgres/store.go`:

```go
package postgres

import (
	"context"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func (s Store) ListTeams(ctx context.Context, principal identitydomain.Principal) ([]teamapp.Team, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, name
		FROM teams
		WHERE organization_id = $1::uuid
		ORDER BY name
	`, principal.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	teams := []teamapp.Team{}
	for rows.Next() {
		var team teamapp.Team
		if err := rows.Scan(&team.ID, &team.Name); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}

func (s Store) ListPeople(ctx context.Context, principal identitydomain.Principal) ([]teamapp.Person, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT p.id::text, p.display_name
		FROM principals p
		JOIN memberships m ON m.principal_id = p.id
		WHERE m.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR m.site_id = ANY($2::uuid[]) OR m.site_id IS NULL)
		ORDER BY p.display_name
	`, principal.OrganizationID, principal.SiteIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people := []teamapp.Person{}
	for rows.Next() {
		var person teamapp.Person
		if err := rows.Scan(&person.ID, &person.DisplayName); err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/team/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/team
git commit -m "feat: add teams and people listing for incident reports"
```

---

### Task 6: Attachment video MIME and list-by-entity

**Files:**
- Modify: `internal/attachment/application/service.go`
- Modify: `internal/attachment/application/service_test.go`
- Modify: `internal/attachment/adapter/postgres/store.go`

**Interfaces:**
- Produces: video MIME types `video/mp4`, `video/webm`, `video/quicktime` allowed for `INCIDENT`/`EXECUTION`/`OBSERVATION` kinds; `Attachment` DTO gains `DownloadURL`; `Service.ListByEntity(ctx, principal, entityKind, entityID) ([]Attachment, error)`; Store interface gains `ListByEntity`.

- [ ] **Step 1: Write the failing test**

Add to `internal/attachment/application/service_test.go`:

```go
type listStore struct {
	fakeStore
	attachments []Attachment
}

func (*listStore) ListByEntity(
	context.Context,
	identitydomain.Principal,
	string,
	string,
) ([]Attachment, error) {
	return []Attachment{{ID: "a1", State: "AVAILABLE"}}, nil
}
```

Also add a `ListByEntity` stub to the existing `fakeStore` so it keeps satisfying the widened Store interface:

```go
func (*fakeStore) ListByEntity(
	context.Context,
	identitydomain.Principal,
	string,
	string,
) ([]Attachment, error) {
	return nil, nil
}
```

func TestCreateAllowsVideoForIncidents(t *testing.T) {
	store := &fakeStore{}
	service := Service{Store: store}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org", SiteIDs: []string{"site"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAttachmentWrite: {},
		},
	}
	_, _, err := service.Create(context.Background(), principal, "request-123", CreateManifest{
		SiteID: "site", EntityKind: "INCIDENT", EntityID: "incident",
		ClientEventID: "event", OriginalFilename: "clip.mp4",
		DeclaredMIME: "video/mp4", SizeBytes: 1024,
		ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("video/mp4 must be allowed: %v", err)
	}
}

func TestListByEntityRequiresWritePermission(t *testing.T) {
	service := Service{Store: &listStore{}}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org", SiteIDs: []string{"site"},
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	if _, err := service.ListByEntity(context.Background(), principal, "INCIDENT", "incident"); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/attachment/application/ -run 'TestCreateAllowsVideoForIncidents|TestListByEntityRequiresWritePermission' -v`
Expected: FAIL (video/mp4 rejected; `ListByEntity` undefined).

- [ ] **Step 3: Implement the changes**

In `internal/attachment/application/service.go`:

```go
case "image/jpeg", "image/png", "image/webp", "audio/m4a",
	"audio/mp4", "audio/mpeg", "audio/wav",
	"video/mp4", "video/webm", "video/quicktime":
```

Add `DownloadURL string json:"download_url,omitempty"` to the `Attachment` struct. Add to the Store interface and Service:

```go
func (s Service) ListByEntity(
	ctx context.Context,
	principal identitydomain.Principal,
	entityKind, entityID string,
) ([]Attachment, error) {
	if !principal.Has(identitydomain.PermissionAttachmentWrite) {
		return nil, ErrForbidden
	}
	if entityKind == "" || entityID == "" {
		return nil, ErrInvalid
	}
	switch entityKind {
	case "EXECUTION", "OBSERVATION", "INCIDENT":
	default:
		return nil, ErrInvalid
	}
	return s.Store.ListByEntity(ctx, principal, entityKind, entityID)
}
```

In `internal/attachment/adapter/postgres/store.go`, add a `ListByEntity` method that selects attachments for the entity within the principal's site scope, loads `object_key`, and for `AVAILABLE`/`UPLOADED` states signs a download URL via `s.Objects.SignedDownloadURL(ctx, objectKey, 15*time.Minute)`, setting `Attachment.DownloadURL`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/attachment/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/attachment
git commit -m "feat: allow video attachments and list attachments for an entity"
```

---

### Task 7: HTTP handlers for incidents, teams, people, attachments

**Files:**
- Modify: `internal/platform/httpserver/incidents.go`
- Create: `internal/platform/httpserver/teams.go`
- Modify: `internal/platform/httpserver/attachments.go`
- Modify: `internal/platform/httpserver/router.go`

**Interfaces:**
- Consumes: team service, incident service, attachment service.
- Produces: `GET /api/v1/teams`, `GET /api/v1/people`, `GET /api/v1/attachments?entity_kind=&entity_id=`, `POST /api/v1/incidents/{incidentID}/close`, `POST /api/v1/incidents/{incidentID}/reopen`; incident detail response composed with `attachments`.

- [ ] **Step 1: Wire the routes**

`internal/platform/httpserver/teams.go`:

```go
package httpserver

import (
	"net/http"

	teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"
	"github.com/go-chi/chi/v5"
)

func mountTeamRoutes(router chi.Router, service teamapp.Service) {
	router.Get("/teams", listTeams(service))
	router.Get("/people", listPeople(service))
}

func listTeams(service teamapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		items, err := service.ListTeams(r.Context(), principal)
		if err != nil {
			writeDomainError(w, err, teamapp.ErrForbidden, teamapp.ErrInvalid)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func listPeople(service teamapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		items, err := service.ListPeople(r.Context(), principal)
		if err != nil {
			writeDomainError(w, err, teamapp.ErrForbidden, teamapp.ErrInvalid)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
```

`internal/platform/httpserver/incidents.go`: add routes:

```go
router.Post("/incidents/{incidentID}/close", closeIncident(incidents))
router.Post("/incidents/{incidentID}/reopen", reopenIncident(incidents))
```

Add handlers mirroring `resolveIncident` with `CloseIncident`/`ReopenIncident`. Update `getIncident` to accept the attachments service and compose:

```go
func getIncident(incidents incidentapp.Service, attachments attachmentapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		...
		result, err := incidents.Get(r.Context(), principal, id)
		if err != nil { ... }
		items, err := attachments.ListByEntity(r.Context(), principal, "INCIDENT", id)
		if err != nil {
			writeDomainError(w, err, attachmentapp.ErrForbidden, attachmentapp.ErrInvalid)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			incidentapp.Incident
			Attachments []attachmentapp.Attachment `json:"attachments"`
		}{result, items})
	}
}
```

`internal/platform/httpserver/attachments.go`: add `router.Get("/attachments", listAttachments(service))`; handler reads `entity_kind` and `entity_id` query params, validates `entity_id` is a UUID, calls `service.ListByEntity`, writes `{items: ...}`.

`internal/platform/httpserver/router.go`: add `Teams teamapp.Service` to `Dependencies`, call `mountTeamRoutes(api, dependencies.Teams)`, and pass `dependencies.Attachments` to `mountIncidentRoutes`.

- [ ] **Step 2: Update `mountIncidentRoutes` signature**

`mountIncidentRoutes(router chi.Router, incidents incidentapp.Service, executions executionapp.Service, attachments attachmentapp.Service)` and its call site in `router.go`.

- [ ] **Step 3: Add handler tests**

Add to `internal/platform/httpserver/incidents_list_test.go` a fake attachment service (or extend the existing fake store with the new methods) and a test that `GET /api/v1/incidents/{id}` returns `attachments` in the body. Add a teams test file `teams_list_test.go` with a fake team store and assert `GET /api/v1/teams` and `GET /api/v1/people` return 200 with `items`.

- [ ] **Step 4: Wire in cmd/api/main.go**

Add `teampostgres "github.com/ZekromNguyen/skawld-maintenance/internal/team/adapter/postgres"` and `teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"` imports; construct `teamStore := teampostgres.Store{Pool: pool}`; add `Teams: teamapp.Service{Store: teamStore}` to the `httpserver.Dependencies`.

- [ ] **Step 5: Run tests**

Run: `go build ./cmd/... && go test ./internal/platform/httpserver/ -run 'TestListIncidentsEnvelope|TestTeams|TestPeople' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/platform/httpserver cmd/api/main.go
git commit -m "feat: expose teams, people, and incident attachments over the API"
```

---

### Task 8: OpenAPI specification

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `web/public/openapi.yaml` (copy from `api/openapi.yaml`)

- [ ] **Step 1: Update the incident schemas**

In `api/openapi.yaml` `CreateIncident`: `required: [site_id, asset_id, summary, priority, source_of_truth, detected_at]`; `severity` property → `priority` with the same enum; add `status: {type: string, enum: [OPEN, IN_PROGRESS, RESOLVED, CLOSED, REOPENED]}`, `details`, `assignee_id` (uuid), `reporter_id` (uuid), `team_id` (uuid).

`Incident` schema: `required` swaps `severity`→`priority`, `state`→`status`; properties renamed the same way, plus `details`, `assignee_id`, `assignee_name`, `reporter_id`, `reporter_name`, `team_id`, `team_name`, `occurred_at`, `time_to_complete_seconds` (integer).

`listIncidents` query param: `state` → `status` with the extended enum.

Add paths:

```yaml
  /api/v1/teams:
    get:
      summary: List organization teams
      tags: [Teams]
      operationId: listTeams
      security: [{sessionCookie: []}, {bearerAuth: []}]
      responses:
        "200":
          description: Teams in the organization.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [items]
                properties:
                  items:
                    type: array
                    items:
                      type: object
                      additionalProperties: false
                      required: [id, name]
                      properties:
                        id: {type: string, format: uuid}
                        name: {type: string}
  /api/v1/people:
    get:
      summary: List organization members
      tags: [Teams]
      operationId: listPeople
      security: [{sessionCookie: []}, {bearerAuth: []}]
      responses:
        "200":
          description: Members visible to the principal.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [items]
                properties:
                  items:
                    type: array
                    items:
                      type: object
                      additionalProperties: false
                      required: [id, display_name]
                      properties:
                        id: {type: string, format: uuid}
                        display_name: {type: string}
  /api/v1/attachments:
    get:
      summary: List attachments for an entity
      tags: [Attachments]
      operationId: listAttachments
      security: [{sessionCookie: []}, {bearerAuth: []}]
      parameters:
        - {in: query, name: entity_kind, required: true, schema: {type: string, enum: [EXECUTION, OBSERVATION, INCIDENT]}}
        - {in: query, name: entity_id, required: true, schema: {type: string, format: uuid}}
      responses:
        "200":
          description: Attachments with signed download URLs.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [items]
                properties:
                  items: {type: array, items: {$ref: "#/components/schemas/Attachment"}}
```

Add `/api/v1/incidents/{incidentID}/close` and `/api/v1/incidents/{incidentID}/reopen` POST paths mirroring `/resolution` with `{expected_version}` bodies.

- [ ] **Step 2: Add the Attachment schema**

```yaml
    Attachment:
      type: object
      required: [id, organization_id, site_id, entity_kind, entity_id, original_filename, declared_mime, size_bytes, checksum_sha256, state]
      properties:
        id: {type: string, format: uuid}
        organization_id: {type: string, format: uuid}
        site_id: {type: string, format: uuid}
        entity_kind: {type: string, enum: [EXECUTION, OBSERVATION, INCIDENT, DOCUMENT_REVISION]}
        entity_id: {type: string, format: uuid}
        client_event_id: {type: string}
        original_filename: {type: string}
        declared_mime: {type: string}
        verified_mime: {type: string}
        size_bytes: {type: integer, format: int64}
        checksum_sha256: {type: string}
        state: {type: string}
        download_url: {type: string, format: uri}
```

- [ ] **Step 3: Sync the web copy**

Run: `cp api/openapi.yaml web/public/openapi.yaml`

- [ ] **Step 4: Validate**

Run: `npx --yes @redocly/cli lint api/openapi.yaml`
Expected: no errors (or only pre-existing warnings).

- [ ] **Step 5: Commit**

```bash
git add api/openapi.yaml web/public/openapi.yaml
git commit -m "docs: update OpenAPI for incident report fields, teams, people, attachments"
```

---

### Task 9: Web types and API client

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`

**Interfaces:**
- Produces: `Incident` type with `priority`, `status`, `details`, `assignee_id/name`, `reporter_id/name`, `team_id/name`, `occurred_at`, `time_to_complete_seconds`, `attachments`; `Team`/`Person`/`Attachment` types; `api.teams()`, `api.people()`, `api.listAttachments(entityKind, entityId)`, `api.uploadIncidentAttachment(incident, siteID, file)`, updated `api.createIncident`.

- [ ] **Step 1: Update `web/src/types.ts`**

```ts
export type IncidentStatus = "OPEN" | "IN_PROGRESS" | "RESOLVED" | "CLOSED" | "REOPENED";
export type Priority = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";

export type Incident = {
  id: string;
  site_id: string;
  asset_id: string;
  asset_tag?: string;
  number: string;
  summary: string;
  details?: string;
  priority: Priority;
  status: IncidentStatus;
  assignee_id?: string;
  assignee_name?: string;
  reporter_id?: string;
  reporter_name?: string;
  team_id?: string;
  team_name?: string;
  occurred_at?: string;
  detected_at: string;
  resolved_at?: string;
  time_to_complete_seconds?: number;
  version: number;
  attachments?: Attachment[];
};

export type Team = { id: string; name: string };
export type Person = { id: string; display_name: string };

export type Attachment = {
  id: string;
  organization_id: string;
  site_id: string;
  entity_kind: "EXECUTION" | "OBSERVATION" | "INCIDENT" | "DOCUMENT_REVISION";
  entity_id: string;
  original_filename: string;
  declared_mime: string;
  verified_mime?: string;
  size_bytes: number;
  checksum_sha256: string;
  state: string;
  upload_url?: string;
  upload_headers?: Record<string, string>;
  download_url?: string;
};
```

- [ ] **Step 2: Update `web/src/api.ts`**

```ts
createIncident: (value: {
  site_id: string;
  asset_id: string;
  summary: string;
  details?: string;
  priority: string;
  status?: string;
  assignee_id?: string;
  reporter_id?: string;
  team_id?: string;
  occurred_at?: string;
}) =>
  command<Incident>("/incidents", {
    ...value,
    source_of_truth: "OWNED_BY_SKAWLD",
    detected_at: new Date().toISOString()
  }),
teams: () => request<{ items: Team[] }>("/teams"),
people: () => request<{ items: Person[] }>("/people"),
listAttachments: (entityKind: string, entityId: string) =>
  request<{ items: Attachment[] }>(`/attachments?entity_kind=${encodeURIComponent(entityKind)}&entity_id=${encodeURIComponent(entityId)}`),
```

Add a generic upload helper (extract from `uploadDocumentRevision`):

```ts
uploadIncidentAttachment: async (incidentId: string, siteId: string, file: File) => {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
  const checksum = Array.from(new Uint8Array(digest))
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
  const manifest = await command<Attachment>("/attachments", {
    site_id: siteId,
    entity_kind: "INCIDENT",
    entity_id: incidentId,
    client_event_id: crypto.randomUUID(),
    original_filename: file.name,
    declared_mime: file.type || "video/mp4",
    size_bytes: file.size,
    checksum_sha256: checksum
  });
  const upload = await fetch(manifest.upload_url!, {
    method: "PUT",
    headers: manifest.upload_headers ?? {},
    body: file
  });
  if (!upload.ok) throw new ApiError({ status: upload.status, title: "Upload failed", detail: `Upload failed with status ${upload.status}` });
  return command<Attachment>(`/attachments/${manifest.id}/complete`, {});
},
closeIncident: (incidentID: string, expectedVersion: number) =>
  command<Incident>(`/incidents/${incidentID}/close`, { expected_version: expectedVersion }),
reopenIncident: (incidentID: string, expectedVersion: number) =>
  command<Incident>(`/incidents/${incidentID}/reopen`, { expected_version: expectedVersion }),
```

- [ ] **Step 2b: Update `ListOptions`**

`web/src/api.ts` `ListOptions` already sends `state`; the queue filters client-side, so no change is strictly required. Leave as-is.

- [ ] **Step 3: Run typecheck**

Run: `cd web && npm run lint`
Expected: FAIL — the console pages still reference `incident.severity`/`incident.state`. Fix in the next tasks; or to keep the build green now, update `labels.ts` (Task 10) and page consumers (Tasks 11-12) in this same PR sequence — Task 9 is committed only when `npm run lint` passes.

- [ ] **Step 4: Commit**

```bash
git add web/src/types.ts web/src/api.ts
git commit -m "feat: add incident report types and API client methods"
```

---

### Task 10: Web labels and i18n

**Files:**
- Modify: `web/src/console/labels.ts`
- Modify: `web/src/i18n/messages.ts`

**Interfaces:**
- Produces: `priorityLabelKey`/`priorityTone` (replacing `severityLabelKey`/`severityTone`), `incidentStatusLabelKey`/`incidentStatusTone` (replacing `incidentStateLabelKey`/`incidentStateTone`, extended with closed/reopened); new message keys for status, priority, details, assignee, reporter, team, date, media, time metrics — in both `en` and `vi` blocks.

- [ ] **Step 1: Update `labels.ts`**

Rename `severityLabelKey` → `priorityLabelKey` (message keys `incident.priority.low|medium|high|critical|unknown`), `severityTone` → `priorityTone` (same values). Rename `incidentStateLabelKey` → `incidentStatusLabelKey` and add `case "CLOSED": return "incident.status.closed"; case "REOPENED": return "incident.status.reopened";` (keys `incident.status.open|inProgress|resolved|closed|reopened|unknown`). Same for `incidentStatusTone` with `closed` → `low`, `reopened` → `medium`.

- [ ] **Step 2: Update `messages.ts`**

In the `en` block, replace `incident.state.*` keys with `incident.status.*` (open/inProgress/resolved/closed/reopened/unknown), `incident.severity.*` with `incident.priority.*`, and add:

```ts
"incident.priority": "Priority",
"incident.status": "Status",
"incident.details": "Details",
"incident.assignee": "Assignee",
"incident.reporter": "Reporter",
"incident.team": "Team",
"incident.date": "Date",
"incident.media": "Video or images",
"incident.media.hint": "Attach photos or video clips (max 50 MB each).",
"incident.timeToComplete": "Time to complete",
"incident.timeToCreate": "Time to create",
"form.status": "Status",
"form.priority": "Priority",
"form.details": "Details",
"form.assignee": "Assignee",
"form.reporter": "Reporter",
"form.team": "Team",
"form.date": "Date",
"form.media": "Video or images",
"form.detailsPrompt": "Describe what happened: what equipment or asset is affected, what you observed, when and where it occurred, any symptoms, alarms, readings, or unusual conditions, and what actions have already been taken.",
"form.selectTeam": "Select a team",
"form.selectPerson": "Select a person",
"incident.media.empty": "No media attached.",
"incident.close": "Close incident",
"incident.close.success": "Incident closed",
"incident.reopen.label": "Reopen",
```

Mirror all renamed and new keys in the `vi` block (Vietnamese translations; use the existing Vietnamese style from the surrounding keys).

- [ ] **Step 3: Update consumers to the new names**

Update every reference to `severityLabelKey`/`severityTone`/`incidentStateLabelKey`/`incidentStateTone` and `.severity`/`.state` on incident objects across:
- `web/src/console/components/IncidentBoard.tsx`
- `web/src/console/pages/IncidentsPage.tsx`
- `web/src/console/pages/IncidentDetailPage.tsx`
- `web/src/console/pages/AssetDetailPage.tsx` (incident rows)
- `web/src/console/components/ForYouTabs.tsx`
- `web/src/console/components/Overview.tsx`

Use `.priority` / `.status` and `priorityTone` / `incidentStatusTone`.

- [ ] **Step 4: Run typecheck**

Run: `cd web && npm run lint`
Expected: PASS.

- [ ] **Step 5: Run web tests**

Run: `cd web && npm test`
Expected: PASS after updating fixtures that use `severity`/`state` in `IncidentsPage.test.tsx` and `IncidentDetailPage.test.tsx` to `priority`/`status`.

- [ ] **Step 6: Commit**

```bash
git add web/src/console/labels.ts web/src/i18n/messages.ts web/src/console
git commit -m "feat: relabel incident status and priority across the console"
```

---

### Task 11: Create incident form redesign

**Files:**
- Modify: `web/src/console/components/CreateIncidentForm.tsx`
- Modify: `web/src/console/pages/IncidentsPage.tsx`

**Interfaces:**
- Consumes: `api.teams()`, `api.people()`, `principal`, `Attachment` flow, updated `createIncident`.
- Produces: form with fields in order: status → video/images → summary → details (with prompt placeholder) → assignee → priority → date → team → reporter. Reporter defaults to `principal.display_name`; status defaults OPEN; date defaults today. `onCreate` receives `{ site_id, asset_id, summary, details, priority, status, assignee_id, reporter_id, team_id, occurred_at, files: File[] }`.

- [ ] **Step 1: Rewrite the form**

`web/src/console/components/CreateIncidentForm.tsx`:

- Props gain `people: Person[]`, `teams: Team[]`, `principal: Principal | undefined`.
- New state: `status` (default `"OPEN"`), `files: File[]`, `details`, `assigneeID`, `priority` (default `""`), `occurredAt` (default today `YYYY-MM-DD`), `teamID`, `reporterID` (default `principal?.id ?? ""`).
- Field order in JSX: status select → media file input (`accept="image/*,video/*"` `multiple`) → summary input → details textarea with `placeholder={t("form.detailsPrompt")}` → assignee select (people) → priority select → date input (`type="date"`) → team select (teams) → reporter select (people).
- Validation: site, asset, summary (≥3 chars), priority required; others optional.
- `submit` calls `props.onCreate({ site_id: asset.site_id, asset_id: asset.id, summary, details, priority, status, assignee_id, reporter_id, team_id, occurred_at: new Date(occurredAt).toISOString(), files })`.

- [ ] **Step 2: Update `IncidentsPage`**

- Fetch `teams` and `people` with `useQuery(() => api.teams().then((r) => r.items), [])` and `useQuery(() => api.people().then((r) => r.items), [])`.
- Replace the `create` command: on success, upload `value.files` sequentially via `api.uploadIncidentAttachment(incident.id, value.site_id, file)`, then navigate to the detail page:

```ts
const create = useCommand(
  async (value: CreateIncidentValue) => {
    const incident = await api.createIncident(value);
    for (const file of value.files) {
      await api.uploadIncidentAttachment(incident.id, value.site_id, file);
    }
    return incident;
  },
  {
    successMessage: t("incidents.create.success"),
    onSuccess: (incident) => navigate(`/incidents/${incident.id}`),
  },
);
```

- Pass `people`, `teams`, `principal` to `<CreateIncidentForm>`.

- [ ] **Step 3: Run typecheck**

Run: `cd web && npm run lint`
Expected: PASS.

- [ ] **Step 4: Run web tests**

Run: `cd web && npm test`
Expected: PASS (update `IncidentsPage.test.tsx` mock for `teams`/`people` and the `createIncident` assertion to include the new payload).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/components/CreateIncidentForm.tsx web/src/console/pages/IncidentsPage.tsx web/src/console/pages/IncidentsPage.test.tsx
git commit -m "feat: redesign the incident create form with ticket fields and media upload"
```

---

### Task 12: Incident queue, board, and detail page

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx`
- Modify: `web/src/console/components/IncidentBoard.tsx`
- Modify: `web/src/console/pages/IncidentDetailPage.tsx`
- Modify: `web/src/console/pages/AssetDetailPage.tsx`

**Interfaces:**
- Consumes: renamed labels/types from Tasks 9-10, `api.listAttachments`.
- Produces: queue shows priority badge, status tabs, assignee/team/date columns; board columns include CLOSED/REOPENED; detail page shows details, assignee/reporter/team/priority/status/occurred date, time metrics, media gallery, close/reopen actions.

- [ ] **Step 1: Update the queue**

`IncidentsPage.tsx`:
- `type StateTab = "OPEN" | "IN_PROGRESS" | "RESOLVED" | "CLOSED" | "ALL"` and `STATE_TABS` accordingly (drop REOPENED from tabs; reopened items appear under their current status and "ALL").
- `counts` map includes `CLOSED`.
- Table columns: rename `severity` column to `priority` (`t("incident.priority")`, `priorityTone(incident.priority)`); rename `state` column to `status`; add `assignee` column (`incident.assignee_name ?? "—"`), `team` column (`incident.team_name ?? "—"`), `date` column (`RelativeTime time={incident.occurred_at ?? incident.detected_at}`).
- Severity filter select uses `priorityLabelKey`.
- `rowClassName` uses `incident.priority.toLowerCase()`.

- [ ] **Step 2: Update the board**

`IncidentBoard.tsx`: `COLUMNS` includes `{ state: "CLOSED", labelKey: "incident.status.closed" }`; rename `SEVERITY_ICONS` usage to `incident.priority` and `priorityLabelKey`; `board-card--${incident.priority.toLowerCase()}`.

- [ ] **Step 3: Update the detail page**

`IncidentDetailPage.tsx`:
- Facts rail: replace severity/state badges with `priorityTone`/`incidentStatusTone`; add rows: Details (pre-wrap text), Assignee, Reporter, Team, Date (`occurred_at ?? detected_at`), Time to complete (`value.time_to_complete_seconds` formatted as `Xd Yh Zm`, else "—"), Time to create (relative to `detected_at`).
- Media gallery panel: fetch `api.listAttachments("INCIDENT", value.id)` via `useQuery`; render images (`<img src={a.download_url}>`) and video (`<video controls src={a.download_url}>`) for `AVAILABLE` states, plus a filename list; empty state `t("incident.media.empty")`.
- Actions: when `value.status === "RESOLVED"` show "Close incident" and "Reopen" `GatedButton`s; wire `useCommand(api.closeIncident)` and `useCommand(api.reopenIncident)` (add `closeIncident`/`reopenIncident` to `api.ts` as `command<Incident>(\`/incidents/${id}/close\`, { expected_version: incident.version })`).

- [ ] **Step 4: Update `AssetDetailPage`**

Incident rows: `incident.severity` → `incident.priority` with `priorityLabelKey`/`priorityTone`.

- [ ] **Step 5: Run typecheck and tests**

Run: `cd web && npm run lint && npm test`
Expected: PASS (update `IncidentDetailPage.test.tsx` fixture to `priority`/`status` and mock `listAttachments`, `closeIncident`, `reopenIncident`).

- [ ] **Step 6: Commit**

```bash
git add web/src/console/pages/IncidentsPage.tsx web/src/console/components/IncidentBoard.tsx web/src/console/pages/IncidentDetailPage.tsx web/src/console/pages/AssetDetailPage.tsx web/src/api.ts web/src/console/pages/IncidentDetailPage.test.tsx
git commit -m "feat: surface incident report fields in queue, board, and detail views"
```

---

### Task 13: End-to-end verification

**Files:**
- None (verification only).

- [ ] **Step 1: Go checks**

Run: `make check`
Expected: PASS (gofmt, go vet, go test).

- [ ] **Step 2: Integration tests**

Run: `TEST_DATABASE_URL="postgres://postgres:postgres_dev_only@localhost:5432/postgres" go test -race -count=1 ./internal/incident/... ./internal/team/... ./internal/attachment/...`
Expected: PASS.

- [ ] **Step 3: Web checks**

Run: `make web-check`
Expected: PASS (openapi copy, lint, vitest, vite build).

- [ ] **Step 4: Manual smoke (optional)**

Run: `make compose-up && go run ./cmd/seed` then `cd web && npm run dev`. Create an incident with all fields and a photo; verify the detail page shows the gallery, assignee/reporter/team, and time metrics; resolve, close, reopen.

- [ ] **Step 5: Update the design doc status**

In `docs/superpowers/specs/2026-08-08-incident-report-redesign-design.md`, change the header `Status: Approved (design)` to `Status: Implemented`.

- [ ] **Step 6: Commit**

```bash
git add docs/superpowers/specs/2026-08-08-incident-report-redesign-design.md
git commit -m "docs: mark incident report redesign as implemented"
```
