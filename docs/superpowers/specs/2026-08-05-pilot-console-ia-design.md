# Design: Pilot console IA — detail pages, workbench, reports, search

Date: 2026-08-05
Status: Approved for implementation

## Problem

The web console is a flat 8-view single-page app (one 1,700-line `App.tsx`
switching via `useState<View>`). The API already exposes detail endpoints for
every entity, but the console renders most of them as panels/tables. Missing:
detail pages, execution workbench, reports library, document detail, hybrid
search, and real URLs. This sub-project delivers the **pilot-critical work
flows** (cluster A) as routed pages with a coherent IA, per
`docs/contributing/web-ui-design-system.md`.

## Decisions (from brainstorming)

- **Routing:** React Router v7, path-based (`BrowserRouter`). Deep links,
  browser back/forward, shareable URLs. Server already supports SPA fallback
  (`try_files $uri /index.html` in the pilot nginx; Vite dev fallback).
- **Scope:** cluster A only (pilot work flows). Cluster B (users/roles, org &
  sites, settings) and marketing page are out of scope.
- **Shell:** existing dark instrument theme kept; sidebar regrouped into
  sections; breadcrumbs added; role-aware nav gated on `principal.permissions`.
- **Data:** keep the `api.ts` fetch client; add a small `useApi` hook with
  `{data, loading, error}` + refetch; every page ships skeleton, empty, and
  error states.
- **Dependencies added (minimal):** `react-router-dom` (v7),
  `@radix-ui/react-dialog`, `@radix-ui/react-tabs`, `@phosphor-icons/react`
  (already installed). No styling framework change; new component classes
  extend the existing token-driven `styles.css`.
- **i18n:** all new copy gets en + vi keys in `messages.ts`.

## Information architecture

### Route map

```
/landing                  public marketing page (existing, untouched)
/                         Dashboard (role-aware landing)
/assets                   Assets list
/assets/:assetId          Asset detail
/incidents                Incident queue
/incidents/:incidentId    Incident detail
/executions/:executionId  Execution workbench  (anchor page)
/reports                  Reports library
/reports/:reportId        Report detail
/handovers                Shift handover
/knowledge                Knowledge documents
/knowledge/:documentId    Document detail
/search?q=                Hybrid search
/quality                  AI quality & safety (existing view, kept)
/demonstrations           Demonstrations (existing view, kept)
/workflows                Learned workflows (existing view, kept)
```

Existing views (quality, demonstrations, workflows) are preserved as pages
inside the new shell; they are not redesigned in this sub-project.

### Shell

- Sidebar (240px) regrouped into sections:
  - Operations: Dashboard, Incidents, Handover
  - Knowledge: Knowledge, Search
  - Records: Reports
  - Learning: Demonstrations, Workflows
  - Quality: AI quality & safety
  - Executions are reached from incident detail and dashboard shortcuts
    (no standalone executions list in this sub-project).
- Topbar: breadcrumbs (`Incidents / IN-1042 / Execution EX-77`), operator
  presence, language switch, sign out (existing).
- Role gating (by permission, not role name):
  - Reports visible to principals with `report:write` or `report:approve`.
  - Execution create/execute actions gated by `execution:write`;
    recommendation actions by `recommendation:run`; report transitions by
    `report:write` / `report:approve`; knowledge transitions by
    `knowledge:write` / `knowledge:approve`; incident resolve by
    `incident:resolve`.
- Auth gate: console routes render after `/api/v1/me` succeeds; on 401
  redirect to `/auth/login` (existing OIDC begin). Deep links preserve the
  intended URL across the login round trip.

## Pages

### Dashboard (`/`)

Role-aware landing:
- Supervisor: open-incident queue, pending report approvals, site scope.
- Technician: assigned executions, measurements due, latest knowledge.
- Manager: pending handovers, recommendation review.
- Administrator: org health, quality summary, assets registered.
Each card is a shortcut linking into the relevant page, not just a number.

### Incident detail (`/incidents/:incidentId`)

Summary, severity badge, linked asset (link to `/assets/:id`), executions
list (each links to its workbench), actions gated by permission:
create execution (`execution:write`), generate recommendation
(`recommendation:run`), resolve (`incident:resolve`).

### Execution workbench (`/executions/:executionId`) — anchor

- Left rail: step list with LOTO / safety gates and state
  (PENDING / COMPLETED / BLOCKED).
- Right: measurements form (`MeasurementForm`), observations/actions,
  evidence links (`EvidenceLinks`), draft-report button that navigates to
  the report page.
- State transitions ASSIGNED -> IN_PROGRESS -> COMPLETED gated by role.

### Reports library + detail (`/reports`, `/reports/:reportId`)

- Library: list with state badges (DRAFT / SUBMITTED / APPROVED).
- Detail: structured content (summary, measurements, observations, actions,
  outcome, unknowns), evidence trail, submit/approve actions gated by
  `report:write` / `report:approve`.

### Knowledge document detail (`/knowledge/:documentId`)

Revisions with approval/ingestion state; approve / retire / request
ingestion gated by `knowledge:write` / `knowledge:approve`; applicability
block.

### Hybrid search (`/search?q=`)

Backed by `POST /search`. Results grouped by type (incidents / documents /
workflows) with evidence score and authority shown. Search box in the
sidebar and dashboard.

### Handover (`/handovers`)

Existing panel promoted to a page; submit / accept / acknowledge
transitions gated by `handover:write` / `handover:accept`.

## Data & components

- `useApi` hook: fetch wrapper returning `{data, loading, error, refetch}`.
- Component classes added to `styles.css` (token-driven): `Skeleton`,
  `Toast` (contextual errors), `Breadcrumb`, badge variants, dialog/tabs
  styling for Radix primitives.
- Empty states are invitations to act (create buttons where permitted);
  errors are specific, in the interface voice, never apologetic.
- Loading states are skeletons matching final layout shapes.

## Testing & verification

- Unit: route smoke tests with `MemoryRouter` (render each page shell,
  assert key elements); existing tests stay green.
- Playwright: deep-link navigation (`/executions/EX-77`), breadcrumb
  rendering, role-gated nav (supervisor vs technician vs manager), desktop
  (1440) and mobile (390) overflow checks, both color schemes.
- `tsc -b`, `vitest run`, `vite build` all green before sign-off.

## Out of scope (future sub-projects)

- Cluster B: user/role management UI (post federated login), org & sites,
  settings, attachments/transcriptions pages.
- Marketing page changes.
- Redesign of demonstrations / workflows / quality pages (kept as-is).
