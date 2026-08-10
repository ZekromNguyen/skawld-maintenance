# Skawld Console Cockpit — Visual Redesign (Linear-class, Light-first)

Date: 2026-08-08
Status: Approved design (pre-implementation)
Scope: Shell + Dashboard + Incidents (list/board/detail) + Execution workbench
Companion: [console-weakness-fixes-design](./2026-08-08-console-weakness-fixes-design.md)
handles trust/operability defects (data contracts, counts, i18n). This spec
is the visual layer and assumes those fixes land first or in parallel.

## 1. Design read

Skawld is an **industrial maintenance intelligence layer**. The console is the
operator cockpit: supervisors, senior techs, technicians, managers, admins.
The design system doc demands a **calibrated instrument**: precise,
trustworthy, dense, mono numerals, hairline grids, emerald accent, Geist type.

The user's brief: the current UI "feels too simple and lacks anything
outstanding." We evaluated Jira's UX/UI against the system:

- **Fits Skawld (borrow):** status-driven queues, dense list/board triage,
  saved views, detail page with metadata rail + activity feed, keyboard-first
  command palette.
- **Does not fit (reject):** Jira's light-blue visual style (conflicts with
  the locked emerald brand), drag-and-drop reordering (execution steps are
  LOTO-gated and ordered; casual reordering undermines evidence/authority),
  JQL query language (field technicians need guided filter chips), the
  "everything is an issue" model (flattens asset/incident/execution/handover/
  report distinctions), playful micro-interactions (wrong safety register).

**Decision (user-approved):** adopt Jira's information architecture at
Linear/Height-grade polish, keep Skawld's dark-and-light emerald instrument
identity, and default the console to **light** (dark remains the toggle).
The redesign targets execution polish and structure, not the palette.

### Signature

The console's signature is the **annunciator strip**: a control-room style
readout band of live operational counts, severity-coded edges, large mono
numerals, whole-cell click-through. It re-expresses the current flat
`MetricCard` row into the one memorable instrument element, and its language
(severity edges + mono numerals on hairline grids) propagates to the incident
alarm-log rows and board cards. Everything else stays disciplined.

## 2. Token system (re-expression of the locked design system)

No new palette. The light console tokens from `web/src/styles.css:5-53` are
kept and refined:

| Token | Value (light) | Use |
|---|---|---|
| `--bg` | `#f4f7f5` | Page background |
| `--surface` | `#ffffff` | Cards, panels |
| `--ink` | `#14201a` | Primary text |
| `--brand` | `#12734a` | Emerald accent (AA on light) |
| `--red` | `#b3372c` | Critical severity |
| `--amber` | `#96610f` | High/medium severity |
| `--blue` | `#2a6fb0` | Informational (sparingly) |

- **Numerals:** `font-variant-numeric: tabular-nums` on all numeric cells,
  counts, and measurements (a new `.mono-num` utility) so values never jitter.
- **Type:** Geist + Geist Mono (unchanged). Console base stays 12-13px.
- **Radius / shadows:** unchanged per design system §2.3-2.4.
- **Dark theme:** unchanged tokens; the toggle persists via
  `ThemeProvider`.

## 3. Shell — persistent global chrome

**Problem:** each page renders its own `Topbar`; the sidebar carries theme,
language, and safety chrome mixed with navigation.

**Change:** split chrome into a persistent `GlobalBar` (new) + page-local
`PageHeader` (kept, slimmed).

- `ConsoleLayout` (`web/src/console/layout/ConsoleLayout.tsx`): render a
  slim `GlobalBar` above `<main>` spanning the content column: command
  palette trigger (⌘K, opens existing `CommandPalette`), global search input,
  `SiteSwitcher`, operator presence, theme toggle, language switch.
- `PageHeader` keeps breadcrumbs + title + actions (drop the search input,
  site switcher, and operator from `PageHeader.tsx:29-58`).
- `Sidebar` (`web/src/console/layout/Sidebar.tsx`): remove theme toggle and
  lang switch from the footer (lines 106-122); keep brand, nav, safety
  boundary, logout. Nav section headings may carry **live real counts**
  (open incidents, in-progress executions, pending handovers) sourced from
  the same data as the dashboard; no fabricated or decorative numbers.

## 4. Dashboard — annunciator strip

