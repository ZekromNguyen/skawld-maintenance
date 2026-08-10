# Skawld Console Weakness Fixes — Design (Brainstorm)

Date: 2026-08-08
Status: Draft for review (pre-implementation)
Scope: All severities from the screen review (7 high, 13 medium, 9 low)
Deliverable: this design doc, then an implementation plan via writing-plans

## 1. Design read

The console is the operator cockpit for industrial maintenance: supervisors,
senior techs, technicians, managers, admins. It already has a strong
foundation: a single tone system (severity/state badges), i18n plumbing, a
shared DataTable, pagination hooks, and per-page tests. The weaknesses found
in review are not about the visual system; they are about **trust and
operability**: data that looks wrong (raw JSON, duplicated rows, misleading
zeroes), state you cannot scan (English enums in a Vietnamese UI, identical
badge tones), and dead ends (no counts, no deep links, empty states with no
call to action).

Guiding principle for this wave: **every number, label, and row an operator
sees must be either truthful or visibly actionable.** No fabricated data, no
JSON dumps, no "100%" that means one sample, no filters that silently only
see page one.

## 2. Workstreams

Fixes are grouped by theme so each workstream lands as one reviewable unit,
frontend and backend together where the truth lives server-side.

- **WS-A Data contracts** — handover raw JSON, report identity, knowledge
  version display.
- **WS-B Server-side filtering & counts** — incident tabs/severity counts,
  dashboard summary numbers, execution state filter.
- **WS-C i18n completeness** — every enum rendered in Vietnamese.
- **WS-D Table information density** — report/knowledge/execution columns
  that let operators tell rows apart.
- **WS-E Empty states & discovery** — CTAs, search affordances, pagination
  feedback, dashboard deep links.
- **WS-F Quality page honesty** — sample sizes, zero-value disclaimers.
- **WS-G Polish & accessibility** — keyboard tabs, relative time, dialog
  focus, minor layout issues.

---

## 3. WS-A: Data contracts

### A1. Handover `open_incidents` renders raw JSON strings (HIGH)

**Current state.** `Content.OpenIncidents []string` in
`internal/handover/application/service.go:32`; the LLM provider emits JSON
objects serialized into those strings (see `describeItems` in
`internal/skawld/deterministic_provider.go:115`, which stringifies items).
`HandoverPanel.tsx:73` renders them verbatim, so an operator sees
`{'asset_tag':'P-302','id':'27c02aee…'…}`.

**Approach options.**

- **A1-1 (recommended): Structured content, normalized at the boundary.**
  Change the schema so `open_incidents`, `active_executions`,
  `safety_concerns`, `follow_up` become `[]ListItem` where
  `ListItem = { title, detail?, severity? }` (`json:"title,detail,severity"`).
  Keep a tolerant decoder: if a slot arrives as a JSON string or a raw
  string, `decodeContent` (handover service) normalizes it into a
  `ListItem` — string → `{title: s}`, JSON string → parsed object. This
  fixes display for both the deterministic provider and any live LLM
  without forcing a prompt re-version.
  *Trade-off:* schema change ripples through `types.ts`, the store's jsonb
  (no migration needed — jsonb is schemaless), and tests. Moderate diff,
  permanent fix.
- **A1-2: Strip to strings server-side.** `decodeContent` keeps `[]string`
  but replaces JSON blobs with `"P-302 · Diagnose high vibration"`.
  *Trade-off:* smallest diff, but loses structured rendering and still
  depends on the provider's string shape.
- **A1-3: Frontend parse + render.** Render a mini-table in `HandoverPanel`
  that JSON.parses each item.
  *Trade-off:* parsing JSON the backend already had in structured form is
  layering violation; fails on plain strings.

**Decision: A1-1 (confirmed 2026-08-08).** Change `Content` to structured
`[]ListItem` (with `severity` on incident items so the panel can badge
them), tolerant decoder, update `types.ts` and
`HandoverPanel`/`HandoverSection` to render title + optional detail lines.
Keep the LLM prompt unchanged for now; the decoder absorbs both shapes.
Also fix the unstable key
`key={`${title}-${index}`}` (HandoverPanel.tsx:97) — use an index-based
stable key with `useId`-style grouping or item title when unique.

