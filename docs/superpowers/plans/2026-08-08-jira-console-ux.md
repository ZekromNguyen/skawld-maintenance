# Jira-IA Console Restyle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the Skawld console toward Jira's information architecture (sidebar groups, tabbed "For you" queue, dense triage table, metadata rail + activity feed, keyboard-first palette) while keeping Skawld's emerald instrument identity and light-first theme. Does not reopen the cockpit or weakness-fixes decisions.

**Architecture:** Pure frontend. Borrow Jira's IA and interaction patterns; render in Skawld's existing token system (`web/src/styles.css`). New primitives (`ForYouTabs`, `ActivityFeed`, `useListKeyboard`, `FilterRow`) compose existing pages. All data comes from existing API contracts (`api.ts`, `types.ts`); no invented fields. All copy through `web/src/i18n/messages.ts` (en + vi).

**Tech Stack:** React 19, react-router 8, Vitest + Testing Library, Phosphor icons, plain CSS, Radix Dialog/Tabs primitives where already used.

## Global Constraints

- Emerald `--brand` is the only accent; no Atlassian blue, no pure black/white (design system §2.1).
- Console density dial 8: hairline grids, mono/tabular numerals (`mono-num`), no playful micro-interactions.
- Motion only `transform`/`opacity`; collapsed under `prefers-reduced-motion`.
- Status never conveyed by color alone; severity rows carry a label badge AND a colored edge.
- i18n: every new label added to BOTH `en` and `vi` in `web/src/i18n/messages.ts` (same `MessageKey`). No hardcoded English in components.
- No em dashes in copy; sentence case; active voice.
- One `h1` per page; `:focus-visible` rings preserved; AA contrast in both themes.
- Do not invent API fields. If a Jira element has no backing field, drop it or render a truthful empty/loading state.
- Existing tests must pass; extend tests where structure changes.
- Reject: JQL (guided filter chips only), drag-and-drop step reordering, "everything is an issue", right-rail promo cards (pending section 0 decision; default reject).
- `web/scripts/verify-console.py` must still pass (nav labels "Incident execution", "Reports" and h1 selectors unchanged).

---

### Task 1: Dashboard "For you" tabbed queue

**Files:**
- Create: `web/src/console/components/ForYouTabs.tsx`
- Create: `web/src/console/components/ForYouTabs.test.tsx`
- Modify: `web/src/console/pages/DashboardPage.tsx` (whole file)
- Modify: `web/src/console/pages/DashboardPage.test.tsx`
- Modify: `web/src/console/components/Overview.tsx` (drop the queue/table sections; keep annunciator + focus panel)
- Modify: `web/src/styles.css` (`.for-you-*`, `.queue-row`, `.queue-row.selected`, tab pills)

**Interfaces:**
- Consumes: `useQuery` results from `api.incidents()`, `api.listExecutions()`, `api.assets()`, `api.pendingHandovers()`; `usePrincipal`; `useI18n`; `useNavigate`; `SavedViews` storage keys (`skawld.incidents.savedViews`).
- Produces: `ForYouTabs` component with props `{ incidents, executions, assets, pendingHandoverCount, loading, error, onRetry }`; 4 tabs (Recommended / Assigned to me / Starred / Viewed), each rendering dense rows with type icon, key, breadcrumb, relative time; active tab persisted to `skawld.dashboard.foryouTab`.

**Behavior:**
- Tabs with count badges from REAL data: Recommended = open incidents count; Assigned to me = in-progress executions count; Starred = saved views count (from localStorage via the `SavedViews` shape); Viewed = recently-viewed count (localStorage `skawld.dashboard.recent`).
- Row anatomy: severity/type icon, mono number (incident number or execution id prefix), title, `asset_tag` breadcrumb, `RelativeTime`.
- Whole-row click-through to detail; `:focus-visible` ring on rows; hover raises surface; selected row persists tint + left severity edge.
- Empty state per tab ("No spaces found" equivalent) with CTA link.
- Loading shows skeletons; error shows `ErrorState` with retry.
- AnnunciatorStrip stays as the signature band above the tabs (rendered by Overview).
- Tabs update without full-page reload; counts never hardcoded.

**Step 1: Write the failing test**

`web/src/console/components/ForYouTabs.test.tsx` — render with fixture incidents/executions, assert: Recommended tab shows open incident rows and count; Assigned to me tab shows in-progress executions; empty tab shows empty-state CTA; clicking a row calls the navigate target; switching tabs persists to localStorage.

**Step 2: Run test to verify it fails**

