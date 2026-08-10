# Console Weakness Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the console screen weaknesses from the 2026-08-08 review: truthful data (no raw JSON, no page-one-only filters, no colliding report numbers), Vietnamese labels everywhere, scannable tables, and actionable empty states.

**Architecture:** Backend-first. Phase 1 adds the truth servers must own: a structured handover content type with a tolerant decoder, an incident severity filter, a single `GET /summary` counts endpoint, and report list identity columns. Phase 2 makes every enum Vietnamese and moves execution filtering server-side. Phase 3 is frontend guidance: empty states, search affordances, quality honesty, a11y, and layout polish. Frontend pages consume the new backend contract through `web/src/api.ts`; labels stay in `labels.ts` + `messages.ts`.

**Tech Stack:** Go (chi, pgx, keyset pagination), React 18 + TypeScript + react-router, vitest + Testing Library, i18n via `messages.ts` (en/vi).

## Global Constraints

- Every new message key MUST be added to BOTH `en` and `vi` objects in `web/src/i18n/messages.ts` (`vi: Record<MessageKey, string>` fails to compile if any key is missing).
- Never render a raw enum or raw JSON to an operator: every badge/label goes through `labels.ts` or a message key.
- Every backend change ships with a test in the same package; run `go test ./...` before committing.
- Every frontend change ships with a test following the `IncidentsPage.test.tsx` convention (vi.mock the api module, wrap in I18nProvider/PrincipalProvider/SiteProvider/ToastProvider/MemoryRouter); run `npm test -- --run` in `web/` before committing.
- jsonb columns need no migration; only schema-level changes to Go structs and JSON shapes.
- Commits are small and per-task; message format per repo convention (imperative, under 72 chars).

---

## Phase 1 — Truth (backend contracts first)

### Task 1: Handover content becomes structured `[]ListItem` with tolerant decoder

**Files:**
- Modify: `internal/handover/application/service.go:30-39` (Content struct), `:316-341` (decodeContent/validateContent)
- Modify: `internal/handover/application/service_test.go`
- Modify: `internal/skawld/deterministic_provider.go:115` (`describeItems` call site)

**Interfaces:**
- Produces: `type ListItem struct { Title string \`json:"title"\`; Detail string \`json:"detail,omitempty"\`; Severity string \`json:"severity,omitempty"\` }` and `Content.OpenIncidents []ListItem` (same for `ActiveExecutions`, `SafetyConcerns`, `FollowUp`, `Unknowns`). `decodeContent` accepts BOTH the old `[]string` shape and the new `[]ListItem` shape.
- Consumes: none (this is the root contract change; Task 2 consumes it).

- [ ] **Step 1: Write the failing tests**

Add to `internal/handover/application/service_test.go`:

```go
func TestDecodeContentAcceptsStructuredListItem(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":[{"title":"P-302 vibration","detail":"High vibes","severity":"HIGH"}],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "P-302 vibration" {
		t.Fatalf("open_incidents = %+v, want structured item", content.OpenIncidents)
	}
	if content.OpenIncidents[0].Severity != "HIGH" {
		t.Fatalf("severity = %q, want HIGH", content.OpenIncidents[0].Severity)
	}
}

func TestDecodeContentAcceptsLegacyStringList(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":["P-302 vibration"],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "P-302 vibration" {
		t.Fatalf("open_incidents = %+v, want legacy string normalized to title", content.OpenIncidents)
	}
}

func TestDecodeContentNormalizesJsonStringItems(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":["{\"asset_tag\":\"P-302\",\"summary\":\"High vibes\"}"],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "High vibes" {
		t.Fatalf("open_incidents = %+v, want embedded summary promoted to title", content.OpenIncidents)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handover/application/ -run TestDecodeContent -v`
Expected: FAIL — `decodeContent` with `DisallowUnknownFields` rejects `"title"` fields; `[]string` unmarshal fails against the new struct.

- [ ] **Step 3: Implement structured content + tolerant decode**

In `internal/handover/application/service.go`, replace the `Content` struct fields:

```go
type ListItem struct {
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type Content struct {
	Summary             string     `json:"summary"`
	OpenIncidents       []ListItem `json:"open_incidents"`
	ActiveExecutions    []ListItem `json:"active_executions"`
	SafetyConcerns      []ListItem `json:"safety_concerns"`
	FollowUp            []ListItem `json:"follow_up"`
	EvidenceIDs         []string   `json:"evidence_ids"`
	Unknowns            []ListItem `json:"unknowns"`
	RequiresHumanReview bool       `json:"requires_human_review"`
}
```

Replace `decodeContent` so it tolerates both shapes (a custom `UnmarshalJSON` on `ListItem` is the cleanest):

```go
// UnmarshalJSON accepts the legacy string form (normalized to Title) and the
// structured {title, detail, severity} form, including a JSON object that a
// provider serialized into a string.
func (item *ListItem) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		var embedded map[string]string
		if json.Unmarshal([]byte(s), &embedded) == nil && len(embedded) > 0 {
			item.Title = firstNonEmpty(embedded["summary"], embedded["title"], embedded["name"], s)
			item.Severity = embedded["severity"]
			return nil
		}
		item.Title = s
		return nil
	}
	type alias ListItem
	var plain alias
	if err := json.Unmarshal(raw, &plain); err != nil {
		return err
	}
	*item = ListItem(plain)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
```

Update `validateContent` so list-length checks stay bounded (same limits, now on `[]ListItem`):

```go
func validateContent(content Content) error {
	if !content.RequiresHumanReview || strings.TrimSpace(content.Summary) == "" ||
		len(content.Summary) > 5000 || len(content.OpenIncidents) > 100 ||
		len(content.ActiveExecutions) > 100 || len(content.SafetyConcerns) > 100 ||
		len(content.FollowUp) > 100 || len(content.Unknowns) > 100 {
		return ErrInvalid
	}
	return nil
}
```

Update `internal/skawld/deterministic_provider.go:115` — `describeItems` already returns `[]string`; the new decoder normalizes strings, so **no change is required there**. Verify `go build ./...` passes (any other consumer of `Content.OpenIncidents` as `[]string` must be updated — grep `OpenIncidents` and fix compile errors).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/handover/... ./internal/skawld/...`
Expected: PASS (including existing `TestDecodeContentRejectsUnknownEvidence` and provider tests).

- [ ] **Step 5: Commit**

```bash
git add internal/handover/application/service.go internal/handover/application/service_test.go
git commit -m "fix: normalize handover content into structured items"
```

---

### Task 2: HandoverPanel renders structured items with stable keys

**Files:**
- Modify: `web/src/types.ts:185-206` (ShiftHandover structured_content)
- Modify: `web/src/console/components/HandoverPanel.tsx:73-105` (HandoverSection + keys)
- Modify: `web/src/console/components/HandoverPanel.test.tsx` (create if missing)

**Interfaces:**
- Consumes: `Content.OpenIncidents: ListItem[]` from Task 1, where `ListItem = { title: string; detail?: string; severity?: string }`.
- Produces: `HandoverSection({ title, items, empty })` where `items: { title: string; detail?: string }[]`; renders `<li>` per item with optional detail line.

- [ ] **Step 1: Write the failing test**

Create `web/src/console/components/HandoverPanel.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "../../i18n/I18nProvider";
import { HandoverPanel } from "./HandoverPanel";
import type { ShiftHandover } from "../../types";

function renderPanel(handover?: Partial<ShiftHandover>) {
  const base: ShiftHandover = {
    id: "h1", site_id: "s1",
    shift_start: new Date().toISOString(), shift_end: new Date().toISOString(),
    state: "DRAFT", version: 1,
    structured_content: {
      summary: "Handover summary",
      open_incidents: [{ title: "P-302 vibration", severity: "HIGH" }],
      active_executions: [], safety_concerns: [], follow_up: [],
      evidence_ids: [], unknowns: [], requires_human_review: false
    },
    evidence: [], provider: "deterministic", model: "mock", prompt_version: "v1"
  };
  return render(
    <I18nProvider>
      <HandoverPanel
        handover={{ ...base, ...handover }}
        pending={false}
        canPrepare={false}
        canCapture={false}
        onPrepare={() => {}}
        onCapture={() => {}}
      />
    </I18nProvider>
  );
}

