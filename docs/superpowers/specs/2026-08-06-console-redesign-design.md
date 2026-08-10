# Skawld Console Redesign — Design Plan

Date: 2026-08-06
Status: Proposal (pre-implementation)

## 1. Design read

Skawld is an **industrial maintenance intelligence layer**. The console is the
operator cockpit: supervisors, senior techs, technicians, managers, admins.
Per the design system doc, it must read as a **calibrated instrument**:
dense, precise, trustworthy, mono numerals, 1px hairlines, dark-first
(console stays dark-only; marketing handles `prefers-color-scheme`). Speed and
information density are the surface's dials (`VISUAL_DENSITY = 8`).

The current console is functionally complete but has systemic problems: a
broken data layer that silently swallows every mutation failure, no
site-awareness despite multi-site principals, dead routes/nav (breadcrumbs
never wired, `/executions` missing, dashboard shortcuts mislabeled), raw enum
text and JSON dumps rendered to operators, `window.prompt` for governance
decisions, four different loading/error/empty patterns, and no confirmations
anywhere. This plan rebuilds the console on a shared foundation, then
redesigns every page against it.

### Signature

The console's signature is the **instrument-state language**: a single tone
system (severity/state badges + severity bars + mono numerals on hairline
grids) applied identically everywhere, so an operator reads status at a
glance from any surface. No decorative status dots; every dot carries state.

## 2. Current-state findings (verified, file:line)

### 2.1 Data layer (worst systemic defect)

| Finding | Location |
|---|---|
| Every mutation swallows errors: `void mutate(...)`, no `catch` anywhere | HandoverPage:41,47,53; DemonstrationsPage:47-66; WorkflowsPage:51-102; IncidentDetailPage:84,89; ExecutionWorkbenchPage:47-55,96; AssetsPage:21-30; KnowledgePage:23-31; ReportDetailPage:33-41 |
| `useApi` refetches only on `tick`; no site/principal deps, no AbortController, no request sequencing (stale response can overwrite newer), `refetch()` returns void so `await refetch()` lies | useApi.ts:16-38; QualityPage:21 |
| Site-scope race: fetches run with `siteID === undefined` at mount and never re-run when principal resolves | DemonstrationsPage:16; WorkflowsPage:18; api.demonstrations(undefined) fetches all sites |
| `site_ids[0]` hard-pinned everywhere; no site switcher; `Topbar` advertises "{count} site scope" | DashboardPage/AssetsPage/SearchPage/HandoverPage; messages.ts:98 |
| `usePrincipal()` called per page + layout (2+ `/me` per navigation) | ConsoleLayout:12 + every page |
| 401 does a hard `window.location.assign`; non-JSON errors crash the error path with `SyntaxError` | api.ts:39-44,46 |
| Dead endpoints never wired: `recordEvidenceView`, `expandWorkflowApplicability`, `editReport` | api.ts:242,308,343 |

### 2.2 IA / navigation / routing

| Finding | Location |
|---|---|
| Breadcrumbs dead: `ConsoleLayout` accepts `trail` but it is never passed | ConsoleLayout:11-17; App.tsx:56 |
| Missing `/executions` route; "Assigned executions" dashboard card points to `/incidents` | DashboardPage:36; App.tsx:62 |
| Dead code: `PlaceholderPage` import, `View` type, `viewTitle`, duplicate `signOut`, ~15 unused imports | App.tsx:1-25,79-102; Sidebar:103-110 |
| No skip-to-content, no command palette, no global search, sidebar nav mislabeled `aria-label` = "Operations overview" | Sidebar:60 |
| Breadcrumb shape inconsistent: `Incidents / IN-1042` vs `Incidents / P-302` (asset, not incident, not a link) | IncidentDetailPage; ExecutionWorkbenchPage:60-65 |

### 2.3 Cross-page consistency

