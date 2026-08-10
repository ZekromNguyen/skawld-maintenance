# Custom Fields Core (Giai đoạn 1 — Multi-tenant configurability)

Status: Proposed | Date: 2026-08-09

## Problem

Skawld targets a shared SaaS for many enterprises, but every enterprise has a
different business process, different incident report, and different required
fields. Today all entity fields are hard-coded:

- `incident` fields live in `internal/incident/domain/incident.go` with fixed
  status/priority enums, validated in `New()` and enforced by PostgreSQL
  CHECK constraints.
- Asset, execution, and report fields are likewise fixed Go structs.
- Tenant isolation exists only as `OrganizationID`/`SiteID` columns scoped in
  handlers; there is no per-tenant configuration layer.

Changing business process for a customer means changing code and shipping a new
version. This does not scale to many tenants.

## Goal (Giai đoạn 1)

Deliver the **custom fields core**: tenant administrators create, edit, retire,
and arrange custom fields on incidents through an admin UI. Values are stored,
validated, searchable, and surfaced in the create form, detail view, queue
list, and saved-view filters. Everything is per-organization (Jira project
settings model).

This is stage 1 of the full Jira model (custom fields → issue types →
workflow → screens). Only the **incident** entity is supported in this stage;
the mechanism is entity-agnostic (`entity_type` column) so asset/execution can
follow without redesign.

## Approach

Config is data, not code. A per-tenant `field_definition` table describes the
shape of custom fields; values live in a JSONB column on incidents (single
`custom_values` map) plus a typed history table for audit. Field definitions
follow the existing immutable-version philosophy of Phase 4: a field that has
ever held data cannot be mutated or deleted, only retired.

## Data model

### New migration `00015_custom_fields.sql`

```sql
CREATE TABLE field_definitions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    entity_type   text NOT NULL,              -- 'incident' (extensible)
    key           text NOT NULL,              -- machine name, e.g. 'po_number'
    label         text NOT NULL,              -- display label (i18n base)
    description   text NOT NULL DEFAULT '',
    field_type    text NOT NULL,              -- TEXT|NUMBER|DATE|SELECT|MULTI_SELECT
    config        jsonb NOT NULL DEFAULT '{}',-- required, options[], min, max, regex, default
    status        text NOT NULL DEFAULT 'ACTIVE', -- ACTIVE|RETIRED
    sort_order    integer NOT NULL DEFAULT 0,
    version       integer NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    retired_at    timestamptz,
    UNIQUE (organization_id, entity_type, key)
);

-- value on the incident row: map field_definition_id -> JSONB value
ALTER TABLE incidents ADD COLUMN custom_values jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE custom_value_history (
    id            bigserial PRIMARY KEY,
    incident_id   uuid NOT NULL REFERENCES incidents(id),
    field_definition_id uuid NOT NULL REFERENCES field_definitions(id),
    principal_id  uuid NOT NULL REFERENCES principals(id),
    value_before  jsonb,
    value_after   jsonb NOT NULL,
    changed_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX custom_value_history_incident_idx
    ON custom_value_history (incident_id, changed_at DESC);
CREATE INDEX field_definitions_org_idx
    ON field_definitions (organization_id, entity_type, sort_order);
CREATE INDEX incidents_custom_values_gin ON incidents USING gin (custom_values);
```

Backfill: none required. Existing orgs get no custom fields until an admin
creates them.

### Field types (stage 1)

| Field type | config keys | JSONB value |
|---|---|---|
| TEXT | `required`, `maxLength`, `regex` | string |
| NUMBER | `required`, `min`, `max`, `step` | number |
| DATE | `required` | ISO 8601 string |
| SELECT | `required`, `options[]` (label, value) | string (option value) |
| MULTI_SELECT | `required`, `options[]` | array of strings |

### Rules

- `key`: `^[a-z][a-z0-9_]{1,63}$`, immutable once created.
- **Immutable after use**: once any incident has a value for a field, only
  `label`, `description`, `sort_order` may change; `field_type`, `config`, and
  `key` are locked. Fields with data can only be retired, never deleted.
- **Retire**: retired fields are hidden from the create/edit form and admin
  list default view, keep all history and values, and remain readable.
- Validation of `value` against `field_type` + `config` happens in the
  application layer (Go) before write — never trust JSONB raw.

## API (mounted under `/api/v1`)

Permission model: existing role permissions stay untouched. A new permission
`field:manage` is added to `Permission` and granted to `RoleAdministrator`
only. Read access to custom fields rides on existing `incident:read`/`incident:create`.