describe("HandoverPanel", () => {
  it("renders structured open incidents as text, not raw JSON", async () => {
    renderPanel();
    expect(screen.getByText("P-302 vibration")).toBeTruthy();
    expect(screen.queryByText(/\{.*asset_tag/s)).toBeNull();
  });

  it("renders the legacy string form via title", async () => {
    renderPanel({
      structured_content: {
        summary: "Handover summary",
        open_incidents: ["Legacy string incident"],
        active_executions: [], safety_concerns: [], follow_up: [],
        evidence_ids: [], unknowns: [], requires_human_review: false
      }
    });
    expect(screen.getByText("Legacy string incident")).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/components/HandoverPanel.test.tsx`
Expected: FAIL — type error (`open_incidents` still `string[]`) or rendered raw JSON.

- [ ] **Step 3: Implement**

In `web/src/types.ts`, update `ShiftHandover`:

```ts
export type HandoverListItem = {
  title: string;
  detail?: string;
  severity?: string;
};

// inside ShiftHandover.structured_content:
open_incidents: HandoverListItem[];
active_executions: HandoverListItem[];
safety_concerns: HandoverListItem[];
follow_up: HandoverListItem[];
unknowns: HandoverListItem[];
```

In `HandoverPanel.tsx`, change `HandoverSection` to accept `{ title: string; items: HandoverListItem[]; empty: string }` and render:

```tsx
<ul>
  {items.map((item, index) => (
    <li key={`${title}-${item.title}-${index}`}>
      {item.title}
      {item.detail ? <span className="handover-detail"> · {item.detail}</span> : null}
    </li>
  ))}
</ul>
```

Keep the `key` index-based (stable within one render; items are append-only lists from a single payload). No call sites pass `severity` for badge rendering yet — leave it on the type for a later enhancement.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npm test -- --run src/console/components/HandoverPanel.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/types.ts web/src/console/components/HandoverPanel.tsx web/src/console/components/HandoverPanel.test.tsx
git commit -m "fix: render handover incidents as structured items"
```

---

### Task 3: Incident severity filter (backend)

**Files:**
- Modify: `internal/incident/application/service.go:59-65` (Filter struct)
- Modify: `internal/incident/adapter/postgres/store.go:151-175` (List query)
- Modify: `internal/platform/httpserver/incidents.go:28-44` (listIncidents handler)
- Test: `internal/incident/adapter/postgres/store_integration_test.go` or service-level test

**Interfaces:**
- Produces: `Filter.Severity string` field; HTTP `GET /incidents?severity=HIGH` accepted; SQL predicate `AND (nullif($N,'') IS NULL OR i.severity = $N)`.
- Consumes: none (Task 6 consumes the HTTP param).

- [ ] **Step 1: Write the failing test**

Add a service-level test that passes a severity filter through to the store. Add to `internal/incident/application/service_test.go`:

```go
type severityRecorder struct{ got string }

func (r *severityRecorder) List(_ context.Context, _ identitydomain.Principal, filter Filter) ([]Incident, string, error) {
	r.got = filter.Severity
	return nil, "", nil
}

func TestListForwardsSeverityFilter(t *testing.T) {
	t.Parallel()
	recorder := &severityRecorder{}
	service := Service{Store: recorder}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionIncidentRead: {}},
	}
	_, _, err := service.List(context.Background(), principal, Filter{Severity: "HIGH"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recorder.got != "HIGH" {
		t.Fatalf("severity = %q, want HIGH", recorder.got)
	}
}
```

(If `Service.Store` is an interface already satisfied by a fake in that test file, reuse the existing fake and add the assertion there instead. Match the file's existing conventions — check how `Service{Store: ...}` is constructed in the existing tests first.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/incident/application/ -run TestListForwardsSeverityFilter -v`
Expected: FAIL — `Filter` has no `Severity` field (compile error).

- [ ] **Step 3: Implement**

In `service.go`, add to `Filter`:

```go
type Filter struct {
	SiteID   string
	AssetID  string
	State    string
	Severity string
	PageSize int
	Cursor   string
}
```

In `store.go` List, add the predicate and arg. Change the WHERE block and args:

```go
	query := incidentSelect + `
		WHERE i.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
		  AND (nullif($4, '') IS NULL OR i.asset_id = $4::uuid)
		  AND (nullif($5, '') IS NULL OR i.state = $5)
		  AND (nullif($6, '') IS NULL OR i.severity = $6)`
	args := []any{
		principal.OrganizationID, principal.SiteIDs,
		filter.SiteID, filter.AssetID, filter.State, filter.Severity,
	}
```

(If `$6` collides with existing cursor args, renumber — the `fmt.Sprintf` cursor args are appended after, so `$6` is safe here.)

In `httpserver/incidents.go`, parse the param:

```go
		filter := incidentapp.Filter{
			SiteID:   r.URL.Query().Get("site_id"),
			AssetID:  r.URL.Query().Get("asset_id"),
			State:    r.URL.Query().Get("state"),
			Severity: r.URL.Query().Get("severity"),
			PageSize: pageSize,
			Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/incident/...`
Expected: PASS. Then run the handler test: `go test ./internal/platform/httpserver/ -run TestIncidents -v` (extend `incidents_list_test.go` with a severity case if the file has an established table pattern — follow it).

- [ ] **Step 5: Commit**

```bash
git add internal/incident/application/service.go internal/incident/adapter/postgres/store.go internal/platform/httpserver/incidents.go
git commit -m "feat: filter incidents by severity server-side"
```

---

### Task 4: `GET /summary` endpoint (backend)

**Files:**
- Create: `internal/platform/httpserver/summary.go`
- Modify: `internal/platform/httpserver/router.go:98-104` (mount call)
- Test: `internal/platform/httpserver/summary_test.go` (create)

**Interfaces:**
- Produces: `GET /api/v1/summary?site_id=` → `{ "open_incidents": int, "in_progress_incidents": int, "resolved_incidents": int, "total_incidents": int, "by_severity": { "LOW": int, "MEDIUM": int, "HIGH": int, "CRITICAL": int }, "active_executions": int, "critical_assets": int, "pending_handovers": int }`.
- Consumes: `Dependencies.Database *pgxpool.Pool` (existing field, router.go:46).

- [ ] **Step 1: Write the failing test**

Create `internal/platform/httpserver/summary_test.go` using the existing handler-test conventions (check `incidents_list_test.go` for how the router is built in tests — mock the pool or use the `New(Dependencies{...})` constructor with a stub database):

```go
func TestSummaryRequiresAuthentication(t *testing.T) {
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
```

For the happy path, follow how `incidents_list_test.go` constructs a handler with a fake service. If the test harness uses real `*pgxpool.Pool`, this test needs the integration pool — match whatever `reports_list_test.go`/`incidents_list_test.go` do and add a case asserting the summary shape: keys present, counts are ints, `by_severity` has all four keys. Copy the exact harness from the closest existing list test.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestSummary -v`
Expected: FAIL — route not found (404/405) or auth failure because route doesn't exist.

- [ ] **Step 3: Implement**

Create `internal/platform/httpserver/summary.go`:

```go
package httpserver

import (
	"net/http"
	"strings"
)

func mountSummaryRoutes(router chi.Router, pool *pgxpool.Pool) {
	router.Get("/summary", summary(pool))
}

func summary(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
		if siteID != "" && !validUUIDParam(w, siteID, "site ID") {
			return
		}

		row := pool.QueryRow(r.Context(), `
			SELECT
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'OPEN'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'IN_PROGRESS'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'RESOLVED'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR i.site_id = $3::uuid)),
				(SELECT count(*) FROM maintenance_executions e
				  WHERE e.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR e.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR e.site_id = $3::uuid)
				    AND e.state = 'IN_PROGRESS'),
				(SELECT count(*) FROM asset_criticalities ac
				  WHERE ac.organization_id = $1::uuid
				    AND ac.superseded_at IS NULL
				    AND ac.rating = 'A'
				    AND EXISTS (
				      SELECT 1 FROM assets a
				      WHERE a.id = ac.asset_id
				        AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR a.site_id = ANY($2::uuid[]))
				        AND ($3::uuid IS NULL OR a.site_id = $3::uuid))),
				(SELECT count(*) FROM shift_handovers h
				  WHERE h.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR h.site_id = ANY($2::uuid[]))
				    AND ($3::uuid IS NULL OR h.site_id = $3::uuid)
				    AND h.state IN ('DRAFT', 'SUBMITTED'))`,
			principal.OrganizationID, principal.SiteIDs, nullableUUID(siteID),
		)

		var open, inProgress, resolved, total, activeExecutions, criticalAssets, pendingHandovers int
		if err := row.Scan(&open, &inProgress, &resolved, &total,
			&activeExecutions, &criticalAssets, &pendingHandovers); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Summary query failed",
			})
			return
		}

		severityCounts := map[string]int{}
		rows, err := pool.Query(r.Context(), `
			SELECT i.severity, count(*)
			FROM incidents i
			WHERE i.organization_id = $1::uuid
			  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
			  AND ($3::uuid IS NULL OR i.site_id = $3::uuid)
			GROUP BY i.severity`,
			principal.OrganizationID, principal.SiteIDs, nullableUUID(siteID),
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Summary query failed",
			})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var severity string
			var count int
			if err := rows.Scan(&severity, &count); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"status": http.StatusInternalServerError, "title": "Summary query failed",
				})
				return
			}
			severityCounts[severity] = count
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"open_incidents":       open,
			"in_progress_incidents": inProgress,
			"resolved_incidents":   resolved,
			"total_incidents":      total,
			"by_severity": map[string]int{
				"LOW":      severityCounts["LOW"],
				"MEDIUM":   severityCounts["MEDIUM"],
				"HIGH":     severityCounts["HIGH"],
				"CRITICAL": severityCounts["CRITICAL"],
			},
			"active_executions":  activeExecutions,
			"critical_assets":    criticalAssets,
			"pending_handovers":  pendingHandovers,
		})
	}
}
```

**Verify column names before committing**: the schema (verified in `migrations/00002_core_maintenance.sql`) has `assets` WITHOUT a criticality column — criticality lives in `asset_criticalities` (rating A/B/C, current row = `superseded_at IS NULL`, unique partial index `asset_criticalities_current_idx`). `maintenance_executions`, `maintenance_reports`, `incidents`, and `shift_handovers` (state DRAFT/SUBMITTED/ACCEPTED/ACKNOWLEDGED) are confirmed table names. Adjust the SQL to match the real schema exactly — the critical-assets subquery above already handles the separate table; double-check `organization_id` exists on `asset_criticalities` (it does, per migration) and that `incidents`/`maintenance_executions` have the state values used. Also check whether `nullableUUID` helper exists in `maintenance_helpers.go` — if not, use the same pattern `httpserver` uses elsewhere for optional UUID params (e.g. pass an empty string and rely on `$3::uuid IS NULL` semantics or a `stringToUUID` helper). Match existing helper names.

In `router.go`, register the route after the other mounts:

```go
		mountSummaryRoutes(api, dependencies.Database)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ -run TestSummary -v`
Expected: PASS. Also `go build ./...` to catch helper-name issues.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/httpserver/summary.go internal/platform/httpserver/summary_test.go internal/platform/httpserver/router.go
git commit -m "feat: add dashboard summary counts endpoint"
```