| Finding | Location |
|---|---|
| Incident state: `IN_PROGRESS` raw vs `IN PROGRESS` | IncidentDetailPage:74,114 vs IncidentQueue:35, IncidentTable:19 |
| Severity class: `toLowerCase()` vs `severityTone()` | IncidentDetailPage:71 vs IncidentQueue:31 |
| Timestamps: `relativeTime` vs raw ISO vs `toLocaleString` (browser locale) | lists vs ExecutionWorkbenchPage:105 vs DemonstrationPanel:172 |
| Error patterns: inline toast + misleading empty content vs bare early-return with no chrome | Dashboard/Incidents vs Detail/Workbench pages |
| `source_of_truth` localized in list, raw enum on detail | AssetsTable:55-58 vs AssetDetailPage:52 |
| Dead badge CSS: `.state-badge.draft/.submitted/.approved` don't exist, DRAFT == APPROVED visually | ReportsPage:48; styles.css:97 |
| Raw `JSON.stringify(observation)` rendered to operators; measurement `8.1MM_PER_S` no separator | ExecutionWorkbenchPage:118-125,103 |
| Sample data in production forms (can create demo duplicates with one Enter) | CreateIncidentForm:21-22; CreateAssetForm:19-21; MeasurementForm:11; KnowledgePanel:15-18 |
| `window.prompt()` for governance decisions (outcome, review reason, JSON path, redaction reason) | DemonstrationPanel:29,40,51,56; WorkflowLearningPanel:40 |
| Hardcoded metric `0`, hardcoded risk tone `"high"`, hardcoded severity options, hardcoded workflow-key names | Overview:20; StepRail:63-65; CreateIncidentForm:39; WorkflowsPage:61-64 |
| Broken modals: CreateAssetForm rendered **outside** the backdrop (unusable); incident modal lacks dialog semantics, ESC, focus trap | AssetsPage:38-48; IncidentsPage:46-53 |
| Execution can never reach COMPLETED (no Complete action); `draftReport` shown unconditionally; read-only users see enabled buttons that no-op | ExecutionWorkbenchPage:77-84,46 |
| Docs unreachable: Knowledge list items not links despite `/knowledge/:id` route; search results not clickable | KnowledgePanel:45-57; SearchPage:78-94 |
| "Add procedure revision" actually creates a new document each time (revision hard-coded R1) | KnowledgePanel:24-29; api.ts:136 |
| Title repeated 3x on detail pages; flat heading trees (h2 under h2, no h3) | DocumentDetailPage:72-94; HandoverPanel:16,44 |
| Retire offered on the only active approved revision with hard-coded reason `"superseded"` | DocumentDetailPage:59-63 |
| Type fields never rendered: `requires_human_review`, `unknowns`, `Execution.actions`, `recordedEvidence`, `reviews`, `evaluations`, `generated_at`, `redactions`, timestamps | multiple (catalogued in review) |
| Font is **Inter** — banned by the design system (Geist is the brand face) | styles.css:6 |

### 2.4 Missing entirely

No filters/sort/search/pagination on any list; no bulk actions; no
confirmations for destructive/safety actions (resolve, LOTO step complete,
retire); no undo; no toast/success feedback; no incident timeline/notes/
assignment/reopen; no document content preview; no evidence view tracking;
no site switcher; no global search; no command palette; no keyboard
shortcuts; no error boundaries; no `/executions` list; no report revision
history; no quality trends/export; inconsistent `EmptyRow` vs empty states;
no per-revision busy state; no in-flight indicators.

## 3. Design system foundation (console surface)

- **Type:** swap Inter → **Geist** (self-hosted, already a dep) at
  `styles.css:6`. Scale: h1 20-22px/700/-0.02em; h2 13-15px/650; h3 12-13px;
  base 12.5-13px; mono numerals for all data (counts, values, tags).
- **Shape lock (console variant):** cards 12px, inputs 10px, buttons 8px,
  dialogs 16px. One system per surface; pill buttons are the marketing CTA
  language and are not used inside the console. Documented deviation from the
  marketing surface's pill buttons.
- **Color:** keep all tokens; add tone-map utilities `.tone-critical/high/
  medium/low/info/success` derived from existing severity palette; fix dead
  `.state-badge.*` classes; status dots only for real state with tone.
- **Focus:** global `:focus-visible` ring (2px brand, offset 2px) on every
  interactive element; existing input outlines formalized.