Replace the four flat `MetricCard`s in `Overview.tsx:59-84` with an
`AnnunciatorStrip` (new component):

- One band of four cells, each: severity-coded left edge (red/amber/red/blue
  for open incidents / in-progress executions / critical assets / pending
  handovers), a mono 32px tabular numeral, a small uppercase label, and an
  eyebrow sub-label. Whole cell is a `Link`.
- Values are the live counts already computed in `Overview.tsx:44-46`
  (no fabricated metrics; empty data renders a truthful `0`).
- Count transitions animate `opacity`/`transform` only; collapsed entirely
  under `prefers-reduced-motion`.
- Below the strip: role focus panel (kept) and the active-incident
  **alarm table** (below).

## 5. Incidents — alarm-log queue + saved views

`IncidentsPage` (`web/src/console/pages/IncidentsPage.tsx`) keeps state tabs
and the list/board toggle. Additions:

- **Filter chip row:** severity and site chips (guided, no query language),
  plus **saved views** — persisted named filter combinations (localStorage,
  mirroring the existing `VIEW_STORAGE_KEY` pattern at
  `IncidentsPage.tsx:30`). Default seed: "My critical queue".
- **List view rows become alarm-log rows:** left severity edge stripe (3px,
  `--red`/`--amber`/`--blue` by severity), mono incident number, summary,
  asset tag, relative time, state `StatusBadge`. Implemented as a new
  `SeverityEdge` style + a `.alarm-row` variant in `DataTable` or a dedicated
  row renderer.
- **Board view** (`IncidentBoard.tsx`): cards gain a top severity edge
  stripe; keep icon + mono number + meta.
- **Incident detail** (`IncidentDetailPage.tsx`): Linear two-column anatomy:
  main column (summary h1, badges, description, timeline/evidence feed) and a
  right `MetadataRail` (state, severity, asset, detected_at, linked
  execution, report, evidence links). Reuses existing panels; the rail is a
  new layout wrapper.

## 6. Execution workbench — procedure cockpit

`ExecutionWorkbenchPage.tsx` + `StepRail.tsx` keep the step rail and
confirmations. Add:

- **LOTO gate readout:** a prominent mono banner at the top of the step rail
  showing real gate state (e.g. "LOTO ACTIVE - intrusive step gated") with
  the `--amber`/`--red` tone of the blocking step. Real state only; renders
  nothing when no gate is active.
- **Right metadata rail:** execution meta, incident link, evidence, and a
  measurements summary (existing data, new layout wrapper).
- Three-region layout: step rail | work area | metadata rail.

## 7. New shared primitives

| Primitive | Where |
|---|---|
| `AnnunciatorStrip` | `web/src/console/components/` |
| `FilterChip` + `SavedViews` | `web/src/console/components/` |
| `MetadataRail` (layout wrapper) | `web/src/console/ui/` |
| `.alarm-row` + `SeverityEdge` style | `web/src/styles.css` |
| `.mono-num` (tabular-nums utility) | `web/src/styles.css` |

## 8. Accessibility, motion, copy

- All design-system §6 rules hold: text+color status (severity is never
  color alone), focus-visible rings, labels above inputs, keyboard nav.
- Motion: CSS only, `transform`/`opacity`, every animation has a reason
  (state change / feedback), `prefers-reduced-motion` collapses all
  transitions. No scroll listeners, no marquee.
- Copy: sentence case, active voice, existing i18n keys extended via
  `web/src/i18n/messages.ts` (en + vi); no em dashes; no decorative labels.

## 9. Out of scope (this wave)

- Other console pages (knowledge, search, reports, handovers,
  demonstrations, workflows, quality): pattern propagation later.
- Marketing page (`/landing`): untouched (has its own design system rules).
- Flutter technician client: untouched.

## 10. Testing

- Existing per-page tests (`IncidentsPage.test.tsx`, `DashboardPage.test.tsx`,
  `ExecutionWorkbenchPage.test.tsx`, `ConsoleLayout.test.tsx`,
  `Sidebar.test.tsx`) must pass; extend where structure changes.
- New component tests for `AnnunciatorStrip` (truthful zeroes, link targets,
  aria), `SavedViews` (persist/restore), `MetadataRail`.
- `web/e2e/*.spec.ts` still pass; run `npm test` and the Playwright suite.
- Manual check: light + dark, en + vi, keyboard-only nav, reduced motion.
