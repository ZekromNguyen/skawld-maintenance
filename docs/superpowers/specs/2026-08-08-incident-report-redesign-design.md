# Incident Report Redesign

Status: Implemented | Date: 2026-08-08

## Problem

The incident form today exposes only asset, summary, severity, and state. The
maintenance team needs a ticket-style incident report with: status, video or
images, summary, details (with a specific prompt), assignee, priority, date,
select team, and reporter.

The requested fields do not exist today. `assignee`, `team`, `reporter`,
`details`, and media upload have zero implementation. `priority` exists only as
`severity`; `status` exists only as `state`.

## Approach

Extend the existing incident module end-to-end rather than creating a parallel
"incident report" entity. This avoids two overlapping queues and reuses the
existing attachments (S3), auth/principal, and org/site multi-tenancy
infrastructure.

## Data model

### `incidents` table changes (one migration, `00005_incident_report_redesign.sql`)

| Your field | Implementation |
|---|---|
| status | rename `state` → `status`; values OPEN, IN_PROGRESS, RESOLVED, CLOSED, REOPENED |
| video or images | existing `attachments` (`entity_kind=INCIDENT`); allow video MIME; add upload UI |
| summary | unchanged |
| details | new `details` text column |
| assignee | new `assignee_id` → `principals(id)`, nullable |
| priority | rename `severity` → `priority`; values LOW/MEDIUM/HIGH/CRITICAL |
| date | existing `occurred_at`, editable in ticket format |
| select team | new `teams` + `team_members` tables, org-scoped |
| reporter | new `reporter_id` → `principals(id)`, auto-filled with logged-in user |

Enums:

- `status`: OPEN, IN_PROGRESS, RESOLVED, CLOSED, REOPENED (extended from
  OPEN/IN_PROGRESS/RESOLVED).
- `priority`: LOW, MEDIUM, HIGH, CRITICAL (renamed from `severity`).

`time to create` = auto `detected_at`; `time to complete` = computed
`resolved_at - detected_at`, displayed on the detail page.

### New tables

- `teams`: id, organization_id, name, created_at. Org-scoped.
- `team_members`: team_id, principal_id, composite PK. Org-scoped via team.

Seeded defaults per organization (e.g. Facilities, Electrical, HVAC) at
migration time for existing orgs and at org creation for new ones.

### Backfill for existing rows

- `occurred_at = detected_at` where null.
- `reporter_id = created_by` where created_by present.

Existing rows carry their current severity→priority and state→status values.

## API

- `GET /api/v1/teams` — org's teams for the dropdown.
- `GET /api/v1/people` — org members (id, display name) for assignee/reporter
  dropdowns.
- Incident create/detail payloads gain: `details`, `assignee_id`, `reporter_id`,
  `team_id`, `priority` (replacing `severity`), `status` (replacing `state`),
  `occurred_at`, `attachments`, and computed `time_to_complete`.

## UI

- **Create form**, in user-specified order: status → video/images upload →
  summary → details (placeholder = the user's prompt verbatim) → assignee →
  priority → date → team → reporter. Defaults: status OPEN, reporter = logged-in
  user, date = today.
- **Detail page**: all fields, attachments gallery, time-to-create /
  time-to-complete.
- **List/board**: priority badge, status tabs, assignee, team, date.

Details placeholder prompt:

> Describe what happened: what equipment or asset is affected, what you
> observed, when and where it occurred, any symptoms, alarms, readings, or
> unusual conditions, and what actions have already been taken.

## Error handling

- Server-side validation mirrors existing `New` validation: required summary,
  valid enum values, asset/org/site scoping from the principal, permission
  gates (existing `PermissionIncidentCreate`).
- Attachment upload errors follow the existing attachment module's
  PENDING_UPLOAD → UPLOADED → AVAILABLE lifecycle and error paths.

## Testing

- Go unit tests: new field validation and persistence, teams/people queries,
  attachment video MIME allowlist.
- Existing frontend test patterns extended to the new form and detail rendering.
