# Design: Role × screen matrix for the full console

Date: 2026-08-06
Status: Approved for documentation (brainstorm follow-on to console redesign)

## Problem

The console redesign spec (`2026-08-06-console-redesign-design.md`) rebuilds the
console surface but does not enumerate which screens and actions each of the
five roles needs. Role gating is done by permission, not role name, and the
permission model already exists (`internal/identity/domain/authorization.go`).
This spec pins down the complete screen inventory for the full console (pilot
cluster A + future cluster B) and the role × screen × action matrix, so page
designers and testers have one authoritative reference.

## Source of truth

- Role → permission mapping: `PermissionsForRole` in
  `internal/identity/domain/authorization.go:93-184`. Unknown roles fail closed.
- API surface: `internal/platform/httpserver/router.go:90-107`.
- Console IA and page redesigns:
  `docs/superpowers/specs/2026-08-05-pilot-console-ia-design.md` and
  `docs/superpowers/specs/2026-08-06-console-redesign-design.md`.

## Roles

| Role | Key identity |
|---|---|
| Administrator | Everything, incl. org creation, workflow publish, users & roles, settings, integrations |
| Maintenance Supervisor | Everything except `organization:create` and `workflow:publish`; the operational approver |
| Senior Technician | The capture + verify role: prerequisites, demonstrations, workflow review; can create incidents |
| Technician | Execution-focused; the only role that cannot create incidents |
| Manager | Read-everything oversight: recommendation review, handover accept, demonstration review |

## Screen inventory (24 screens)

### Cluster A — operations & knowledge (exists / planned)

| # | Screen | Route |
|---|---|---|
| 1 | Dashboard (role-aware landing) | `/` |
| 2 | Incident queue | `/incidents` |
| 3 | Incident detail (+ timeline/notes/assignment) | `/incidents/:id` |
| 4 | Executions list | `/executions` |
| 5 | Execution workbench (steps, LOTO, measurements, evidence) | `/executions/:id` |
| 6 | Assets list | `/assets` |
| 7 | Asset detail (+ linked history, applicability) | `/assets/:id` |
| 8 | Reports library | `/reports` |
| 9 | Report detail (+ revisions) | `/reports/:id` |
| 10 | Knowledge list | `/knowledge` |
| 11 | Document detail (revisions, approval state) | `/knowledge/:id` |
| 12 | Hybrid search | `/search?q=` |
| 13 | Shift handover | `/handovers` |
| 14 | Demonstrations (+ capture flow) | `/demonstrations` |
| 15 | Workflows (+ applicability review, compile) | `/workflows` |
| 16 | AI quality & safety | `/quality` |

### Cluster B — governance & admin (future work)

| # | Screen | Purpose |
|---|---|---|
| 17 | Users & roles | Directory sync, role grants, approval-authority grants (ADR-0003) |
| 18 | Organizations & sites | Org creation, site registry, principal↔site scope |
| 19 | Recommendations review queue | `recommendation:review` backlog (currently buried in incident detail) |
| 20 | Settings | System config, SDK version gate (ADR-0002) |
| 21 | Integrations | EAM/CMMS import, transcription, source-of-truth mapping (ADR-0004) |
| 22 | Attachments & transcriptions | Evidence/media library (routes exist, no UI) |
| 23 | Audit trail | Governance history: approvals, imports, publishing |

## Role × screen matrix

Read access to all cluster A screens is universal (every role holds the six
read permissions: asset, incident, execution, knowledge, demonstration,
workflow). Differences are actions plus cluster B visibility.

| Screen | Admin | Supervisor | Sr. Tech | Technician | Manager |
|---|---|---|---|---|---|
| Dashboard focus | Org health, quality | Open incidents, approvals | My executions, measurements due | Assigned executions | Pending handovers, rec review |
| Incidents | create + resolve | create + resolve | create (no resolve) | view only | view only |
| Executions | all + write | all + write | write + verify prereq | write | view only |
| Asset criticality | approve | approve | view | view | view |
| Reports | write + approve | write + approve | write (draft) | write (draft) | view |
| Knowledge | write + approve | write + approve | review only | view | view |
| Handover | write + accept | write + accept | write | write | write + accept |
| Demonstrations | capture + review | capture + review | capture + review | capture | review only |
| Workflows | review + publish | review (no publish) | review | view | view |
| Recommendations | run + review | run + review | run only | run | review only |
| Quality | full | full | view | view | view |
| Users & roles | manage | view | — | — | — |
| Orgs & sites | manage | view | — | — | — |
| Recommendations queue | review | review | — | — | review |
| Settings / Integrations / Audit | manage | import run, audit view | — | — | — |

## Action-level gating rules

- **Nav gating:** sidebar items hide when the principal lacks every write
  permission for that area (e.g. Users & roles only for Administrator).
- **Detail-page action gating:** actions are disabled-with-reason when the
  permission is missing; never a no-op enabled button.
- **Resolve / Retire / Approve / Publish / Accept** always go through
  `ConfirmDialog` (destructive or governance-significant).
- **Read-only view vs hidden:** cluster A screens are visible to all roles;
  cluster B screens hide for roles without any applicable permission.

## Key findings

1. **Technician is the only role that cannot create incidents** — their path
   in is via assigned executions, not queue entry.
2. **Supervisor ≠ Admin:** supervisor holds `workflow:review` but not
   `workflow:publish`, and not `organization:create`. Publishing is admin-only.
3. **Senior Technician is the capture + verify role** — the only non-supervisor
   with `execution:prerequisite:verify` + `demonstration:review`; the linchpin
   of the learning loop (Plan.md:230 "capture senior work as demonstrations").
4. **Recommendations review queue is missing** — `recommendation:review`
   (manager/supervisor/admin) has no home screen today; add as a cluster B item.
5. **Cluster B has no backend yet** — federated login means roles come from the
   IdP; only `/organizations` and `/sites/{id}` exist today. Cluster B is a
   roadmap, not pilot scope.

## Out of scope

- Implementation of cluster B screens (separate sub-projects after pilot).
- Redesign of demonstrations / workflows / quality pages (covered by console
  redesign spec, phases 3).
- Marketing page.