---

### Task 5: Dashboard consumes `/summary` and caps its lists

**Files:**
- Modify: `web/src/api.ts` (add `summary` method)
- Modify: `web/src/console/pages/DashboardPage.tsx`
- Modify: `web/src/console/components/Overview.tsx:23-47` (props)
- Modify: `web/src/console/pages/DashboardPage.test.tsx`
- Modify: `web/src/types.ts` (DashboardSummary)

**Interfaces:**
- Consumes: `/summary` payload shape from Task 4; existing `useQuery` hook.
- Produces: `api.summary(siteID?: string): Promise<DashboardSummary>`; `Overview` prop `summary?: DashboardSummary` used for the four metric cards, replacing aggregation from arrays.

- [ ] **Step 1: Write the failing test**

Add to `web/src/console/pages/DashboardPage.test.tsx` a case asserting metric cards use the summary endpoint:

```tsx
it("renders metric cards from the summary endpoint", async () => {
  (api.summary as ReturnType<typeof vi.fn>).mockResolvedValue({
    open_incidents: 3, in_progress_incidents: 2, resolved_incidents: 9,
    total_incidents: 14,
    by_severity: { LOW: 4, MEDIUM: 5, HIGH: 3, CRITICAL: 2 },
    active_executions: 2, critical_assets: 1, pending_handovers: 1
  });
  renderPage();
  expect(await screen.findByText("3")).toBeTruthy();
  expect(screen.getByText("2")).toBeTruthy();
});
```