```
GET    /admin/field-definitions?entity_type=incident
POST   /admin/field-definitions
GET    /admin/field-definitions/{id}
PATCH  /admin/field-definitions/{id}        -- label/description/sort_order only
POST   /admin/field-definitions/{id}/retire
GET    /admin/field-definitions/{id}/history
```

Incident payloads gain (create + detail + update):

- `custom_values: map[string]any` — keyed by field **key** in the API (the
  server resolves key → definition, validates, stores by definition id). This
  keeps API payloads human-readable.
- List responses return definitions (id, key, label, type, config) in a
  `custom_fields` section so the UI can render values without a second round
  trip.

Search/filter:

- `GET /api/v1/incidents?custom_field.<key>=value` — TEXT exact / NUMBER
  range / SELECT value filter, joined with existing saved-view filters.
- TEXT/NUMBER fields are indexed via the GIN index; filters push down to SQL
  (`custom_values->>'key'`), pagination and cursor behavior unchanged.

## UI (web console)

### Admin page `/admin/custom-fields`

- Table of fields: label, key, type badge, required, sort order, status.
- Create dialog: label, key (auto-suggested from label), type, config per type
  (options editor for SELECT/MULTI_SELECT), required, sort order. Live preview
  of the rendered form control.
- Row actions: Edit (locked fields disabled with lock icon + tooltip
  explaining immutability), Retire (confirmation dialog showing how many
  incidents hold values), History (audit drawer).
- Permission-gated: link visible only with `field:manage`; route guarded by
  `AuthorizedRoute` (existing pattern).
- Everything follows the existing design system tokens
  (`docs/contributing/web-ui-design-system.md`) and i18n via `messages.ts`.

### Incident forms and views

- `CreateIncidentForm` and edit dialog render active custom fields
  dynamically (sorted by `sort_order`, appended after the fixed fields, in a
  "Custom fields" section) with per-type controls (TextInput, NumberInput,
  DatePicker, Select, MultiSelect) validated against `config`.
- `IncidentDetailPage` renders a "Custom fields" card (label/value pairs,
  only present values).
- `IncidentsPage` queue: custom fields are not new columns; a "Fields"
  disclosure row expands to show custom field values per row; saved-view
  filters gain custom field filter chips (existing saved-views pattern).

### i18n

Field labels are stored per-tenant as plain text (not i18n keys) — the label
*is* the tenant's wording. UI chrome (section titles, admin labels) uses
existing `messages.ts` keys.

## Backend implementation notes

- New package `internal/customfield/` mirroring existing module layout:
  `domain/` (definition, value validation), `application/` (service), and a
  Postgres adapter. Alternative: extend `internal/incident` — rejected because
  the feature is entity-agnostic and must grow into issue types + workflow.
- `Incident` domain gains `CustomValues map[string]any`; `New()`/update
  validation now includes custom field validation resolved through the
  `customfield` service. The postgres store persists `custom_values` and
  writes history rows in the same transaction.
- JSONB writes use `jsonb_set`/full-row update as today's store pattern;
  optimistic locking via existing `version` column stays.
- History capture: create sets one history row per value; update diffs old vs
  new values; no change → no row.

## Error handling

- Unknown field key → `422` with the key listed.
- Invalid value (wrong type, out of range, not in options, regex fail,
  required missing) → `422` listing each field key + reason.
- Retiring a field that is already retired → `409`.
- Editing a locked property (type/config) on a field that has data → `409`
  with explanation; the UI disables those inputs so this is a defense in depth.

## Testing

- Domain: value validation table tests per field type + config edge cases
  (boundaries, regex, required, unknown option).
- Adapter: integration test against the migration — CRUD, immutability after
  use, retire keeps history, GIN search filter correctness, history rows on
  create/update.
- API: handler tests for permission (`field:manage` vs other roles, fail
  closed), 422/409 cases, incident list custom-field filter.
- Web: component tests for the dynamic form renderer (per type), admin table
  (locked-state display, retire flow), and filter chips; existing
  `CreateIncidentForm` tests updated for the new section.

## Out of scope (later stages)

- Issue types (stage 2), workflow/status engine (stage 3), screens/report
  templates (stage 4), full admin console polish (stage 5).
- Custom fields on asset/execution/report entities (mechanism supports them;
  wiring is a follow-up).
- Tenant branding/theming, per-tenant status enums.