- **Dark-first only** for the console (per design system doc).
- **Grid rhythm:** content max-width 1440px; consistent 16px gutter;
  hairline `--line` separators; no shadows except elevation on dialogs/toasts.

## 4. Foundation: shared primitives (Phase 0)

New shared layer under `web/src/console/` (or `web/src/ui/`):

1. **Data layer**
   - `useQuery(fetcher, deps)` — replaces `useApi`: keys on `deps`, request
     sequencing (drop stale), AbortController, `{data, loading, error, refetch}`,
     `refetch(): Promise<T>` that resolves when the fetch settles.
   - `useMutation(action)` — `{pending, error, run}`; auto-wires error to
     toast; optional `onSuccess`.
   - `SiteContext` — `activeSiteId` (default `site_ids[0]`), `siteSwitcher`
     UI; all `useQuery` calls depend on it (fixes the mount race).
   - `PrincipalProvider` — fetch `/me` once, memoized; `usePrincipal` consumes.
   - `api.ts` hardening: parse JSON defensively (non-JSON → friendly error),
     preserve `ApiError.status`, 401 keeps redirect but surfaces notice.
2. **Feedback**
   - `ToastProvider` — success/error/info, `aria-live="polite"`, auto-dismiss
     (errors persist + Retry action), stacked.
   - `ConfirmDialog` — destructive confirmations with optional typed reason
     field (used for Resolve, Retire, safety-step complete, redaction,
     reject/recompile).
   - `Dialog` (Radix) — replaces all hand-rolled modals: focus trap, ESC,
     `aria-labelledby`, scroll-lock. Radix is already a dependency.
3. **Layout**
   - `PageHeader` — breadcrumbs (route-driven, single source in a route
     config map) + h1 + actions slot. Kills per-page `Topbar` duplication and
     the dead `trail` prop.
   - `Breadcrumbs` — generated from route config; every detail page gets
     `Area / Entity / Item` automatically.
   - `Sidebar` — icon + label per item, real `aria-label` ("Primary"), active
     via `aria-current="page"`, section labels; adds **Executions** item.
   - `Topbar` — global: site switcher (when >1 site), global search trigger,
     principal menu (name, site scope, sign out), in future notifications.
   - `AppShell` — grid layout, skip-to-content link (console is missing one).
4. **Primitives**
   - `DataTable` — config-driven columns (sortable, optional server
     pagination), loading skeleton rows, `EmptyState`, error banner.
   - `StatusBadge` / `StateBadge` — tone-mapped + localized label map
     (one source of truth for enum → label → tone).
   - `EmptyState` (icon, title, body, optional CTA) and `ErrorState`
     (message, retry, back) — replace `.empty` and bare `toast-error`.
   - `Skeleton` primitives (line/block/table) with `aria-busy`.
   - `FormField` — label (htmlFor/id), input, hint, inline error, required.
   - `RelativeTime` — locale-aware (uses app locale, not browser).
   - `MetricCard` — real count + tone + optional link.
   - `Timeline` — vertical event rail for incident/execution/handover history.
   - `ProgressBar` — stepsProgress display.
   - `EvidencePanel` — empty state included, wires `recordEvidenceView`.
   - `CopyButton`, `SearchInput` (debounced, URL-synced), `Pager`.

## 5. Page-by-page redesign

### 5.1 Dashboard
**Wrong today:** fake `0` metric; "Active incidents" table shows RESOLVED
rows; non-clickable rows; "Assigned executions" card → `/incidents`; "Tap to
open" filler; `assets.error` ignored; error + empty conflated.
**Redesign:**
- KPI row (all computed): Open incidents, Executions in progress, Critical
  assets, Pending handovers. No fake metrics.
- **My queue** panel: incidents assigned to me / in progress, clickable rows
  → detail. (Assignment is backend-dependent; fallback = IN_PROGRESS state.)
- **Active incidents** table: state tabs (Open/All), severity, asset, age
  (`RelativeTime`), clickable rows.
- **Executions in progress** panel → `/executions` (fixes the dead shortcut).
- Shortcut cards become real ops widgets with count badges and correct routes.
- Error handling: full `ErrorState` with Retry; empty → `EmptyState`.