### A2. Report identity: `RP-{revision}` collides across documents (HIGH)

**Current state.** `ReportsPage.tsx:78` renders `RP-{report.revision}`;
every document starts at R1, so different reports show identical numbers.
The backend does expose a monotonic identity: report `id` (UUID) and
`execution_id`. The report list query already selects `created_at,
updated_at` (`internal/report/adapter/postgres/store.go:289`), but the
frontend `MaintenanceReport` type omits them.

**Approach options.**

- **A2-1 (recommended): Show the execution/incident context, not a fake
  number.** Rename the column to "Số hiệu" and render a human reference:
  incident number when the execution links to one, else `execution_id`
  prefix; add `updated_at` column. Backend: expose `created_at`/`updated_at`
  on the list payload (they are already selected — add to the domain
  projection), and the incident number (join already exists at
  store.go:45-60 for report detail; extend to list).
  *Trade-off:* medium backend + frontend diff; the real fix — numbers must
  come from the incident, not a per-document counter.
- **A2-2: Add a global report sequence number.** New DB sequence; render
  `RP-{seq}`.
  *Trade-off:* honest numbering but new migration + sequence ownership;
  more work than the symptom needs.
- **A2-3: Just add `updated_at` + asset column.**
  *Trade-off:* incomplete; identity still collides.

**Recommendation: A2-1.** Also add an asset column (report list joins the
execution; asset tag is available via the execution). Two distinguishing
columns plus the incident link make rows scannable.

### A3. Knowledge page: 7 duplicate rows, no version grouping (HIGH)

**Current state.** `KnowledgePage.tsx` lists `KnowledgeDocument.revisions`
as flat rows; `P-302 Bearing Lubrication SOP` appears once per revision with
identical `R1` labels and no "current/effective" marker. Draft vs approved
revisions are indistinguishable.

**Approach options.**

- **A3-1 (recommended): Group by document, expand revisions.** Each row is
  a document (title + document_type + authority badge); revisions collapse
  under it with per-revision state badge, `R1/R2` chip, and a
  "Đang áp dụng" (effective) marker on the current approved revision.
  Add pagination via `api.documents` (page exists) and sort by
  recently-updated.
  *Trade-off:* moderate frontend change; requires the document list to
  expose revision update times (check `DocumentRevision` — add `updated_at`
  if missing).
- **A3-2: Keep flat rows, add columns.** Add `revision`, `approval status`,
  `updated_at` columns to the flat table.
  *Trade-off:* smaller diff, but still noise for multi-revision docs.
- **A3-3: Single latest revision per document + revision detail page.**
  *Trade-off:* cleanest but adds a route; more work.

**Recommendation: A3-1**, with pagination and a sort. If revision timestamps
are absent from the API, add them (small backend addition).

---

## 4. WS-B: Server-side filtering & counts

### B1. Incident tab counts and severity filter only see page one (HIGH)

**Current state.** `IncidentsPage.tsx:56-80` filters client-side after
`usePaginatedList` loads page one, so tabs/counts/severity lie once more
than 25 incidents exist. The backend `Filter` supports `State` (single
string) but **not `Severity`** (`internal/incident/application/service.go:59`),
and the HTTP handler parses `state` only
(`internal/platform/httpserver/incidents.go:34`).

**Approach options.**

- **B1-1 (recommended): Server-side filtering + a counts endpoint.**
  Add `Severity` to `incidentapp.Filter`, wire it through handler → store
  query. Add `GET /incidents/counts?site_id=` returning
  `{open, in_progress, resolved, total, by_severity:{low,medium,high,critical}}`
  so tab badges and severity totals are truthful regardless of page.
  Frontend: pass `state`/`severity` into `api.incidents` (the hook's
  dependency already supports param changes), read counts from the new
  endpoint. Search (`number/summary/asset_tag`) can stay client-side on the
  current page, but the search box should note it searches loaded results,
  or move to a server `q` param if the store supports it.
  *Trade-off:* backend work (filter + counts query + tests), frontend work;
  the correct fix.