Run: `npx vitest run src/console/components/ForYouTabs.test.tsx`
Expected: FAIL (module not found)

**Step 3: Implement `ForYouTabs`**

Tab pills use `.tab` + `.tab.active` with `.count` badges. Rows are `<Link>`-wrapped `.queue-row` elements with `.alarm-edge` severity stripe. Recent-viewed tracking helper writes `skawld.dashboard.recent` when a row opens.

**Step 4: Rewire `DashboardPage`/`Overview`**

`DashboardPage` keeps its four queries and renders `<PageHeader>` + `<Overview>` (annunciator + focus panel) + `<ForYouTabs>`. Remove the my-queue and active-incident tables from `Overview` (they move into the tabs).

**Step 5: Run tests**

`npx vitest run src/console/pages/DashboardPage.test.tsx src/console/components/ForYouTabs.test.tsx` — PASS, existing dashboard assertions updated to new tab surface.

---

### Task 2: Sidebar Jira grouping (primary / recents / recommended / cross-links / customize)

**Files:**
- Modify: `web/src/console/layout/Sidebar.tsx` (whole file)
- Modify: `web/src/console/layout/Sidebar.test.tsx`
- Modify: `web/src/i18n/messages.ts` (add `sidebar.primary`, `sidebar.recents`, `sidebar.recommended`, `sidebar.crossLinks`, `sidebar.customize`)
- Modify: `web/src/styles.css` (section heading styles already exist; count badge on nav item)

**Behavior:**
- Groups: **Primary** (Operations: Overview, Incidents, Executions, Handover), **Recents** (recent sites from `principal.site_ids` when > 1, else hidden), **Recommended** (role focus links from `FOCUS_CONFIG`, permission-filtered), **Cross-links** (Assets, Knowledge, Search, Reports, Quality, Demonstrations, Workflows), **Customize** (theme toggle + language select moved back from GlobalBar footer-equivalent; keep sign-out and safety boundary in the footer).
- Nav items keep `report:write` gating; aria-current; active styles.
- No fabricated counts.

**Step 1: Write the failing test** — assert group headings render, cross-link gating for `report:write`, recents hidden for single-site principals.

**Step 2-5:** implement, run `npx vitest run src/console/layout/Sidebar.test.tsx src/console/layout/ConsoleLayout.test.tsx src/console/layout/GlobalBar.test.tsx`, keep green.

---