### 5.2 Incidents (queue + detail)
**Wrong today:** no filters/sort/pagination; dead "New" for unauthorized;
broken modal semantics; silent create/resolve failures; raw state text;
severity class divergence; no confirm on resolve; create doesn't navigate;
no timeline/assignment/reopen.
**Redesign:**
- **Queue page:** toolbar — search (debounced), state tabs with counts
  (Open / In progress / Resolved / All), severity filter, site (from
  context); sortable columns; pagination (cursor, backend-dependent; fallback
  client-side). Row = severity bar + number + summary + asset + age + badge.
- **New incident:** proper `Dialog`, `FormField` validation, i18n severity,
  no demo defaults; success → navigate to detail; inline error + toast.
- **Detail page:** left facts panel (asset, severity, state, detected,
  timeline), right executions panel (create → navigates to workbench,
  rows link to workbench, show date + state). Resolve/Reopen gated with
  `ConfirmDialog` (typed reason), disabled-with-reason tooltips for missing
  permissions, success toasts, `expected_version` on resolve.
- **Incident timeline** (new, API-dependent): state transitions + notes.

### 5.3 Executions (new page) + Workbench
**Wrong today:** no `/executions` list; workbench can never reach COMPLETED;
`draftReport` unconditional; read-only dead clicks; one click completes LOTO
steps; `JSON.stringify` observations; `8.1MM_PER_S`; breadcrumb dead end;
version conflicts silent; `actions`/evidence domain never rendered.
**Redesign:**
- **New `/executions` page:** table across all executions — purpose,
  incident link, asset, state, progress %, started; state filter; rows →
  workbench. Sidebar item under Operations. Fixes the "Assigned executions"
  promise.
- **Workbench:** h1 = purpose once; breadcrumb `Incidents / IN-1042 /
  Execution`; state + `ProgressBar`; actions — Start (ASSIGNED), **Complete**
  (IN_PROGRESS, wired to the existing state machine), Draft report → link to
  report; gated with tooltips.
- **Step rail:** completed steps show check + undo (non-gated steps);
  safety-significant steps (`SAFETY_SIGNIFICANT` risk) require `ConfirmDialog`;
  prerequisite state rendered; blocked reason outside the disabled control
  (keyboard reachable); risk tone mapped (not all "high").
- **Measurements:** required + validation, unit suffix (not readOnly input),
  reset after submit, `RelativeTime`, `aria-live` list, failure toast.
- **Observations:** structured list (no JSON dump) + add-observation input
  (API-dependent); fallback: read-only structured rendering.
- **Evidence/actions panel:** renders `Execution.actions` + recorded evidence
  via `EvidencePanel` with empty state.
- **Version conflict (409):** dialog "Changed by another operator" → reload
  + re-apply.

### 5.4 Assets
**Wrong today:** broken modal (form outside backdrop); site pinned;
silent create; demo defaults; all statuses green; detail page lies about
"linked history"; approve fire-and-forget; raw `source_of_truth`.
**Redesign:**
- **List:** fixed `Dialog` + site selector + no demo defaults + validation +
  permission-gated action (hidden for non-creators); search/filter (tag,
  class, status) + sort; status tone map; parent/hierarchy column; clickable
  rows.
- **Detail:** **real linked history** — incidents (filtered by asset),
  executions, and applicable workflows (wire `api.applicableWorkflows` — the
  endpoint exists and is unused); approve criticality with busy state,
  success toast, refetch, only when a rating exists; localized
  `source_of_truth`; status badge tone.

### 5.5 Reports
**Wrong today:** inline table (inconsistent), dead badge CSS, error = empty
row, no dates/links/metadata, `requires_human_review` invisible, no evidence
empty state, no execution link, no edit path, no permission explanation.
**Redesign:**
- **List:** `DataTable` — work order, execution link, summary, state badge
  (tone classes added), author/provider, date (API-dependent; fallback omit
  with comment), state filter; count from list.