- **B1-2: Frontend-only: fetch all incidents.** `fetchAll` then filter.
  *Trade-off:* repeats the dashboard anti-pattern; unbounded payloads.
- **B1-3: Fetch counts via separate per-state queries.** `api.incidents({state: X, page_size: 1})` and read `X-Total-Count`.
  *Trade-off:* requires total-count support in the list response (not
  present today); 5 extra requests.

**Recommendation: B1-1.** Severity filter in SQL; counts come from the
incident-count portion of the single `/summary` endpoint (see B2-1).

### B2. Dashboard fetches all incidents/assets/executions/handovers (HIGH)

**Current state.** `DashboardPage.tsx:18-21` calls four list endpoints with
no `page_size`, and `Overview` aggregates counts in the browser. As data
grows this is unbounded; counts are also wrong if the API caps pages.

**Approach options.**

- **B2-1 (recommended): Dedicated `/summary` endpoint + capped lists.**
  Backend: `GET /summary?site_id=` returns
  `{open_incidents, active_executions, critical_assets, pending_handovers}`
  in one query (cheap COUNT/GROUP BYs). Frontend dashboard uses it for
  metric cards; keeps capped list queries only for the "my queue" and
  incident table (already paginated).
  *Trade-off:* new backend endpoint + tests; the durable fix.
- **B2-2: Cap each query at `page_size: 25` and aggregate locally.**
  *Trade-off:* bounded but counts lie (same bug as B1).
- **B2-3: Keep as-is, note as known limitation.**
  *Trade-off:* acceptable at pilot scale, not at production scale.

**Decision: B2-1 (confirmed 2026-08-08).** Single `GET /summary?site_id=`
endpoint that includes the incident counts (from B1-1) in the same payload,
so there is no separate `/incidents/counts` resource. Dashboard uses it for
metric cards; keeps capped list queries only for the "my queue" and
incident table (already paginated). B1-1's tab badges and severity totals
read the incident-count portion of the same payload.

### B3. Execution state filter is client-side (MEDIUM)

**Current state.** `ExecutionsPage.tsx:26-29` filters by state after
loading page one. The backend `executionapp.Filter` already supports
`State` (`internal/execution/application/service.go:108`).