### Task 3: Incidents queue — filter row alignment + selected row + keyboard nav

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx` (filter row placement; selected-row state; j/k nav)
- Modify: `web/src/console/pages/IncidentsPage.test.tsx`
- Create: `web/src/console/useListKeyboard.ts`
- Create: `web/src/console/useListKeyboard.test.tsx`
- Modify: `web/src/console/ui/DataTable.tsx` (accept `selectedKey`, `onSelect`, expose row index for keyboard nav; focus ring on rows)
- Modify: `web/src/styles.css` (`.data-row.selected`, row focus ring)

**Behavior:**
- Saved views + filter chips align into a Jira-style filter row directly above the table (SavedViews row + severity chips + search + view toggle in one bar).
- Row hover raises; selected row persistent tint + left severity edge; whole-row click-through opens detail; `:focus-visible` ring on rows.
- `useListKeyboard`: `j`/`k` move selection, `Enter` opens selected row, `Esc` clears; returns `{selectedIndex, setSelectedIndex, onKeyDown}`.
- Tabs and chips update list without reload; counts from loaded data.

**Step 1: Write failing test** for `useListKeyboard` (j moves down, k moves up, Enter triggers open, clamps bounds) and extend `IncidentsPage.test.tsx` (selected row class, j/k navigation).

**Step 2-5:** implement, run the two test files + `primitives.test.tsx`.

---

### Task 4: Incident detail — activity feed below content

**Files:**
- Create: `web/src/console/components/ActivityFeed.tsx`
- Create: `web/src/console/components/ActivityFeed.test.tsx`
- Modify: `web/src/console/pages/IncidentDetailPage.tsx` (render feed under the executions table in `.detail-main`)
- Modify: `web/src/styles.css` (`.activity-*`)

**Behavior:**
- Timeline of REAL events for the incident: detection (`detected_at`), linked executions (purpose + state), measurements recorded (`observed_at`), evidence attachments. No invented timestamps: entries without a timestamp render as context rows.
- Each entry: mono timestamp or relative time, type icon, label, and link where a route exists.
- Empty state: "No activity recorded yet."

**Step 1-5:** TDD; run `ActivityFeed.test.tsx` + `IncidentDetailPage.test.tsx`.

---

### Task 5: Create dialogs — Jira modal layout

**Files:**
- Modify: `web/src/console/components/CreateIncidentForm.tsx` (site context switcher at top; filter asset select by chosen site)
- Modify: `web/src/console/components/CreateAssetForm.tsx` (align layout; required markers)
- Modify: `web/src/i18n/messages.ts` (`form.siteContext`, `form.selectSite`, `form.selectAssetForSite`)
- Modify: `web/src/styles.css` (`.dialog-context-row`)

**Behavior:**
- Jira create-issue modal anatomy: context switcher row (site) above fields, required markers (`FormField required` already renders `*`), stacked fields, footer actions.
- `CreateIncidentForm` gains a site selector that filters the asset dropdown to assets on that site; validation unchanged; tests updated.

**Step 1-5:** TDD; run `IncidentsPage.test.tsx` + `AssetsPage.test.tsx`.

---

### Task 6: Keyboard layer — `/` opens palette, shortcuts documented

**Files:**
- Modify: `web/src/console/components/CommandPalette.tsx` (listen for `/` when not typing in an input)
- Modify: `web/src/console/components/CommandPalette.test.tsx`
- Modify: `web/src/console/layout/GlobalBar.tsx` (kbd hint shows `/` too; keep ⌘K)
- Modify: `web/src/console/components/ShortcutCard.tsx` (document keyboard shortcuts: `/` palette, `j/k` rows, `Enter` open)
- Modify: `web/src/i18n/messages.ts` (`cmd.slashHint`, `shortcuts.*`)

**Behavior:**
- `/` or ⌘K opens palette; palette input focused; Esc closes.
- ShortcutCard renders a compact shortcut legend (non-navigable variant) used on the dashboard.

**Step 1-5:** TDD; run `CommandPalette.test.tsx` + `GlobalBar.test.tsx`.

---

### Task 7: i18n + CSS polish + responsive

**Files:**
- Modify: `web/src/i18n/messages.ts` (all new keys, en + vi, alphabetical placement)
- Modify: `web/src/styles.css` (tab pills, queue rows, activity feed, dialog context row, nav count badges; `@media (max-width: 767px)` sidebar collapses to icon-only nav; ensure no horizontal overflow at 360px)

**Behavior:**
- Sidebar below md (768px): icon-only nav items, labels visually hidden.
- No horizontal overflow at 360px on any console screen (verify via existing overflow e2e pattern).

**Step 1-3:** implement; `npx tsc -b` clean; `npm test` green.

---

### Task 8: e2e specs — one per redesigned screen

**Files:**
- Create: `web/e2e/console-happy.spec.ts` (dashboard + incidents happy path with mocked API routes)
- Create: `web/e2e/console-empty.spec.ts` (empty states)
- Create: `web/e2e/console-gated.spec.ts` (permission-gated create actions)

**Behavior:**
- Use `page.route("**/api/v1/**")` to fulfill fixture JSON (principal, incidents, executions, assets, handovers) so specs run without Keycloak.
- Happy: dashboard renders annunciator + tabs; incidents table renders rows; detail renders metadata rail + activity feed.
- Empty: empty list fixtures render empty-state CTAs.
- Gated: principal without `incident:create` hides create buttons.
- No horizontal overflow at 360px asserted in happy spec.

**Step 1-3:** write specs; run `npx playwright test` (needs `npm run dev`; verify server startup). If the full stack is unavailable, specs still assert via mocked routes.

---

### Task 9: Full verification

- Run `npm test` in `web/` — all green.
- Run `npx tsc -b` — no errors.
- Run `npx playwright test` — green (mocked routes).
- Run `python3 web/scripts/verify-console.py` if the stack is available; otherwise confirm selectors unchanged (nav labels "Incident execution", "Reports"; workbench h1).
- Manual note: visual pass against reference images deferred (images not attached in this session; only the Home screen was listed as attached in the prompt).

---

## Self-review checklist

- Spec coverage: section 3 screens 1-5 all have tasks (dashboard tabs, incident queue, incident detail, workbench unchanged, create dialogs); section 4 adopt list (queues, dense table, saved views, tab pills, row hover/selected, metadata rail + activity feed, keyboard palette, relative time, sidebar grouping) all present; reject list honored.
- Placeholder scan: no TBD/TODO in task steps.
- Type consistency: `ForYouTabs` props match `DashboardPage` query data; `useListKeyboard` return shape used consistently by `DataTable` and pages.