(Extend the existing `vi.mock("../../api")` block with `summary: vi.fn()`. Match the existing renderPage wrapper.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/DashboardPage.test.tsx`
Expected: FAIL — `api.summary` is undefined.

- [ ] **Step 3: Implement**

In `web/src/types.ts`:

```ts
export type DashboardSummary = {
  open_incidents: number;
  in_progress_incidents: number;
  resolved_incidents: number;
  total_incidents: number;
  by_severity: { LOW: number; MEDIUM: number; HIGH: number; CRITICAL: number };
  active_executions: number;
  critical_assets: number;
  pending_handovers: number;
};
```

In `web/src/api.ts`, add to the `api` object:

```ts
  summary: (siteID?: string) =>
    request<DashboardSummary>(`/summary${siteID ? `?site_id=${siteID}` : ""}`),
```

(Add `DashboardSummary` to the type import at the top of `api.ts`.)

In `DashboardPage.tsx`, fetch summary instead of four unbounded lists; keep capped lists only for the queue + table:

```tsx
export function DashboardPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids?.[0];
  const summary = useQuery(() => api.summary(siteID), [siteID]);
  const incidents = useQuery(() =>
    api.incidents({ site_id: siteID, page_size: 25 }).then((list) => list.items),
    [siteID],
  );
  const executions = useQuery(() =>
    api.listExecutions({ site_id: siteID, page_size: 25 }).then((list) => list.items),
    [siteID],
  );
  const handovers = useQuery(() => api.pendingHandovers(), []);

  const loading = summary.loading || incidents.loading || executions.loading || handovers.loading;
  const error = summary.error ?? incidents.error ?? executions.error ?? handovers.error;
  const onRetry = () => {
    void summary.refetch();
    void incidents.refetch();
    void executions.refetch();
    void handovers.refetch();
  };

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.overview")} principal={principal} />
        <Overview
          focus={FOCUS_CONFIG[focusRole(principal)]}
          summary={summary.data}
          incidents={incidents.data ?? []}
          executions={executions.data ?? []}
          pendingHandoverCount={handovers.data?.length ?? 0}
          loading={loading}
          error={error}
          onRetry={onRetry}
        />
      </section>
    </PageTrailProvider>
  );
}
```

(Remove the `assets` query and the `assets` prop; the critical-assets count now comes from `summary.critical_assets`.)

In `Overview.tsx`, change the props: remove `assets`, add `summary?: DashboardSummary`, and drive the metric cards from it:

```tsx
export function Overview({
  focus,
  summary,
  incidents,
  executions,
  pendingHandoverCount,
  loading,
  error,
  onRetry,
}: {
  focus: FocusConfig;
  summary?: DashboardSummary;
  incidents: Incident[];
  executions: Execution[];
  pendingHandoverCount: number;
  loading: boolean;
  error?: string;
  onRetry: () => void;
}) {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const open = incidents.filter((incident) => incident.state !== "RESOLVED");
  const inProgress = executions.filter((execution) => execution.state === "IN_PROGRESS");
```

Metric cards:

```tsx
        <MetricCard
          label={t("dashboard.openIncidents")}
          value={String(summary?.open_incidents ?? open.length)}
          to="/incidents"
          tone="critical"
        />
        <MetricCard
          label={t("dashboard.executionsInProgress")}
          value={String(summary?.active_executions ?? inProgress.length)}
          to="/executions"
          tone="medium"
        />
        <MetricCard
          label={t("dashboard.criticalAssets")}
          value={String(summary?.critical_assets ?? 0)}
          to="/assets"
          tone="high"
        />
        <MetricCard
          label={t("dashboard.pendingHandovers")}
          value={String(pendingHandoverCount)}
          to="/handovers"
          tone="info"
        />
```

Keep the queue/table sections reading from the capped `incidents`/`executions` arrays. Remove the `critical` asset computation.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npm test -- --run src/console/pages/DashboardPage.test.tsx`
Expected: PASS. Run `npm test -- --run src/console` for regressions (Overview is used by DashboardPage only).

- [ ] **Step 5: Commit**

```bash
git add web/src/types.ts web/src/api.ts web/src/console/pages/DashboardPage.tsx web/src/console/components/Overview.tsx web/src/console/pages/DashboardPage.test.tsx
git commit -m "feat: drive dashboard metrics from summary endpoint"
```

---

### Task 6: IncidentsPage uses server-side filters + summary counts

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx`
- Modify: `web/src/console/pages/IncidentsPage.test.tsx`

**Interfaces:**
- Consumes: `api.incidents({ state, severity, page_size, cursor })` (severity now supported by Task 3); `api.summary(siteID)` (Task 4).
- Produces: tabs + severity filter refetch from the server; tab counts and severity totals come from `summary.by_severity` and state counts.

- [ ] **Step 1: Write the failing test**

Add to `IncidentsPage.test.tsx` (extend the existing `vi.mock` with `summary`):

```tsx
it("refetches with the selected severity", async () => {
  (api.summary as ReturnType<typeof vi.fn>).mockResolvedValue({
    open_incidents: 1, in_progress_incidents: 1, resolved_incidents: 1,
    total_incidents: 3,
    by_severity: { LOW: 1, MEDIUM: 1, HIGH: 1, CRITICAL: 0 },
    active_executions: 0, critical_assets: 0, pending_handovers: 0
  });
  renderPage();
  await screen.findByText("Pump vibration");
  fireEvent.change(screen.getByLabelText(/severity/i), { target: { value: "HIGH" } });
  await waitFor(() => {
    const calls = (api.incidents as ReturnType<typeof vi.fn>).mock.calls;
    expect(calls.some(([opts]) => opts && opts.severity === "HIGH")).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: FAIL — `api.incidents` never receives `severity` (client-side filter still in place).

- [ ] **Step 3: Implement**

Rewrite the data flow in `IncidentsPage.tsx`:

```tsx
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids?.[0];
  const incidents = usePaginatedList(
    (params) => api.incidents({ ...params, site_id: siteID, state: [tab], severity }),
    [tab, severity, siteID],
  );
  const summary = useQuery(() => api.summary(siteID), [siteID]);
```

Remove the client-side `rows` filter for state/severity (keep only the search `query` filter, applied to the loaded page, with the label hint from messages):

```tsx
  const rows = useMemo(() => {
    const items = incidents.items;
    if (!query.trim()) return items;
    const q = query.trim().toLowerCase();
    return items.filter(
      (incident) =>
        incident.number.toLowerCase().includes(q) ||
        incident.summary.toLowerCase().includes(q) ||
        (incident.asset_tag ?? "").toLowerCase().includes(q),
    );
  }, [incidents.items, query]);
```

Tab counts from summary (truthful regardless of page):

```tsx
  const counts = {
    OPEN: summary.data?.open_incidents ?? 0,
    IN_PROGRESS: summary.data?.in_progress_incidents ?? 0,
    RESOLVED: summary.data?.resolved_incidents ?? 0,
    ALL: summary.data?.total_incidents ?? 0,
  } as Record<StateTab, number>;
```

Note: because the tab now drives a server refetch (`state` is a dependency of the hook), clicking a tab shows the loading state while data reloads. Keep `incidents.loading` on the DataTable as-is. Add a small helper text near the search box: use a new message key `incidents.filter.searchHint` = en "Searches loaded results" / vi "Tìm trong kết quả đã tải" (see Task 10 for the two-language message requirement; add the key now).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: PASS — update existing assertions that assumed client-side filtering if they relied on all fixtures being loaded at once (the mock `incidents` still returns all three fixtures on the first page, so tab assertions should still hold; the `by_severity` fixture must include the states used).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/pages/IncidentsPage.tsx web/src/console/pages/IncidentsPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat: server-filter incidents with truthful tab counts"
```

---

### Task 7: Report list exposes identity columns (backend)

**Files:**
- Modify: `internal/report/application/service.go:41-64` (Report struct)
- Modify: `internal/report/adapter/postgres/store.go:277-311` (List select + scan)
- Test: `internal/report/adapter/postgres/store_integration_test.go` (extend)

**Interfaces:**
- Produces: `Report.CreatedAt`/`Report.UpdatedAt` (already in struct — verify the list scan populates them; the List select already selects `created_at, updated_at` but the scan may skip them), plus `Report.AssetTag string \`json:"asset_tag,omitempty"\`` and `Report.IncidentNumber string \`json:"incident_number,omitempty"\`` from a LEFT JOIN.
- Consumes: none (Task 8 consumes the JSON).

- [ ] **Step 1: Verify current scan then write the failing test**

First read the List scan in `store.go` (after the select at :311) to confirm `created_at`/`updated_at` are scanned. Then add an integration test asserting the new fields: create a report whose execution links an incident, list reports, and assert `IncidentNumber` and `AssetTag` are populated.

```go
// in store_integration_test.go, following the existing test harness
func TestListReportsExposesIncidentNumberAndAssetTag(t *testing.T) {
	// reuse the harness: create asset -> incident -> execution -> report
	// then List and assert the report carries the incident number and asset tag
}
```

(Match the exact harness setup of the existing report store integration tests — they already create assets/incidents/executions.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/report/adapter/postgres/ -run TestListReportsExposesIncidentNumberAndAssetTag -v`
Expected: FAIL — fields are empty (`""`).

- [ ] **Step 3: Implement**

In `service.go`, add fields to `Report`:

```go
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AssetTag       string    `json:"asset_tag,omitempty"`
	IncidentNumber string    `json:"incident_number,omitempty"`
```

In `store.go` List, extend the select with the joins and columns (keep the existing `WHERE` on `maintenance_reports`; LEFT JOIN so reports without incidents still list):

```go
	query := `
		SELECT r.id::text, r.organization_id::text, r.site_id::text, r.execution_id::text,
		       r.revision, r.version, r.state, r.structured_content, r.evidence_snapshot,
		       coalesce(r.provider, ''), coalesce(r.model, ''),
		       coalesce(r.model_version, ''), coalesce(r.prompt_version, ''),
		       coalesce(r.input_sha256, ''), coalesce(r.output_sha256, ''),
		       r.generated_by_kind, coalesce(r.submitted_by::text, ''), r.submitted_at,
		       coalesce(r.approved_by::text, ''), r.approved_at, r.created_at, r.updated_at,
		       coalesce(a.tag, ''), coalesce(i.number, '')
		FROM maintenance_reports r
		LEFT JOIN maintenance_executions e ON e.id = r.execution_id
		LEFT JOIN assets a ON a.id = e.asset_id
		LEFT JOIN incidents i ON i.id = e.incident_id
		WHERE r.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR r.site_id = ANY($2::uuid[]))`
```

Update the row scan to append `&value.AssetTag, &value.IncidentNumber` (and `created_at`/`updated_at` if not already scanned). Verify the cursor predicate and ORDER BY still reference `r.created_at, r.id` (now unambiguous with joins) — update `created_at` → `r.created_at` in the cursor `fmt.Sprintf` if the joins make it ambiguous.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/report/...`
Expected: PASS. Also run `go test ./internal/platform/httpserver/ -run TestReports -v`.

- [ ] **Step 5: Commit**

```bash
git add internal/report/application/service.go internal/report/adapter/postgres/store.go internal/report/adapter/postgres/store_integration_test.go
git commit -m "feat: expose asset and incident identity on report list"
```

---

### Task 8: ReportsPage shows distinguishing columns

**Files:**
- Modify: `web/src/types.ts:163-183` (MaintenanceReport)
- Modify: `web/src/console/pages/ReportsPage.tsx`
- Modify: `web/src/console/pages/ReportsPage.test.tsx`

**Interfaces:**
- Consumes: `report.created_at`, `report.updated_at`, `report.asset_tag`, `report.incident_number` from Task 7.
- Produces: columns "Số hiệu" (incident link), "Tài sản", "Cập nhật" (RelativeTime on updated_at); `RP-{revision}` demoted to a secondary chip.

- [ ] **Step 1: Write the failing test**

Add to `ReportsPage.test.tsx` (extend fixtures with the new fields):

```tsx
it("shows incident number and updated time for each report", async () => {
  (api.reports as ReturnType<typeof vi.fn>).mockResolvedValue({
    items: [{
      id: "r1", execution_id: "e1", revision: 1, version: 1,
      state: "APPROVED",
      structured_content: { summary: "Diagnose high vibration on P-302", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false },
      evidence: [],
      asset_tag: "P-302",
      incident_number: "IN-1042",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    }],
    next_cursor: null, has_more: false
  });
  renderPage();
  expect(await screen.findByText("IN-1042")).toBeTruthy();
  expect(screen.getByText("P-302")).toBeTruthy();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/ReportsPage.test.tsx`
Expected: FAIL — `IN-1042` not rendered.

- [ ] **Step 3: Implement**

In `types.ts`, add to `MaintenanceReport`:

```ts
  asset_tag?: string;
  incident_number?: string;
  created_at: string;
  updated_at: string;
```

In `ReportsPage.tsx`, replace the `workOrder` column with an identity column and add the new columns:

```tsx
            {
              key: "incident",
              header: t("report.workOrder"),
              render: (report) =>
                report.incident_number ? (
                  <Link
                    to={`/incidents/${report.execution_id}`}
                    className="mono strong"
                    onClick={(event) => event.stopPropagation()}
                  >
                    {report.incident_number}
                  </Link>
                ) : (
                  <span className="mono muted">{report.execution_id.slice(0, 8)}</span>
                ),
              sortValue: (r) => r.incident_number ?? ""
            },
            {
              key: "asset",
              header: t("dashboard.table.asset"),
              render: (report) => report.asset_tag ?? "—"
            },
            {
              key: "updated",
              header: t("report.updated"),
              render: (report) => <RelativeTime time={report.updated_at} locale={locale} />,
              sortValue: (r) => r.updated_at
            },
```

(The incident link goes to the incident via the execution's incident; if `incident_number` is present but we only have `execution_id`, link to `/executions/{execution_id}` instead — choose the most truthful target; if the report links an incident, the execution detail page links onward.)

Add message keys: `report.updated` = en "Updated" / vi "Cập nhật" (see Task 10 pattern). Import `RelativeTime` in ReportsPage.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npm test -- --run src/console/pages/ReportsPage.test.tsx`
Expected: PASS. Fix any fixture-type errors from the new required `created_at`/`updated_at` fields (make them required in the type and update fixtures).

- [ ] **Step 5: Commit**

```bash
git add web/src/types.ts web/src/console/pages/ReportsPage.tsx web/src/console/pages/ReportsPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat: distinguish reports by incident, asset, and update time"
```

---

### Task 9: Knowledge page groups revisions with an effective marker

**Files:**
- Modify: `web/src/console/pages/KnowledgePage.tsx`
- Modify: `web/src/console/components/KnowledgePanel.tsx`
- Modify: `web/src/console/pages/KnowledgePage.test.tsx`

**Interfaces:**
- Consumes: existing `KnowledgeDocument.revisions` (`approval_status`, `revision`).
- Produces: row per document with expandable revisions; "Đang áp dụng" marker on the latest APPROVED revision (decision: effective = latest APPROVED; if none approved, the latest revision shows "Bản nháp").

- [ ] **Step 1: Write the failing test**

Add to `KnowledgePage.test.tsx`:

```tsx
it("marks the latest approved revision as effective", async () => {
  (api.documents as ReturnType<typeof vi.fn>).mockResolvedValue({
    items: [{
      id: "d1", site_id: "s1", document_type: "SOP",
      title: "P-302 Bearing Lubrication SOP", authority: "SITE_APPROVED",
      revisions: [
        { id: "r1", document_id: "d1", revision: "R1", approval_status: "APPROVED", ingestion_state: "READY", language: "en", version: 1, applicability: [] },
        { id: "r2", document_id: "d1", revision: "R2", approval_status: "DRAFT", ingestion_state: "READY", language: "en", version: 1, applicability: [] }
      ]
    }],
    next_cursor: null, has_more: false
  });
  renderPage();
  expect(await screen.findByText(/đang áp dụng/i)).toBeTruthy();
});
```

(Match the exact rendering text to the message key you choose.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/KnowledgePage.test.tsx`
Expected: FAIL — no effective marker rendered.

- [ ] **Step 3: Implement**

In `KnowledgePanel.tsx`, replace the flat "revision" column with a document-level column showing revision count and an effective badge, and render revisions as an expandable row (use `<details>` like EvidenceLinks, or a simple two-row layout). Compute effective revision:

```tsx
function effectiveRevision(document: KnowledgeDocument) {
  const approved = document.revisions.filter((r) => r.approval_status === "APPROVED");
  if (approved.length > 0) return approved[approved.length - 1];
  return document.revisions[document.revisions.length - 1] ?? undefined;
}
```

Render per document: title link, document_type, authority badge (existing), and a revisions summary — `R1 · R2` chips with `StatusBadge` per revision plus a `source-badge` "Đang áp dụng" on the effective one. Add message key `knowledge.effective` = en "Effective" / vi "Đang áp dụng". Keep the `DataTable` (rows stay documents); the revision chips live inside the revision column cell.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npm test -- --run src/console/pages/KnowledgePage.test.tsx`
Expected: PASS. Existing assertions that matched the old flat row format must be updated.

- [ ] **Step 5: Commit**

```bash
git add web/src/console/pages/KnowledgePage.tsx web/src/console/components/KnowledgePanel.tsx web/src/console/pages/KnowledgePage.test.tsx web/src/i18n/messages.ts
git commit -m "feat: group knowledge revisions with effective marker"
```

---

## Phase 2 — Scan & labels

### Task 10: Execution and handover state labels are Vietnamese

**Files:**
- Modify: `web/src/console/labels.ts`
- Modify: `web/src/i18n/messages.ts`
- Modify: `web/src/console/pages/ExecutionsPage.tsx:36-47,81-88`
- Modify: `web/src/console/pages/HandoverPage.tsx` (history badges)
- Modify: `web/src/console/components/Overview.tsx:136-142` (my-queue state badge)
- Test: `web/src/console/pages/ExecutionsPage.test.tsx` (extend)

**Interfaces:**
- Produces: `executionStateLabelKey(state): MessageKey` + `executionStateTone(state): Tone`; `handoverStateLabelKey(state): MessageKey`; message keys `execution.state.assigned/inProgress/completed`, `handover.state.*` (existing keys at messages.ts:340-343 can be reused for handover badges).

- [ ] **Step 1: Write the failing test**

Add to `ExecutionsPage.test.tsx`:

```tsx
it("renders execution states in Vietnamese", async () => {
  (api.listExecutions as ReturnType<typeof vi.fn>).mockResolvedValue({
    items: [{
      id: "e1", incident_id: "i1", asset_id: "a1", asset_tag: "P-302",
      purpose: "High vibration pump inspection", state: "IN_PROGRESS",
      version: 1, steps: [], measurements: [], observations: [], actions: []
    }],
    next_cursor: null, has_more: false
  });
  renderPage();
  expect(await screen.findByText(/đang xử lý/i)).toBeTruthy();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/ExecutionsPage.test.tsx`
Expected: FAIL — renders "IN PROGRESS".

- [ ] **Step 3: Implement**

In `labels.ts`:

```ts
export function executionStateLabelKey(state: string): MessageKey {
  switch (state) {
    case "ASSIGNED": return "execution.state.assigned";
    case "IN_PROGRESS": return "execution.state.inProgress";
    case "COMPLETED": return "execution.state.completed";
    default: return "execution.state.unknown";
  }
}

export function executionStateTone(state: string): Tone {
  switch (state) {
    case "ASSIGNED": return "info";
    case "IN_PROGRESS": return "medium";
    case "COMPLETED": return "success";
    default: return "info";
  }
}
```

In `messages.ts`, add to `en`:

```ts
  "execution.state.assigned": "Assigned",
  "execution.state.inProgress": "In progress",
  "execution.state.completed": "Completed",
  "execution.state.unknown": "Unknown",
```

and to `vi`:

```ts
  "execution.state.assigned": "Đã phân công",
  "execution.state.inProgress": "Đang xử lý",
  "execution.state.completed": "Đã hoàn thành",
  "execution.state.unknown": "Không xác định",
```

(Add them in the same relative position as the other `execution.*` keys in each object so the diff stays readable.)

In `ExecutionsPage.tsx`: replace `state.replace("_", " ")` in the tab button and the StatusBadge label with `t(executionStateLabelKey(state))`, and replace the `EXECUTION_TONE` map usage with `executionStateTone(execution.state)`. Remove the now-unused `EXECUTION_TONE` constant and `Tone` import if unused.

In `HandoverPage.tsx`: find where history rows render `handover.state` badges and use `handoverStateTone` (exists) + the existing `handover.state.*` message keys via `STATE_LABEL`-style mapping (mirror HandoverPanel's `STATE_LABEL` map).

In `Overview.tsx` my-queue badge, replace `execution.state.replace("_", " ")` with `t(executionStateLabelKey(execution.state))` and use `executionStateTone`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npm test -- --run src/console/pages/ExecutionsPage.test.tsx`
Expected: PASS. Run `npm test -- --run src/console` for regressions.

- [ ] **Step 5: Commit**

```bash
git add web/src/console/labels.ts web/src/i18n/messages.ts web/src/console/pages/ExecutionsPage.tsx web/src/console/pages/HandoverPage.tsx web/src/console/components/Overview.tsx
git commit -m "i18n: render execution and handover states in Vietnamese"
```

---

### Task 11: ExecutionsPage filters server-side and shows incident number

**Files:**
- Modify: `web/src/console/pages/ExecutionsPage.tsx`
- Modify: `web/src/console/pages/ExecutionsPage.test.tsx`
- Backend: `internal/execution/adapter/postgres/query.go:39-46` (add incident number join) + `internal/execution/application/service.go` (Execution struct) + tests

**Interfaces:**
- Consumes: `executionapp.Filter.State` (exists, service.go:108); new `Execution.IncidentNumber` field.
- Produces: `api.listExecutions({ state })` refetch on tab change; SỰ CỐ column renders incident number link; TÀI SẢN column renders asset tag.

- [ ] **Step 1: Backend test first**

Add to `internal/execution/adapter/postgres/` integration test (match existing harness):

```go
func TestListExecutionsExposesIncidentNumber(t *testing.T) {
	// create asset -> incident -> execution, then List and assert IncidentNumber populated
}
```

Run: `go test ./internal/execution/adapter/postgres/ -run TestListExecutionsExposesIncidentNumber -v` — expect FAIL (field empty).

- [ ] **Step 2: Backend implementation**

In `internal/execution/application/service.go`, add to the `Execution` struct:

```go
	IncidentNumber string `json:"incident_number,omitempty"`
```

In `internal/execution/adapter/postgres/query.go`, the List query selects only `e.id::text` then hydrates rows via the load function (`loadExecution`-style, which joins `assets a` at :112). Add the incident join there:

```go
		LEFT JOIN incidents i ON i.id = e.incident_id
```

and select `coalesce(i.number, '')` into `&value.IncidentNumber` in the same Scan.

Run: `go test ./internal/execution/...` — PASS.

- [ ] **Step 3: Frontend test**

Add to `ExecutionsPage.test.tsx`:

```tsx
it("shows the incident number in the incident column", async () => {
  (api.listExecutions as ReturnType<typeof vi.fn>).mockResolvedValue({
    items: [{
      id: "e1", incident_id: "i1", asset_id: "a1", asset_tag: "P-302",
      incident_number: "IN-1042", purpose: "High vibration pump inspection",
      state: "IN_PROGRESS", version: 1, steps: [], measurements: [],
      observations: [], actions: []
    }],
    next_cursor: null, has_more: false
  });
  renderPage();
  expect(await screen.findByText("IN-1042")).toBeTruthy();
});
```

Run: `cd web && npm test -- --run src/console/pages/ExecutionsPage.test.tsx` — expect FAIL.

- [ ] **Step 4: Frontend implementation**

In `web/src/types.ts`, add `incident_number?: string` to `Execution`.

In `ExecutionsPage.tsx`:

```tsx
  const executions = usePaginatedList(
    (params) => api.listExecutions({ ...params, state: filter === "ALL" ? undefined : filter }),
    [filter],
  );
```

Remove the client-side `rows` filter (use `executions.items` directly) and change the SỰ CỐ column render:

```tsx
              render: (execution) =>
                execution.incident_id ? (
                  <Link to={`/incidents/${execution.incident_id}`}>
                    {execution.incident_number ?? execution.asset_tag ?? execution.incident_id}
                  </Link>
                ) : (
                  <span className="muted">—</span>
                ),
```

(Check `api.listExecutions` accepts `state` as a string in `ListOptions.state: string[]` — pass `state: filter === "ALL" ? undefined : [filter]` to match the `listQuery` append loop.)

- [ ] **Step 5: Run tests and commit**

Run: `cd web && npm test -- --run src/console/pages/ExecutionsPage.test.tsx` — PASS; `go test ./internal/execution/...` — PASS.

```bash
git add internal/execution/application/service.go internal/execution/adapter/postgres/query.go web/src/types.ts web/src/console/pages/ExecutionsPage.tsx web/src/console/pages/ExecutionsPage.test.tsx
git commit -m "feat: server-filter executions and show incident numbers"
```

---

### Task 12: Distinct severity badge tones

**Files:**
- Modify: `web/src/console/ui/StatusBadge.tsx` (or the CSS it references)
- Modify: `web/src/console/labels.ts` (verify `severityTone` mapping distinct)
- Test: `web/src/console/ui/primitives.test.tsx` or `StatusBadge` test

**Interfaces:**
- Consumes: `severityTone` (labels.ts:20).
- Produces: visually distinct treatments for `low`/`medium`/`high`/`critical` severity badges (e.g., colored text/dot variants), verified by a test asserting the tone classes differ.

- [ ] **Step 1: Write the failing test**

Add to `web/src/console/ui/primitives.test.tsx` (or a new `StatusBadge.test.tsx`):

```tsx
it("renders distinct classes for each severity tone", () => {
  const { container, rerender } = render(
    <I18nProvider><StatusBadge tone="low" label="Low" /></I18nProvider>,
  );
  const classes = new Set<string>();
  for (const tone of ["low", "medium", "high", "critical"] as const) {
    rerender(<I18nProvider><StatusBadge tone={tone} label={tone} /></I18nProvider>);
    classes.add(container.querySelector("[class*='badge']")?.className ?? "");
  }
  expect(classes.size).toBe(4);
});
```

Run: `cd web && npm test -- --run src/console/ui/primitives.test.tsx` — expect FAIL if tones collapse to the same class.

- [ ] **Step 2: Implement**

Inspect `StatusBadge.tsx` + the CSS file that styles `.badge` tones. Add distinct treatment per tone (e.g., `--tone-low` outline + muted dot, `--tone-medium` amber, `--tone-high` orange/red, `--tone-critical` solid red) so severity is scannable. If the CSS already has four distinct classes, the failing test will reveal which two collapse; fix only the collapsed pair.

- [ ] **Step 3: Run test and commit**

Run: `cd web && npm test -- --run src/console/ui/primitives.test.tsx` — PASS.

```bash
git add web/src/console/ui/StatusBadge.tsx web/src/console/labels.ts web/src/console/ui/primitives.test.tsx
git commit -m "fix: make severity badge tones visually distinct"
```

---

## Phase 3 — Guidance & polish

### Task 13: Search page affordances

**Files:**
- Modify: `web/src/console/pages/SearchPage.tsx`
- Modify: `web/src/console/pages/SearchPage.test.tsx`

**Interfaces:**
- Consumes: existing search api + `useQuery`.
- Produces: autofocus input, Enter-to-search, scope selector (asset/incident/document), recent queries (localStorage), no-results explanation.

- [ ] **Step 1: Write the failing test**

```tsx
it("autofocuses the search input", async () => {
  renderPage();
  expect(screen.getByLabelText(/search/i)).toHaveFocus();
});
```

- [ ] **Step 2: Run to verify failure** — `cd web && npm test -- --run src/console/pages/SearchPage.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — add `autoFocus` to the input; scope segmented control with `useState<Scope>("all")`; persist recent queries in `localStorage` under `skawld.search.recent` (max 5, deduped); render them under the input before any search; when results are empty and a query was run, show `EmptyState` with the query echoed instead of a blank line.

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/pages/SearchPage.tsx web/src/console/pages/SearchPage.test.tsx
git commit -m "feat: add search affordances and recent queries"
```

---

### Task 14: Dashboard empty states and deep links

**Files:**
- Modify: `web/src/console/components/Overview.tsx:86-150`
- Modify: `web/src/i18n/messages.ts`
- Modify: `web/src/console/pages/IncidentsPage.tsx` (support `?state=` query param)

**Interfaces:**
- Consumes: `EmptyState` component.
- Produces: my-queue empty state with a primary CTA ("Xem tất cả quy trình" → `/executions`); "Sức khỏe tổ chức" chips become links (quality → `/quality`, incidents → `/incidents?state=OPEN`, assets → `/assets`); IncidentsPage reads `?state=` on mount.

- [ ] **Step 1: Write the failing test** (extend `DashboardPage.test.tsx`)

```tsx
it("offers a call to action in the empty queue state", async () => {
  (api.summary as ReturnType<typeof vi.fn>).mockResolvedValue({ ...emptySummary });
  (api.listExecutions as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [], next_cursor: null, has_more: false });
  renderPage();
  expect(await screen.findByRole("link", { name: /view all/i })).toBeTruthy();
});
```

- [ ] **Step 2: Run to verify failure** — expect FAIL.

- [ ] **Step 3: Implement**

In `Overview.tsx`, the my-queue empty branch (currently `EmptyState title={...}` at :107) becomes:

```tsx
        {inProgress.length === 0 ? (
          <EmptyState
            title={t("dashboard.myQueueEmpty")}
            action={{ label: t("dashboard.viewAll"), to: "/executions" }}
          />
        ) : (
```

Check `EmptyState` props — if it has no `action` prop, extend it with an optional `action?: { label: string; to: string }` rendering a `<Link className="primary-button">`. The "Sức khỏe tổ chức" section (the `focus-links` block at :90-96) already renders links — make each link's `to` deep-link (`/incidents?state=OPEN` for the incident chip; verify `focus.links` label/to come from `dashboardFocus.ts`, updating there if the "to" is generic).

In `IncidentsPage.tsx`, on mount read `state` from search params and set the tab:

```tsx
  useEffect(() => {
    const fromParams = params.get("state");
    if (fromParams && STATE_TABS.includes(fromParams as StateTab)) {
      setTab(fromParams as StateTab);
    }
  }, [params]);
```

(Replace the existing `create` param effect or extend it.)

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/components/Overview.tsx web/src/console/ui/EmptyState.tsx web/src/console/pages/IncidentsPage.tsx web/src/console/pages/DashboardPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat: add dashboard empty-state CTAs and deep links"
```

---

### Task 15: Load-more feedback and reports flash fix

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx:183-193` (load-more button)
- Modify: `web/src/console/pages/ExecutionsPage.tsx:108-118`
- Modify: `web/src/console/pages/ReportsPage.tsx:39-49` (flash fix)

**Interfaces:**
- Consumes: `items.length` and summary counts.
- Produces: load-more button shows loaded count when a total is known (`Đang hiển thị 25 / 132`), spinner/disabled while loading; reports history reset keyed by filter, not by every data arrival.

- [ ] **Step 1: Write the failing test** (extend `IncidentsPage.test.tsx`)

```tsx
it("shows loaded count on the load-more button", async () => {
  (api.incidents as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [...three], next_cursor: "c1", has_more: true });
  (api.summary as ReturnType<typeof vi.fn>).mockResolvedValue({ ...summaryWithTotal132 });
  renderPage();
  expect(await screen.findByText(/25 \/ 132/i)).toBeTruthy();
});
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement**

In `IncidentsPage.tsx`, replace the load-more button label:

```tsx
        {incidents.hasMore ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void incidents.loadMore()}
            disabled={incidents.loading}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}{incidents.items.length > 0 && summary.data
              ? ` · ${incidents.items.length} / ${summary.data.total_incidents}`
              : ""}
          </button>
        ) : null}
```

For `ReportsPage.tsx` flash fix: replace the `useEffect` resetting `history` with a ref-tracked filter key:

```tsx
  const filterKey = `${stateFilter}`;
  const lastFilterKey = useRef(filterKey);
  useEffect(() => {
    if (lastFilterKey.current !== filterKey) {
      setHistory([]);
      lastFilterKey.current = filterKey;
    }
    setNextCursor(reports.data?.next_cursor ?? null);
  }, [reports.data, filterKey]);
```

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/pages/IncidentsPage.tsx web/src/console/pages/ExecutionsPage.tsx web/src/console/pages/ReportsPage.tsx
git commit -m "fix: show load progress and stop reports button flash"
```

---

### Task 16: Quality page honesty (sample sizes + no-data)

**Files:**
- Modify: `web/src/console/components/QualityPanel.tsx`
- Modify: `web/src/console/pages/QualityPage.tsx`
- Modify: `web/src/console/pages/QualityPage.test.tsx`

**Interfaces:**
- Consumes: `EvaluationSummary` (types.ts:142-161, has `recommendations`, `reviewed`, `llm_calls`, etc.).
- Produces: percentages rendered as "100% (1/1)" when a denominator exists; cards showing `0` or `$0.00` when the sample is empty render muted "Chưa có dữ liệu".

- [ ] **Step 1: Write the failing test**

```tsx
it("shows denominator next to percentage metrics", async () => {
  (api.evaluationSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
    recommendations: 1, reviewed: 1, review_coverage: 1,
    recommendation_acceptance: 0, human_override_rate: 0, unsafe_recommendation_rate: 0,
    unsupported_recommendation_rate: 0, incorrect_next_step_rate: 0, evidence_coverage: 1,
    retrieval_precision: 1, llm_calls: 0, average_latency_ms: 0, tokens_in: 0, tokens_out: 0,
    estimated_cost_micros: 0, workflow_evaluations: 0, workflow_gate_pass_rate: 0,
    generated_at: new Date().toISOString()
  });
  renderPage();
  expect(await screen.findByText(/100% \(1\/1\)/i)).toBeTruthy();
});
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement**

In `QualityPanel.tsx`, find the "Ghi đề của con người 100%" header badge — change it to derive from `reviewed / recommendations` and render `(x/y)`. For zero-value cards in `QualityPage.tsx`, wrap the render: when the source sample (`llm_calls === 0` for cost cards, `recommendations === 0` for coverage cards) is zero, render muted `t("quality.noData")` text instead of `0%`/`$0.00`. Add message key `quality.noData` = en "No data yet" / vi "Chưa có dữ liệu".

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/components/QualityPanel.tsx web/src/console/pages/QualityPage.tsx web/src/console/pages/QualityPage.test.tsx web/src/i18n/messages.ts
git commit -m "fix: show sample sizes and no-data states on quality page"
```

---

### Task 17: Keyboard tabs and dialog focus

**Files:**
- Create: `web/src/console/ui/Tabs.tsx`
- Modify: `web/src/console/pages/IncidentsPage.tsx:96-110`, `web/src/console/pages/ExecutionsPage.tsx:36-47` (use Tabs)
- Modify: `web/src/console/feedback/Dialog.tsx` (verify focus trap)
- Test: `web/src/console/ui/Tabs.test.tsx` (create)

**Interfaces:**
- Produces: `Tabs({ label, tabs: { id, label, count? }[], active, onChange })` with arrow-key navigation, `aria-controls`/`tabpanel` wiring, `role="tablist"`/`role="tab"`/`role="tabpanel"`.

- [ ] **Step 1: Write the failing test**

```tsx
it("moves focus with arrow keys", async () => {
  render(<Tabs label="filter" tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }]} active="a" onChange={() => {}} />);
  const first = screen.getByRole("tab", { name: "A" });
  first.focus();
  fireEvent.keyDown(first, { key: "ArrowRight" });
  expect(screen.getByRole("tab", { name: "B" })).toHaveFocus();
});
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement** — `Tabs.tsx` with `onKeyDown` handling ArrowLeft/ArrowRight/Home/End, `tabIndex` rotation (active tab `0`, others `-1`), `aria-selected`, `aria-controls` pointing at a generated panel id. Update IncidentsPage and ExecutionsPage to use it, keeping the existing `.incident-tabs`/`.tab` classes so styling is unchanged.

For `Dialog.tsx`: verify it already traps focus (check for `focus-trap` or manual tab cycling); if not, add a `useEffect` that on open focuses the dialog and traps Tab within it, restoring focus to the trigger on close. Add a test in `web/src/console/feedback/Dialog.test.tsx` if one exists.

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/ui/Tabs.tsx web/src/console/ui/Tabs.test.tsx web/src/console/pages/IncidentsPage.tsx web/src/console/pages/ExecutionsPage.tsx web/src/console/feedback/Dialog.tsx
git commit -m "a11y: add keyboard navigation to tabs and dialog focus trap"
```

---

### Task 18: Evidence links navigate to source records

**Files:**
- Modify: `web/src/console/components/EvidenceLinks.tsx`
- Modify: `web/src/console/pages/ReportDetailPage.tsx` / `HandoverPanel.tsx` (pass a resolver)
- Test: extend `HandoverPanel.test.tsx` or new `EvidenceLinks.test.tsx`

**Interfaces:**
- Produces: `EvidenceLinks` renders a `<Link>` when the evidence `kind`/`source_id` resolves to a routable record (incident → `/incidents/{source_id}`, document → `/knowledge/{source_id}`); falls back to the current chip for other kinds.

- [ ] **Step 1: Write the failing test**

```tsx
it("links incident evidence to the incident page", async () => {
  render(
    <MemoryRouter>
      <I18nProvider>
        <EvidenceLinks
          evidence={[{ id: "incident:abc", kind: "INCIDENT", source_id: "abc", title: "IN-1042", locator: "", authority: "SITE_APPROVED", content: "", content_sha256: "", score: { rrf_score: 1 } }]}
          selected={["incident:abc"]}
        />
      </I18nProvider>
    </MemoryRouter>,
  );
  expect(screen.getByRole("link", { name: /IN-1042/i })).toHaveAttribute("href", "/incidents/abc");
});
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement** — in `EvidenceLinks.tsx`, when `item.kind` is an incident-like kind (check the actual kind strings used — grep `kind: "INCIDENT"` or `"incident"` in the codebase) render `<Link to={`/incidents/${item.source_id}`}>`, and for document evidence `<Link to={`/knowledge/${item.source_id}`}>`. Keep the existing `details` disclosure. Match the exact `kind` values the backend emits (check `Evidence.kind` usages and the deterministic provider).

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/components/EvidenceLinks.tsx web/src/console/components/EvidenceLinks.test.tsx
git commit -m "feat: make evidence links navigate to source records"
```

---

### Task 19: Relative-time tooltips and layout polish

**Files:**
- Modify: `web/src/console/ui/RelativeTime.tsx`
- Modify: `web/src/console/layout/Topbar.tsx`
- Modify: `web/src/console/layout/Sidebar.tsx`
- Modify: `web/src/i18n/messages.ts`

**Interfaces:**
- Produces: `RelativeTime` sets `title` attribute to the absolute local datetime; Topbar gains a hairline separator between identity and scope; Sidebar safety card is dismissible (localStorage `skawld.safety.dismissed`).

- [ ] **Step 1: Write the failing test** (extend any RelativeTime test)

```tsx
it("exposes absolute time as a tooltip", () => {
  const { container } = render(<I18nProvider><RelativeTime time="2026-08-06T12:00:00Z" locale="vi" /></I18nProvider>);
  expect(container.querySelector("time")?.getAttribute("title")).toContain("2026");
});
```

- [ ] **Step 2: Run to verify failure**

- [ ] **Step 3: Implement**

`RelativeTime.tsx`: add `title={new Date(time).toLocaleString(locale === "vi" ? "vi-VN" : "en-US")}` to the `<time>` element (match how the component currently formats; read the file first).

`Topbar.tsx`: add a `1px` hairline divider (border) between the principal block and the site-scope block; add hover states.

`Sidebar.tsx`: wrap the safety card in a dismissible container — a close button (×) that sets `localStorage["skawld.safety.dismissed"] = "1"`; on mount, skip rendering when set. Add message key `sidebar.dismiss` = en "Dismiss" / vi "Đóng".

- [ ] **Step 4: Run and commit**

```bash
git add web/src/console/ui/RelativeTime.tsx web/src/console/layout/Topbar.tsx web/src/console/layout/Sidebar.tsx web/src/i18n/messages.ts
git commit -m "polish: relative-time tooltips, topbar divider, dismissible safety card"
```

---

## Self-Review Summary

**Spec coverage (2026-08-08 design doc):**
- A1 handover raw JSON → Tasks 1-2
- A2 report identity → Tasks 7-8
- A3 knowledge duplicates → Task 9
- B1 incident page-one counts → Tasks 3, 4, 6
- B2 dashboard unbounded fetch → Tasks 4, 5
- B3 execution client filter → Task 11
- C i18n enums → Tasks 10, 12 (labels)
- D1 execution column redundancy → Task 11
- D2 report columns → Task 8
- D3 knowledge pagination/sort → Task 9 (grouping + existing pagination)
- D4 relative-time tooltip → Task 19
- E1 search canvas → Task 13
- E2 my-queue CTA → Task 14
- E3 org-health chips → Task 14
- E4 load-more feedback → Task 15
- E5 empty states → Tasks 13, 14
- F1/F2 quality honesty → Task 16
- G1 keyboard tabs → Task 17
- G2 dialog focus → Task 17
- G3 reports flash → Task 15
- G4 topbar → Task 19
- G5 sidebar card → Task 19
- G7 evidence links → Task 18
- G8 severity tones → Task 12

**Deferred decisions (from spec §14.3):** incident search stays client-side with an explicit hint (Task 6); knowledge effective marker = latest APPROVED else latest (Task 9); report identity fallback = execution id prefix when no incident (Task 8).

**Placeholder scan:** none — every task has concrete code and test steps. Task 4's SQL column names were verified against `migrations/` and corrected (criticality is a separate table).