**Recommendation.** Pass the selected state to `api.listExecutions` and let
the pagination hook refetch on change (same pattern as B1-1, minus the
counts endpoint — tabs without counts don't lie).

---

## 5. WS-C: i18n completeness

**Current state.** `ExecutionsPage.tsx:45,84` renders raw
`ASSIGNED / IN_PROGRESS / COMPLETED`; handover history badges render
`DRAFT/ACCEPTED` (`HandoverPage.tsx`); `ReportsPage` falls back to raw state
when the label key is missing (ReportsPage.tsx:104 — actually fine, keys
exist). `labels.ts` has a clean `*LabelKey` + `*Tone` pattern.

**Recommendation.** Add `executionStateLabelKey`/`executionStateTone` and
`handoverStateLabelKey` (handover tones already exist) to `labels.ts` with
Vietnamese message keys in `messages.ts`, then use them in
`ExecutionsPage` and `HandoverPage`. Also fix the tone mapping so severity
badges are visually distinguishable (see G8).

---

## 6. WS-D: Table information density

### D1. Executions: "SỰ CỐ" and "TÀI SẢN" columns show the same value (MEDIUM)

`ExecutionsPage.tsx:62-77` — when an execution links to an incident, both
columns render `P-302` (asset tag). **Fix:** SỰ CỐ column shows the
incident `number` (link to `/incidents/{id}`), TÀI SẢN shows the asset tag.
Requires incident number on the execution payload or a lookup; check what
`api.execution`/list returns (`execution.incident_id` + `asset_tag` exist).

### D2. Reports table lacks distinguishing columns (HIGH)

Covered in A2-1 (identity + `updated_at` + asset).

### D3. Knowledge lacks pagination and sort (MEDIUM)

Covered in A3-1 (grouped docs + pagination + sort by updated).

### D4. Incident rows: relative time with no absolute tooltip (LOW)

`RelativeTime.tsx` — add `title={absolute datetime}` (and the same on
handover shift window).

---

## 7. WS-E: Empty states & discovery

### E1. Search page is a blank canvas (MEDIUM)

`SearchPage.tsx` — one input + button. **Fix:** autofocus, Enter-key hint,
scope filter (asset / incident / document), recent searches (localStorage),
and a "no results" explanation instead of silent blankness. Keep scope
simple: a segmented control that drives the existing search endpoint.

### E2. Dashboard "Hàng đợi của tôi" empty state has no CTA (MEDIUM)

`Overview.tsx` — replace the single centered line with an empty state
component: title, one-line explanation, and a primary button ("Xem tất cả
quy trình" → `/executions`, or "Tạo quy trình" when the user can create).

### E3. "Sức khỏe tổ chức" chips look clickable but are not (MEDIUM)

`Overview.tsx` — the three chips (Chất lượng và an toàn AI / Sự cố đang mở /
Tài sản quan trọng) should either navigate (quality → `/quality`,
incidents → `/incidents?state=OPEN`, assets → `/assets`) or be restyled as
non-interactive summary rows. **Fix:** make them links; add deep-link
support for `?state=OPEN` on IncidentsPage.

### E4. Load-more button has no progress feedback (MEDIUM)

`IncidentsPage.tsx:183-193` (and ExecutionsPage, ReportsPage). **Fix:** show
"Đang hiển thị X" when a total count is known (B1-1 counts make this free on
Incidents; for others use `items.length` + a "Đã tải X" hint and disabled
spinner state while `loading`).

### E5. Knowledge/reports empty states (LOW)

Ensure both use the shared `EmptyState` with a contextual CTA ("Tạo tài liệu"
/ "Chạy quy trình đầu tiên"), not the bare table placeholder.

---

## 8. WS-F: Quality page honesty

### F1. "Ghi đề của con người 100%" with one sample is misleading (MEDIUM)

`QualityPanel.tsx` — every percentage should show its denominator:
"100% (1/1)". `EvaluationSummary` already carries `recommendations`,
`reviewed`, etc.

### F2. Zero values read as real numbers (MEDIUM)

`QualityPage.tsx` — cards showing `0%` / `$0.00` should render a muted "Chưa
có dữ liệu" (no data) state instead of a hard zero when the underlying
sample is zero (`llm_calls === 0` etc.), keeping the footer disclaimer but
making the primary signal honest at a glance.

---

## 9. WS-G: Polish & accessibility

### G1. Tab lists claim `role="tablist"` but lack keyboard nav (LOW)

`IncidentsPage.tsx:97-110`, `ExecutionsPage.tsx:36-47` — implement
arrow-key navigation, `aria-controls`/`tabpanel` linkage, `aria-selected`
(already present), and `role="tabpanel"` on the table container. Small
shared `Tabs` component (or a `useTabsKeyboard` hook) to avoid duplicating.

### G2. Dialog lacks focus management (LOW)

`CreateIncidentForm.tsx` dialog — verify `Dialog.tsx` traps focus and
restores it on close; add `aria-labelledby` to the title if missing.

### G3. Reports load-more flash (LOW)

`ReportsPage.tsx:39-42` — `useEffect` resets `history` on every
`reports.data` change, making the button blink. **Fix:** reset history only
when the filter/options key changes, not on every data arrival (compare a
`useRef` of the last filter value).

### G4. Topbar stacked identity has no separator (LOW)

`Topbar.tsx` — add a hairline divider and explicit hover states so
role/scope block reads as one unit, not two clickable unknowns.

### G5. Sidebar safety card permanently consumes space (LOW)

`Sidebar.tsx` — make "RANH GIỚI AN TOÀN" dismissible (localStorage flag),
collapsing to a small shield icon with the full text in a tooltip.

### G6. HandoverSection unstable keys (LOW)

Covered in A1-1.

### G7. Handover evidence chips are not links (MEDIUM)

`EvidenceLinks.tsx` — "INCIDENT · INC-20260804-27C02AEE" chips render as
plain text. **Fix:** when the evidence `source_id` is a resolvable incident
(evidence kinds include incident IDs), render a real `<Link>` to
`/incidents/{source_id}`; same for document evidence → knowledge page.

### G8. Severity badges use indistinguishable tones (LOW)

Review screenshot shows "Thấp"/"Cao" in the same dark pill. `severityTone`
exists (`labels.ts:20`) with distinct tones — verify the CSS actually
differentiates `low`/`medium`/`high`/`critical` (likely all dark). **Fix:**
map tones to visually distinct treatments (e.g., outline + colored dot, or
color-coded pills) and add a test asserting tone uniqueness for the four
severities.

---

## 10. Backend changes summary

| Change | Where |
|---|---|
| Handover `Content` → structured `[]ListItem` + tolerant decoder | `internal/handover/application/service.go` (+ tests) |
| Report list payload: expose `created_at`, `updated_at`, incident number, asset tag | `internal/report/application` + `adapter/postgres/store.go` (+ tests) |
| Incident filter: add `Severity`; store query | `internal/incident/application/service.go:59`, `internal/incident/adapter/postgres/store.go` |
| New `GET /summary` (dashboard metrics + incident counts) | `internal/platform/httpserver` |
| Document revision `updated_at` if missing (for knowledge sort) | `internal/knowledge` (verify) |

All backend changes ship with unit/integration tests matching the existing
patterns (`*_test.go` alongside handlers and services).

## 11. Frontend changes summary

| Change | Where |
|---|---|
| Structured handover rendering + stable keys | `HandoverPanel.tsx`, `types.ts` |
| Server-driven incident filters + counts | `IncidentsPage.tsx`, `usePaginatedList` usage |
| Summary-driven dashboard metrics + capped lists | `DashboardPage.tsx`, `Overview.tsx` |
| Execution server-side filter + i18n states | `ExecutionsPage.tsx`, `labels.ts`, `messages.ts` |
| Report identity/columns (updated, asset, incident link) | `ReportsPage.tsx`, `types.ts` |
| Knowledge document grouping + pagination + effective marker | `KnowledgePage.tsx` |
| Empty states + CTAs + load-more feedback | `Overview.tsx`, `SearchPage.tsx`, list pages |
| Quality sample sizes + no-data states | `QualityPanel.tsx`, `QualityPage.tsx` |
| A11y: tabs keyboard, dialog focus, evidence links, topbar/sidebar polish | layout/ui/components |

## 12. Phasing

**Phase 1 — truth (WS-A + WS-B):** handover raw JSON, report identity,
incident counts/filters, dashboard summary, knowledge grouping. This is
where operators currently see lies.

**Phase 2 — scan & labels (WS-C + WS-D):** i18n states, execution filter,
report/execution columns, knowledge pagination/sort.

**Phase 3 — guidance & polish (WS-E + WS-F + WS-G):** empty states, CTAs,
quality honesty, a11y, layout polish.

Each phase is independently shippable and testable.

## 13. Testing strategy

- Extend the existing per-page tests (`pages/*.test.tsx`) for every changed
  render path; follow the established test conventions (see
  `IncidentsPage.test.tsx` for the list-page pattern).
- Backend: unit tests for the tolerant handover decoder and incident
  severity filter; handler tests for `/counts` and `/summary`.
- Tone test asserting the four severity tones are visually distinct.
- A11y: keyboard navigation test for the shared tabs component.
- Manual pass on every screen after each phase using the existing seed data.

## 14. Decisions (confirmed 2026-08-08)

1. **Counts shape:** one `GET /summary` endpoint that includes the
   incident counts; no separate `/incidents/counts` resource.
2. **Handover schema:** widen `Content` to structured `[]ListItem`
   (`{title, detail?, severity?}`) with a tolerant decoder for old stored
   JSON/string shapes; the LLM prompt is unchanged for now.
3. **Remaining decisions deferred to implementation:** report identity
   fallback when an execution has no linked incident; the "current
   revision" rule for the knowledge effective marker (latest APPROVED vs
   latest regardless of state); server-side vs client-side incident search
   (`q` param vs "tìm trong trang đã tải" hint). Each is flagged inline in
   the relevant section and will be resolved during planning/implementation
   with the smallest truthful option.