- **Detail:** safety banner when `requires_human_review`; unknowns list;
  evidence with `EmptyState`; execution_id → workbench link; metadata block
  (provider, model, prompt_version); submit/approve via `ConfirmDialog` with
  disabled-with-reason when permission missing; wire `editReport` (Phase 2);
  revision history (API-dependent).

### 5.6 Documents (Knowledge)
**Wrong today:** "Add revision" creates a new doc; list not links; raw status
text; duplicate low-quality search; unlabeled input; no content preview;
broken retire guard; global busy; ingestion_error rendered as fixed toast.
**Redesign:**
- **List:** document cards are links → detail; `StateBadge` for
  approval_status/ingestion_state; single search (removed duplicate, or both
  funnel to `/search`); upload with progress + inline error (digest
  mismatch) + `FormField` + required; proper "Add revision" (append to
  existing document, API-dependent; otherwise relabel to "New document").
- **Detail:** one h1 (title only); per-revision busy; retire only on
  APPROVED/SUPERSEDED revisions with `ConfirmDialog` + reason; approve only
  DRAFT/REVIEW_REQUIRED with confirm; wire REVIEW_REQUIRED path; content
  preview of approved revision (read-only, from evidence content); ingest
  error inline, not toast; human-readable applicability; version +
  source_reference shown; heading hierarchy h1→h2→h3.

### 5.7 Search
**Wrong today:** results not clickable; stale results during reload; query
not URL-synced; hard limit 8; raw rrf_score jargon; no empty state; no term
highlighting.
**Redesign:** URL-synced `?q=`; debounced; result kinds map to routes
(knowledge → `/knowledge/:id`, incidents → `/incidents/:id`, evidence →
parent doc); term `<mark>` highlighting; clear stale rows while loading
(shimmer) or "showing previous results"; `EmptyState` per query; load-more /
pagination (parameterize limit); score hidden behind tooltip or removed;
site filter.

### 5.8 Handover
**Wrong today:** `items[0]` assumed latest; no history; not site-scoped;
"Shift handover draft" title lies for SUBMITTED/ACCEPTED; unknowns +
`requires_human_review` invisible; no timestamps; silent transitions; ungated
prepare/capture; no success feedback.
**Redesign:** sort by `shift_start` desc; **history list** of prior
handovers; site-scoped fetch; state-accurate panel title; safety banner for
`requires_human_review` + unknowns; shift window + timestamps; transitions
via `ConfirmDialog`? (no — non-destructive: direct with success toast +
disabled-with-reason); prepare/capture permission-gated; capture → confirm →
navigate; `<h3>` section headings.

### 5.9 Demonstrations
**Wrong today:** site-scope race; no way to start capture from the page;
`window.prompt` ×4; raw JSON payloads; redactions invisible; approved traces
re-reviewable; correction link not clickable; browser-locale dates.
**Redesign:** site-correct fetch; start-capture entry points (empty-state
CTA linking to Incident/Handover flows + actions there); `Dialog`-based
forms for outcome/review/redaction (no JSON path — field selector + reason);
redaction indicator on masked events + view-reason; review actions gated on
`review_status`; clickable correction link (scroll/highlight target);
app-locale dates; `aria-current` on selection; copy button on payloads;
filters (status, review_status); i18n for hardcoded eyebrow text.

### 5.10 Workflows
**Wrong today:** hardcoded workflow-key names (mislabel any new key);
compile doesn't enforce same key; applicability review picks first asset
silently with VALIDATED hardcoded; reviews/evaluations never rendered;
REJECTED/RETIRED candidates dead ends; sequence_consistency 0% lies; compile
button placement ambiguous; selection persists after compile.
**Redesign:** name resolution from data (`description`/`name` on
`WorkflowVersion`, fallback to key); compile validates same workflow_key
(inline error); applicability review becomes a form (site, asset, class,
manufacturer, model, status) — no silent pick; governance trail panel
renders `reviews[]` + `evaluations[]`; REJECTED/RETIRED affordances (view
reason, recompile, delete with confirm); "No analysis" state for missing
sequence data; compile button grouped with its selection list; selection
cleared + success toast after compile; effective_at/published_by shown.

### 5.11 Quality
**Wrong today:** refresh busy lies (`await refetch()` returns void);
`generated_at` never rendered; raw micros/tokens; "00%" padStart; three
number formats; no trends/drill-down.
**Redesign:** real async refresh (fix `useQuery.refetch` to return a
promise); "Last updated {generated_at}" via `RelativeTime`; format micros →
currency and tokens with `Intl.NumberFormat`; percent via locale formatter
("0%", not "00%"); optional auto-poll; per-site/per-asset breakdown and
trend chart + export (Phase 2, API-dependent).

## 6. New pages & features (summary)

- `/executions` — executions list (new route + sidebar item).
- Site switcher (topbar, global).
- Global search trigger (topbar → `/search?q=`).
- Command palette (Ctrl/Cmd+K) — navigation + actions (Phase 4).
- Toast/notification system; ConfirmDialog everywhere destructive.
- Incident timeline + notes + assignment + reopen (API-dependent).
- Document content viewer (read-only, Phase 1; download Phase 2).
- Report revision history + edit (API-dependent).
- EmptyState/ErrorState on every surface; keyboard queue nav (j/k, Phase 4).

## 7. Accessibility baseline (applies to every page)

Skip-to-content link in the console shell; one h1 per page, ordered h1→h2→h3;
global `:focus-visible`; Radix dialogs (focus trap, ESC, labelled); focus
management on list selection; `aria-current`/`aria-selected`; `aria-live`
for toasts and async list updates; no `window.prompt` anywhere; labels via
`FormField` (htmlFor/id); `lang` attribute synced on locale change;
`prefers-reduced-motion` honored (skeleton shimmer collapsed).

## 8. Implementation phases (each ends testable, green, committed)

- **Phase 0 — Foundation:** Geist swap + console tokens/tone system; route
  config + route-driven breadcrumbs; `useQuery`/`useMutation`/`SiteContext`/
  `PrincipalProvider`; `Dialog`, `ConfirmDialog`, `ToastProvider`,
  `DataTable`, `StatusBadge`, `EmptyState`, `ErrorState`, `Skeleton`,
  `FormField`, `RelativeTime`, `MetricCard`; console shell (skip link,
  topbar, sidebar update, `/executions` route); tests for all primitives.
- **Phase 1 — Operations:** Dashboard, Incidents queue + detail,
  Executions list, Workbench (steps/measurements/observations/evidence,
  complete action, confirmations, conflict dialog).
- **Phase 2 — Knowledge & records:** Assets, Reports, Documents, Search
  (+ document viewer, report edit if API available).
- **Phase 3 — Learning:** Handover, Demonstrations, Workflows, Quality.
- **Phase 4 — Global polish:** command palette, keyboard nav, notifications
  center, print view, e2e suite (Playwright) covering the core flows at
  mobile + desktop, final a11y pass.

## 9. Explicit decisions & flagged API dependencies

- Console keeps dark-first (no light mode) per the design system.
- Console button radius 8px (not pill): pill is the marketing CTA language;
  the console is internally consistent. Documented deviation.
- Backend-dependent items (flagged, not blocked): incident timeline/notes/
  assignment/reopen; execution observations write + completion state
  transition (verify exists server-side); server-side filters/pagination;
  report edit + revision history; document content download; quality
  trends/export. Frontend implements fallbacks (client-side filter/sort,
  read-only structured rendering) so every phase ships testable.
- All i18n: every new label gets EN + VI (existing `MessageKey`/`Record`
  typing enforces this).

## 10. Success criteria

1. Every current flow reachable in ≤ 2 clicks; zero dead routes/links.
2. Zero silent failures: every mutation → success toast or error toast with
   retry; version conflicts surfaced with recovery.
3. Single tone/state/badge language everywhere; no raw enum or JSON dumps.
4. Multi-site principals get correct, switcher-driven scope on every page.
5. a11y: skip link, one h1/page, focus rings, labeled dialogs, no prompts.
6. All existing unit tests pass; new per-primitive tests; Playwright e2e for
   dashboard → incident → execution and multi-site flows.
7. `web-check` (lint + unit + build) green after every phase.
