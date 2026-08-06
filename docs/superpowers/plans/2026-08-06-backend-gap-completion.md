# Backend Gap Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three remaining backend gaps from `docs/superpowers/specs/2026-08-06-backend-gap-completion-design.md`: paginate six unbounded list endpoints, make the EAM/CMMS connector operable via CLI + API with a PostgreSQL sink, and add env-selectable AI provider adapters.

**Architecture:** (1) Extend the existing keyset-cursor pagination pattern (reports/handovers) to assets, incidents, executions, documents, demonstrations, and workflows, keeping the `{items, next_cursor, has_more}` envelope. (2) Implement a PostgreSQL `ProjectionSink` that upserts `EXTERNAL_REFERENCE` asset rows, then operate it through a new `cmd/import` CLI and a new `POST /api/v1/integrations/imports` route. (3) Add OpenAI-compatible structured/embedding HTTP adapters and an Anthropic structured adapter in `internal/skawld`, selected by env with deterministic providers as the default fallback.

**Tech Stack:** Go 1.x (module `github.com/ZekromNguyen/skawld-maintenance`), pgx v5 + pgxpool, chi v5, River (jobs), PostgreSQL 16 + pgvector, `internal/platform/keyset`, `internal/integration`, `internal/skawld` boundary. Tests: stdlib `testing` + `httptest`; DB integration tests gated on `TEST_DATABASE_URL`.

## Global Constraints

- `skawld-sdk-go` stays pinned to `v0.2.0`; no `skawld-sdk-go` import may appear outside `internal/skawld`.
- Domain and application packages must not import SQL/HTTP/provider packages. New providers live in `internal/skawld`; the sink is an adapter (`internal/integration/adapter/postgres`).
- Env vars are unprefixed (`STRUCTURED_PROVIDER`, not `SKAWLD_STRUCTURED_PROVIDER`), resolved via the existing `env()` helper in `internal/platform/config/config.go`.
- Every list endpoint keeps the response envelope `{"items": [...], "next_cursor": null|string, "has_more": bool}`; `page_size` must be an integer 1..100 (default 25, via `parsePageSize` in `internal/platform/httpserver/maintenance_helpers.go:85`).
- Errors use the existing `Problem` body and `writeDomainError` mappings; mutations keep idempotency keys and audit.
- TDD: write the failing test first, run it (verify FAIL), implement, run again (verify PASS), commit. Run `gofmt -l .` and `go vet ./...` before each commit.
- Integration tests require `TEST_DATABASE_URL` (they skip otherwise); verify locally with `TEST_DATABASE_URL=... make test-integration`.
- Do not add comments unless they explain *why*; match surrounding code style. No em dashes in code.

---

## Part A — Pagination for six list endpoints

Pattern (replicated per endpoint, modeled on handovers at
`internal/handover/application/service.go:181-224` and
`internal/handover/adapter/postgres/store.go:255-300`):

- Handler: `pageSize, ok := parsePageSize(w, r)`; `cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))`; pass both into the service; write `{"items": items, "next_cursor": nullableString(next), "has_more": hasMore}`.
- Service: returns `([]T, string, error)` — `(items, nextCursor, err)`. Default/cap page size (25/100); when `cursor != ""`, `key, err := keyset.Decode(cursor)` and convert to `key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID`; call store; when `hasMore && len(items) > 0`, `nextCursor = keyset.Encode(<sort field of last item>, last.ID)`.
- Store: returns `([]T, bool, error)` — `(items, hasMore, err)`. Split cursor at `strings.LastIndex(cursor, "|")`; append predicate `(col, id) < ($n, $n::uuid)`; `ORDER BY col DESC, id DESC LIMIT $m` where `$m = filter.PageSize+1`; `hasMore = len(items) > pageSize` (drop the extra row).

The per-endpoint sort column `col` is the endpoint's existing natural order column (below). Cursor ID must be a UUID (enforced by `keyset.Decode`).

### Task A1: Paginate GET /api/v1/assets

**Files:**
- Modify: `internal/asset/application/service.go` (Filter struct at `:93`, `List` at `:162`, `Asset` struct at `:46`)
- Modify: `internal/asset/adapter/postgres/store.go:246-280` (List query + `scanAsset`)
- Modify: `internal/platform/httpserver/assets.go:17-35` (listAssets)
- Test: `internal/platform/httpserver/assets_list_test.go` (create)

**Interfaces:**
- Consumes: `assetapp.Filter{SiteID, Query string, PageSize int, Cursor string}`; `assetapp.Asset` gains `CreatedAt time.Time json:"created_at"`.
- Produces: `assetapp.Service.List(ctx, principal, Filter) ([]Asset, string, error)` (items, next-cursor string, err); `assetapp.Store.List` returns `([]Asset, bool, error)` (items, hasMore, err) — the handler builds the envelope from `(items, next, err)`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/assets_list_test.go`:

```go
package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type listAssetStore struct{}

func (*listAssetStore) List(context.Context, identitydomain.Principal, assetapp.Filter) ([]assetapp.Asset, bool, error) {
	return []assetapp.Asset{{
		ID: "00000000-0000-0000-0000-00000000000b",
		CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func TestListAssetsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Assets: assetapp.Service{
			Store:       &listAssetStore{},
			Authorities: fakeAuthorities{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/assets?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []assetapp.Asset `json:"items"`
		NextCursor *string          `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}
```

Note: `fakeAuthorities` already exists in `internal/platform/httpserver/router_test.go` (verify; if not, copy its minimal definition — it implements `identitydomain.AuthorityReader` with `CanApprove(...) bool`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListAssetsEnvelope -v`
Expected: FAIL — `assets.go` writes `{"items": ...}` with no `next_cursor`/`has_more`, and the service signature has no `(…, bool, error)`.

- [ ] **Step 3: Add PageSize/Cursor and CreatedAt to the asset layer**

In `internal/asset/application/service.go`:

```go
type Filter struct {
	SiteID   string
	Query    string
	PageSize int
	Cursor   string
}
```

Add `CreatedAt time.Time \`json:"created_at"\`` to the `Asset` struct. Change `Service.List` to:

```go
func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter Filter,
) ([]Asset, string, error) {
	if !principal.Has(identitydomain.PermissionAssetRead) {
		return nil, "", ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return nil, "", ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return nil, "", ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, principal, filter)
	if err != nil {
		return nil, "", err
	}
	if !hasMore || len(items) == 0 {
		return items, "", nil
	}
	last := items[len(items)-1]
	return items, keyset.Encode(last.CreatedAt, last.ID), nil
}
```

Add the imports `"time"` and `"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"` to the service file. Update the `Store` interface in the same file: `List(context.Context, identitydomain.Principal, Filter) ([]Asset, bool, error)`.

- [ ] **Step 4: Update the store query**

In `internal/asset/adapter/postgres/store.go`, change `List` to:

```go
func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter application.Filter,
) ([]application.Asset, bool, error) {
	query := `
		SELECT
			a.id::text, a.organization_id::text, a.site_id::text,
			coalesce(r.parent_asset_id::text, ''), a.tag, a.name, a.asset_class,
			coalesce(a.manufacturer, ''), coalesce(a.model, ''), a.status,
			a.source_of_truth, coalesce(a.external_system, ''),
			coalesce(a.external_id, ''), coalesce(a.external_version, ''), a.version,
			a.created_at
		FROM assets a
		LEFT JOIN asset_relationships r
		  ON r.child_asset_id = a.id AND r.relationship_type = 'CONTAINS'
		WHERE a.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR a.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR a.site_id = $3::uuid)
		  AND (
		      nullif($4, '') IS NULL
		      OR a.tag ILIKE '%' || $4 || '%'
		      OR a.name ILIKE '%' || $4 || '%'
		      OR a.asset_class ILIKE '%' || $4 || '%'
		  )`
	args := []any{
		principal.OrganizationID, principal.SiteIDs,
		filter.SiteID, strings.TrimSpace(filter.Query),
	}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, application.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (a.created_at, a.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY a.created_at DESC, a.id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()
	result := make([]application.Asset, 0, filter.PageSize+1)
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, false, err
		}
		result = append(result, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate assets: %w", err)
	}
	hasMore := len(result) > filter.PageSize
	if hasMore {
		result = result[:filter.PageSize]
	}
	return result, hasMore, nil
}
```

Update `scanAsset` to scan the new `a.created_at` column into `&asset.CreatedAt`. Check the existing `scanAsset` call sites: only `List` uses the full SELECT; `Get`/other queries use their own column lists, so `scanAsset` must keep its signature and the extra scan is appended to the SELECT list only where created_at is selected — if `scanAsset` is shared, add `a.created_at` to every SELECT that feeds it. Verify with `go build ./...`.

- [ ] **Step 5: Update the handler**

In `internal/platform/httpserver/assets.go`, replace `listAssets` body (keep the `principalFromRequest` guard):

```go
func listAssets(service assetapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		pageSize, ok := parsePageSize(w, r)
		if !ok {
			return
		}
		filter := assetapp.Filter{
			SiteID:   r.URL.Query().Get("site_id"),
			Query:    r.URL.Query().Get("q"),
			PageSize: pageSize,
			Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		}
		if filter.SiteID != "" && !validUUIDParam(w, filter.SiteID, "site ID") {
			return
		}
		items, next, err := service.List(r.Context(), principal, filter)
		if err != nil {
			writeDomainError(w, err, assetapp.ErrForbidden, assetapp.ErrNotFound,
				assetapp.ErrInvalid, assetapp.ErrVersionConflict)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":       items,
			"next_cursor": nullableString(next),
			"has_more":    next != "",
		})
	}
}
```

(Note: `Service.List` returns the encoded next cursor string, so the handler does not need a `ListResult` type. Keep `assetapp.ErrNotFound` in the mapping if it exists, else drop it.)

- [ ] **Step 6: Update OpenAPI for GET /assets**

In `api/openapi.yaml`, add to the `GET /api/v1/assets` parameters: `page_size` (integer, 1-100, default 25) and `cursor` (string); change the `200` response schema to include `items` (array), `next_cursor` (nullable string), `has_more` (boolean). Mirror the schema used by `GET /api/v1/handovers`.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/asset/... && go vet ./... && go build ./...`
Expected: PASS. Also run `gofmt -l .` (must print nothing).

- [ ] **Step 8: Commit**

```bash
git add internal/asset internal/platform/httpserver/assets.go internal/platform/httpserver/assets_list_test.go api/openapi.yaml
git commit -m "feat: paginate the asset list endpoint"
```

### Task A2: Paginate GET /api/v1/incidents

**Files:**
- Modify: `internal/incident/application/service.go` (Filter at `:58`, `List` at `:103`)
- Modify: `internal/incident/adapter/postgres/store.go:151-177` (List)
- Modify: `internal/platform/httpserver/incidents.go:23-45` (listIncidents)
- Test: `internal/platform/httpserver/incidents_list_test.go` (create)

**Interfaces:**
- Consumes: `incidentapp.Filter{SiteID, AssetID, State string, PageSize int, Cursor string}`; sort column is `i.detected_at` (already on `Incident` as `DetectedAt`).
- Produces: `incidentapp.Service.List(ctx, principal, Filter) ([]Incident, string, error)` (items, next-cursor string, err); `incidentapp.Store.List` returns `([]Incident, bool, error)` (items, hasMore, err).

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/incidents_list_test.go`, modeled on A1: a fake `incidentapp.Store` whose `List` returns one incident with `ID: "00000000-0000-0000-0000-00000000000c"` and `DetectedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)` plus `hasMore = true`; principal with `PermissionIncidentRead`; request `/api/v1/incidents?page_size=1`; assert `200`, envelope with 1 item, `has_more = true`, non-empty `next_cursor`. The `Incident` JSON type used by the store is `incidentapp.Incident` (it has both `Incident` and `IncidentListItem` types — check which `List` returns and assert against that).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListIncidentsEnvelope -v`
Expected: FAIL (no envelope, signature mismatch).

- [ ] **Step 3: Update the application service**

Add `PageSize int` and `Cursor string` to `incidentapp.Filter`. Change `Service.List` to the same shape as A1 Step 3 with the `([]Incident, string, error)` return (permission `PermissionIncidentRead`, page defaults, `keyset.Decode` → `timestamp|id`, store call returning `([]Incident, bool, error)`, `hasMore && len(items) > 0` → `keyset.Encode(last.DetectedAt, last.ID)`). Add `"time"` and the `keyset` import.

- [ ] **Step 4: Update the store query**

`incidentapp.Store.List` returns `([]Incident, bool, error)`. In `internal/incident/adapter/postgres/store.go`, append the cursor predicate after the existing `state` filter:

```go
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, incidentapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (i.detected_at, i.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY i.detected_at DESC, i.id DESC LIMIT $%d", len(args))
```

Convert the query string to the `query += fmt.Sprintf(...)` style used by the handover store (the current incident List uses one literal string; refactor to append style, keeping the same `incidentSelect` prefix). Truncate `result` to `filter.PageSize` when `hasMore`. Ensure `incidentapp.ErrInvalid` exists (it does: used by the service).

- [ ] **Step 5: Update the handler**

Same shape as A1 Step 5, using `filter := incidentapp.Filter{SiteID: r.URL.Query().Get("site_id"), AssetID: r.URL.Query().Get("asset_id"), State: r.URL.Query().Get("state"), PageSize: pageSize, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}` and `writeDomainError(w, err, incidentapp.ErrForbidden, incidentapp.ErrNotFound, incidentapp.ErrInvalid)`.

- [ ] **Step 6: Update OpenAPI for GET /incidents**

Add `page_size` + `cursor` params and the envelope response schema (mirror handovers).

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/incident/... && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 8: Commit**

```bash
git add internal/incident internal/platform/httpserver/incidents.go internal/platform/httpserver/incidents_list_test.go api/openapi.yaml
git commit -m "feat: paginate the incident list endpoint"
```

### Task A3: Paginate GET /api/v1/executions

**Files:**
- Modify: `internal/execution/application/service.go` (Filter at `:107`, `List` at `:245`, `Execution` struct at `:188`)
- Modify: `internal/execution/adapter/postgres/query.go:27-70` (List) and `internal/execution/adapter/postgres/lifecycle.go` (the `loadExecution` SELECT that scans into `Execution`)
- Modify: `internal/platform/httpserver/executions.go:30-50` (listExecutions)
- Test: `internal/platform/httpserver/executions_list_test.go` (create)

**Interfaces:**
- Consumes: `executionapp.Filter{SiteID, State, AssignedTo string, PageSize int, Cursor string}`; `executionapp.Execution` gains `UpdatedAt time.Time json:"updated_at"`.
- Produces: `executionapp.Service.List(ctx, principal, Filter) ([]Execution, string, error)` (items, next-cursor, err); store `List` returns `([]Execution, bool, error)`; the ID query adds the cursor predicate on `(e.updated_at, e.id)` and `LIMIT pageSize+1`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/executions_list_test.go` modeled on A1: fake store `List` returns `[]executionapp.Execution{{ID: "00000000-0000-0000-0000-00000000000d", UpdatedAt: ...}}` with `hasMore = true`; principal with `PermissionExecutionRead`; assert envelope. The fake store must implement the full `executionapp.Store` interface — check `internal/execution/application/service.go` for the `Store` interface (Create/Get/List/Start/CompleteStep/RecordMeasurement/RecordObservation/CompleteAction/RecordAction/RecordDecision/VerifyPrerequisite/Complete) and stub the unused methods like `listReportStore` does in `reports_list_test.go`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListExecutionsEnvelope -v`
Expected: FAIL.

- [ ] **Step 3: Update the application service and struct**

Add `PageSize`/`Cursor` to `executionapp.Filter`. Add `UpdatedAt time.Time \`json:"updated_at"\`` to `Execution`. Change `Service.List` to the standard shape with the `([]Execution, string, error)` return (permission `PermissionExecutionRead`; page defaults; `keyset.Decode` → `timestamp|id`; store returns `([]Execution, bool, error)`; `hasMore && len(items) > 0` → `keyset.Encode(last.UpdatedAt, last.ID)`).

- [ ] **Step 4: Update the store query**

In `internal/execution/adapter/postgres/query.go`, change `List` to append the cursor predicate after `assigned_to`:

```go
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, executionapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (e.updated_at, e.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY e.updated_at DESC, e.id DESC LIMIT $%d", len(args))
```

Truncate `ids` to `filter.PageSize` when more than that many were fetched, then `hasMore := len(ids) > filter.PageSize` *before* truncation; after loading, return `(result, hasMore, nil)`. Update `loadExecution` in `internal/execution/adapter/postgres/lifecycle.go` to select `e.updated_at` and scan it into `value.UpdatedAt`. Locate the SELECT in `loadExecution` (it uses a `queryer` interface; add `e.updated_at` to the column list and a matching `&value.UpdatedAt` scan).

- [ ] **Step 5: Update the handler**

Same shape as A1 Step 5 with `filter := executionapp.Filter{SiteID: ..., State: r.URL.Query().Get("state"), AssignedTo: r.URL.Query().Get("assigned_to"), PageSize: pageSize, Cursor: ...}` and `writeDomainError(w, err, executionapp.ErrForbidden, executionapp.ErrNotFound, executionapp.ErrInvalid)`.

- [ ] **Step 6: Update OpenAPI for GET /executions**

Add `page_size` + `cursor` params and the envelope response schema.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/execution/... && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 8: Commit**

```bash
git add internal/execution internal/platform/httpserver/executions.go internal/platform/httpserver/executions_list_test.go api/openapi.yaml
git commit -m "feat: paginate the execution list endpoint"
```

### Task A4: Paginate GET /api/v1/documents

**Files:**
- Modify: `internal/knowledge/application/service.go` (Filter at `:50`, `ListDocuments` at `:225`)
- Modify: `internal/knowledge/adapter/postgres/query.go:43-75` (ListDocuments)
- Modify: `internal/platform/httpserver/knowledge.go:24-42` (listDocuments)
- Test: `internal/platform/httpserver/documents_list_test.go` (create)

**Interfaces:**
- Consumes: `knowledgeapp.Filter{SiteID string, PageSize int, Cursor string}`; `knowledgedomain.Document` already has `UpdatedAt` (`internal/knowledge/domain/document.go:109`).
- Produces: `knowledgeapp.Service.ListDocuments(ctx, principal, Filter) ([]Document, string, error)` (items, next-cursor, err); `knowledgeapp.Store.ListDocuments` returns `([]Document, bool, error)`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/documents_list_test.go` modeled on A1: fake `knowledgeapp.Store` whose `ListDocuments` returns `[]knowledgedomain.Document{{ID: "00000000-0000-0000-0000-00000000000e", UpdatedAt: ...}}` with `hasMore = true`; principal with `PermissionKnowledgeRead`; request `/api/v1/documents?page_size=1`; assert envelope. The fake must implement the full `knowledgeapp.Store` interface (check `internal/knowledge/application/service.go`; stub unused methods like `listReportStore`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListDocumentsEnvelope -v`
Expected: FAIL.

- [ ] **Step 3: Update the application service**

Add `PageSize`/`Cursor` to `knowledgeapp.Filter`. Change `Service.ListDocuments` to the standard shape with the `([]Document, string, error)` return (permission `PermissionKnowledgeRead`; page defaults; `keyset.Decode`; store returns `([]Document, bool, error)`; `keyset.Encode(last.UpdatedAt, last.ID)` when `hasMore`).

- [ ] **Step 4: Update the store query**

Replace the hardcoded `LIMIT 100` in `ListDocuments` with the cursor pattern. The existing query uses args `($1 org, $2 siteIDs, $3 siteID)`:

```go
	query := `
		SELECT id::text
		FROM documents
		WHERE organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR site_id IS NULL OR site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR site_id::text = nullif($3, ''))`
	args := []any{principal.OrganizationID, principal.SiteIDs, filter.SiteID}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, knowledgeapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (updated_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY updated_at DESC, id DESC LIMIT $%d", len(args))
	rows, err := s.Pool.Query(ctx, query, args...)
```

Fetch `pageSize+1` IDs, set `hasMore` before truncating to `filter.PageSize`, then load each document with `GetDocument` and return `(result, hasMore, nil)`. (Do not call `rows.Close()` before reading all rows — the current code defers `rows.Close()` and reads all rows first; keep that structure and collect all IDs, then truncate the ID slice.)

- [ ] **Step 5: Update the handler**

Same shape as A1 Step 5 with `filter := knowledgeapp.Filter{SiteID: r.URL.Query().Get("site_id"), PageSize: pageSize, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}` and the existing `writeKnowledgeError` (or `writeDomainError` — use whatever `listDocuments` already calls).

- [ ] **Step 6: Update OpenAPI for GET /documents**

Add `page_size` + `cursor` params and the envelope response schema.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/knowledge/... && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 8: Commit**

```bash
git add internal/knowledge internal/platform/httpserver/knowledge.go internal/platform/httpserver/documents_list_test.go api/openapi.yaml
git commit -m "feat: paginate the document list endpoint"
```

### Task A5: Paginate GET /api/v1/demonstrations

**Files:**
- Modify: `internal/demonstration/application/service.go` (`List` at `:173`, `Gateway` interface at `:126`)
- Modify: `internal/skawld/demonstrations.go:166-200` (`DemonstrationGateway.List`)
- Modify: `internal/platform/httpserver/demonstrations.go:53-68` (listDemonstrations)
- Modify: `internal/skawld/demonstrations_integration_test.go` (existing calls to `Gateway.List`)
- Test: `internal/platform/httpserver/demonstrations_list_test.go` (create)

**Interfaces:**
- Consumes: `demonstrationapp.ListFilter{PageSize int, Cursor string}` (new, defined in the service file); `Demonstration` already has `StartedAt` (`service.go:124`).
- Produces: `demonstrationapp.Service.List(ctx, principal, siteID string, filter ListFilter) ([]Demonstration, string, error)` (items, next-cursor, err); `demonstrationapp.Gateway.List(ctx, principal, siteID string, filter ListFilter) ([]Demonstration, bool, error)`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/demonstrations_list_test.go`: fake `demonstrationapp.Gateway` implementing all interface methods (`Start`, `Get`, `List`, `Complete`, `RecordEvidenceView`, `Review`, `RedactEvent`) with `List` returning `[]demonstrationapp.Demonstration{{ID: "00000000-0000-0000-0000-00000000000f", StartedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)}}` and `hasMore = true`; principal with `PermissionDemonstrationRead`; request `/api/v1/demonstrations?page_size=1`; assert the envelope.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListDemonstrationsEnvelope -v`
Expected: FAIL.

- [ ] **Step 3: Update the application service**

Add near the `Gateway` interface:

```go
type ListFilter struct {
	PageSize int
	Cursor   string
}
```

Change the `Gateway` interface method `List(context.Context, identitydomain.Principal, string) ([]Demonstration, error)` to `List(context.Context, identitydomain.Principal, string, ListFilter) ([]Demonstration, bool, error)`. Change `Service.List` to the standard shape with the `([]Demonstration, string, error)` return (permission `PermissionDemonstrationRead`; site check; page defaults; `keyset.Decode` → `timestamp|id`; gateway returns `([]Demonstration, bool, error)`; `keyset.Encode(last.StartedAt, last.ID)` when `hasMore`).

- [ ] **Step 4: Update the gateway query**

In `internal/skawld/demonstrations.go`, change `DemonstrationGateway.List` to take the `ListFilter` and append:

```go
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, demonstrationapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (started_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY started_at DESC, id DESC LIMIT $%d", len(args))
```

Truncate `ids` to `filter.PageSize` (keeping `hasMore` computed before truncation), then load each via `g.Get` and return `(output, hasMore, nil)`.

- [ ] **Step 5: Update the handler**

Same shape as A1 Step 5 with `filter := demonstrationapp.ListFilter{PageSize: pageSize, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}` and the existing site_id param handling kept.

- [ ] **Step 6: Update the integration test call sites**

`internal/skawld/demonstrations_integration_test.go` calls `Gateway.List(...)` — update those calls to pass `demonstrationapp.ListFilter{}` and handle the `(…, bool, error)` return (assert on the slice; ignore/assert `hasMore` as appropriate).

- [ ] **Step 7: Update OpenAPI for GET /demonstrations**

Add `page_size` + `cursor` params and the envelope response schema.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/demonstration/... ./internal/skawld/ && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 9: Commit**

```bash
git add internal/demonstration internal/skawld internal/platform/httpserver/demonstrations.go internal/platform/httpserver/demonstrations_list_test.go api/openapi.yaml
git commit -m "feat: paginate the demonstration list endpoint"
```

### Task A6: Paginate GET /api/v1/workflows

**Files:**
- Modify: `internal/workflow/application/service.go` (`List` at `:191`, `Gateway` interface, `Version` struct at `:107`)
- Modify: `internal/skawld/workflow_learning.go:219-258` (`WorkflowLearningGateway.List`)
- Modify: `internal/platform/httpserver/workflows.go:63-75` (listWorkflows)
- Modify: `internal/skawld/workflow_learning_integration_test.go` (existing calls to `Gateway.List`)
- Test: `internal/platform/httpserver/workflows_list_test.go` (create)

**Interfaces:**
- Consumes: `workflowapp.ListFilter{PageSize int, Cursor string}` (new); `Version` already has `CreatedAt` (`service.go:132`).
- Produces: `workflowapp.Service.List(ctx, principal, filter ListFilter) ([]Version, string, error)` (items, next-cursor, err); `workflowapp.Gateway.List(ctx, principal, filter ListFilter) ([]Version, bool, error)`.

**Cursor note (three-part):** workflow versions are keyed by `(workflow_id, version)` and `created_at` is not unique per row, so a two-part cursor can silently skip rows when two versions share `(created_at, workflow_id)`. Use a three-part cursor `created_at|workflow_id|version`, encoded as base64url JSON `{"t":...,"w":...,"v":...}`. Implement `encodeWorkflowCursor(createdAt time.Time, workflowID string, version int) string` and `decodeWorkflowCursor(raw string) (time.Time, string, int, error)` in `internal/workflow/application/service.go` (small, private, ~30 lines; base64.RawURLEncoding + encoding/json). On decode error return `ErrInvalid`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/workflows_list_test.go`: fake `workflowapp.Gateway` implementing all interface methods (check the `Gateway` interface in the service file; stub unused methods) with `List` returning `[]workflowapp.Version{{WorkflowID: "00000000-0000-0000-0000-000000000010", Version: 1, CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)}}` and `hasMore = true`; principal with `PermissionWorkflowRead`; request `/api/v1/workflows?page_size=1`; assert the envelope.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestListWorkflowsEnvelope -v`
Expected: FAIL.

- [ ] **Step 3: Update the application service**

Add `ListFilter{PageSize, Cursor}` and the two cursor helper functions. Change the `Gateway` interface `List` to `List(context.Context, identitydomain.Principal, ListFilter) ([]Version, bool, error)`. Change `Service.List` to the standard shape with the `([]Version, string, error)` return (permission `PermissionWorkflowRead`; page defaults; decode via `decodeWorkflowCursor`; gateway returns `([]Version, bool, error)`; when `hasMore && len(items) > 0`, the returned cursor is `encodeWorkflowCursor(last.CreatedAt, last.WorkflowID, last.Version)`).

- [ ] **Step 4: Update the gateway query**

In `internal/skawld/workflow_learning.go`, change `WorkflowLearningGateway.List` to append:

```go
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, workflowapp.ErrInvalid
		}
		parts := strings.SplitN(filter.Cursor, "|", 3)
		if len(parts) != 3 {
			return nil, false, workflowapp.ErrInvalid
		}
		args = append(args, parts[0], parts[1], parts[2])
		query += fmt.Sprintf(
			" AND (v.created_at, v.workflow_id, v.version) < ($%d, $%d::uuid, $%d)",
			len(args)-2, len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(
		" ORDER BY v.created_at DESC, v.workflow_id DESC, v.version DESC LIMIT $%d",
		len(args))
```

The service decodes the base64 cursor and passes it to the gateway as `created_at RFC3339Nano|workflow_id|version` (same `timestamp|id`-style contract as the other stores, but with a third part). The identity SELECT stays `workflow_id, version`; truncate `identities` to `filter.PageSize` (with `hasMore` computed first), then load via `g.Get` and return `(result, hasMore, nil)`.

- [ ] **Step 5: Update the handler**

Same shape as A1 Step 5 with `filter := workflowapp.ListFilter{PageSize: pageSize, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}`.

- [ ] **Step 6: Update the integration test call sites**

`internal/skawld/workflow_learning_integration_test.go` calls `Gateway.List(...)` — update to pass `workflowapp.ListFilter{}` and handle the tuple return.

- [ ] **Step 7: Update OpenAPI for GET /workflows**

Add `page_size` + `cursor` params and the envelope response schema.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ ./internal/workflow/... ./internal/skawld/ && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 9: Commit**

```bash
git add internal/workflow internal/skawld internal/platform/httpserver/workflows.go internal/platform/httpserver/workflows_list_test.go api/openapi.yaml
git commit -m "feat: paginate the workflow list endpoint"
```

---

## Part B — EAM/CMMS connector operation (CLI + API)

### Task B1: PostgreSQL external-asset projection sink

**Files:**
- Create: `internal/integration/adapter/postgres/sink.go`
- Create: `internal/integration/adapter/postgres/sink_integration_test.go`
- Modify: `cmd/api/main.go` (wire the sink — later in Task B3; this task only creates the type)

**Interfaces:**
- Consumes: `integrationdomain.ExternalRecord` (`Kind`, `OrganizationID`, `SiteID`, `ExternalSystem`, `ExternalID`, `ExternalVersion`, `ObservedAt`, `Attributes map[string]any`); `integrationapp.ProjectionSink` (`Apply(ctx, principal, records) error`); `id.Generator`, `clock.Clock`, `audit.Sink`, `database.InTx`.
- Produces: `postgres.Sink{Pool *pgxpool.Pool; IDs id.Generator; Clock clock.Clock; Audit audit.Sink}` implementing `Apply`.

- [ ] **Step 1: Write the failing integration test**

Create `internal/integration/adapter/postgres/sink_integration_test.go`:

```go
//go:build integration
// +build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
```

Wait — the repo's integration tests do not use a build tag; they skip on missing `TEST_DATABASE_URL` (see `internal/execution/adapter/postgres/store_integration_test.go`). Match that: no build tag; `t.Skip` when the env var is empty. Do NOT use a build tag in this file.

Write three test functions (each uses `openTestPool` + seeds an org/site via raw SQL, then calls `sink.Apply` with a principal built from `identitydomain.Principal{ID: ..., OrganizationID: ..., SiteIDs: [...]}` with `PermissionsForRole(RoleAdministrator)`):

- `TestSinkInsertsExternalAsset` — one `ASSET` record with `Attributes{"tag": "P-302", "name": "Pump P-302", "asset_class": "PUMP"}`; assert the row exists with `source_of_truth = 'EXTERNAL_REFERENCE'`, `external_system`/`external_id`/`external_version` set, `sync_status` non-null.
- `TestSinkPreservesOwnedBySkawld` — pre-insert an `OWNED_BY_SKAWLD` asset with the same `external_system`/`external_id`; `Apply` succeeds and the row's `name`/`tag` are unchanged.
- `TestSinkUpsertByVersion` — insert an external asset with `external_version = "v1"`, then `Apply` again with the same version (assert unchanged) and with `"v2"` (assert `external_version = 'v2'` and `version` bumped).
- `TestSinkRejectsUnsupportedKind` — a `WORK_REFERENCE` record makes `Apply` return an error and inserts nothing.

- [ ] **Step 2: Run test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test ./internal/integration/adapter/postgres/ -count=1`
Expected: FAIL — `postgres` package does not exist yet (package not found).

- [ ] **Step 3: Implement the sink**

Create `internal/integration/adapter/postgres/sink.go`:

```go
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Sink struct {
	Pool  *pgxpool.Pool
	IDs   id.Generator
	Clock clock.Clock
	Audit audit.Sink
}

var _ application.ProjectionSink = Sink{}

func (s Sink) Apply(
	ctx context.Context,
	principal identitydomain.Principal,
	records []integrationdomain.ExternalRecord,
) error {
	now := s.Clock.Now()
	type outcome struct{}
	return database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		for _, record := range records {
			if record.Kind != "ASSET" {
				return outcome{}, fmt.Errorf("sink cannot project external record kind %q", record.Kind)
			}
			tag, err := requiredString(record.Attributes, "tag")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			name, err := requiredString(record.Attributes, "name")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			assetClass, err := requiredString(record.Attributes, "asset_class")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			attributesJSON, err := json.Marshal(record.Attributes)
			if err != nil {
				return outcome{}, fmt.Errorf("encode external asset attributes: %w", err)
			}

			var existingSource, existingVersion string
			err = tx.QueryRow(ctx, `
				SELECT source_of_truth, coalesce(external_version, '')
				FROM assets
				WHERE organization_id = $1::uuid
				  AND external_system = $2
				  AND external_id = $3
			`, principal.OrganizationID, record.ExternalSystem, record.ExternalID).
				Scan(&existingSource, &existingVersion)
			switch {
			case err == nil && existingSource == "OWNED_BY_SKAWLD":
				// An owned record can never be re-owned or modified by an
				// external projection.
				continue
			case err == nil && existingVersion == record.ExternalVersion:
				continue
			case err == nil:
				if _, err := tx.Exec(ctx, `
					UPDATE assets
					SET name = $1, asset_class = $2, external_version = $3,
					    attributes = $4, sync_status = 'IN_SYNC',
					    version = version + 1, updated_at = $5
					WHERE organization_id = $6::uuid
					  AND external_system = $7
					  AND external_id = $8
				`, name, assetClass, record.ExternalVersion, attributesJSON, now,
					principal.OrganizationID, record.ExternalSystem, record.ExternalID); err != nil {
					return outcome{}, fmt.Errorf("update external asset: %w", err)
				}
			case errors.Is(err, pgx.ErrNoRows):
				if _, err := tx.Exec(ctx, `
					INSERT INTO assets (
						id, organization_id, site_id, tag, name, asset_class,
						manufacturer, model, status, source_of_truth,
						external_system, external_id, external_version, sync_status,
						attributes, version, created_at, updated_at
					) VALUES (
						$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
						nullif($7, ''), nullif($8, ''), coalesce(nullif($9, ''), 'ACTIVE'),
						'EXTERNAL_REFERENCE', $10, $11, $12, 'IN_SYNC',
						$13, 1, $14, $14
					)
				`, s.IDs.New(), principal.OrganizationID, record.SiteID, tag, name, assetClass,
					optionalString(record.Attributes, "manufacturer"),
					optionalString(record.Attributes, "model"),
					optionalString(record.Attributes, "status"),
					record.ExternalSystem, record.ExternalID, record.ExternalVersion,
					attributesJSON, now); err != nil {
					return outcome{}, fmt.Errorf("insert external asset: %w", err)
				}
			default:
				return outcome{}, fmt.Errorf("lookup external asset: %w", err)
			}
			s.Audit.Append(ctx, tx, audit.Event{
				OrganizationID: principal.OrganizationID,
				SiteID:         record.SiteID,
				ActorID:        principal.ID,
				Action:         "integration.external-asset-projected",
				EntityKind:     "asset",
				EntityID:       record.ExternalID,
				Attributes: map[string]any{
					"external_system":   record.ExternalSystem,
					"external_version":  record.ExternalVersion,
				},
				OccurredAt: now,
			})
		}
		return outcome{}, nil
	})
}

func requiredString(attributes map[string]any, key string) (string, error) {
	value := strings.TrimSpace(fmt.Sprint(attributes[key]))
	if value == "" {
		return "", fmt.Errorf("missing required attribute %q", key)
	}
	return value, nil
}

func optionalString(attributes map[string]any, key string) string {
	value, ok := attributes[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
```

The real `audit.Event` fields are `ID, OrganizationID, SiteID, ActorID, Action, EntityKind, EntityID, Reason, RequestID, ExecutionID, WorkflowID, ApprovalID, AIInvolvement, Before, After, Attributes, OccurredAt` and `Append(ctx, tx, event)` (see `internal/platform/audit/audit.go:12-60`). Use exactly the fields shown in the code block above.

- [ ] **Step 4: Run tests to verify they pass**

Run: `TEST_DATABASE_URL=... go test ./internal/integration/adapter/postgres/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/integration/adapter/postgres
git commit -m "feat: add external asset projection sink for connector imports"
```

### Task B2: cmd/import CLI

**Files:**
- Create: `cmd/import/main.go`
- Create: `test/fixtures/import-p302.ndjson` (fixture snapshot)
- Create: `cmd/import/main_test.go`
- Modify: `ReadMe.md` (usage section) and `docs/runbooks/demo-data.md` (import usage)

**Interfaces:**
- Consumes: `ndjson.New(path, allowedRoot, identity) (Connector, error)`; `integrationapp.Importer{Connector, Sink}`; `postgres.Sink` (Task B1); `identitydomain.PermissionsForRole(RoleAdministrator)`; `database.Open`.
- Produces: runnable `go run ./cmd/import -snapshot <file> -site-id <uuid>` binary.

- [ ] **Step 1: Create the fixture**

Create `test/fixtures/import-p302.ndjson` (one record per line; `observed_at` RFC3339):

```ndjson
{"kind":"ASSET","organization_id":"REPLACED_BY_PRINCIPAL","site_id":"REPLACED_BY_SITE","external_system":"CMMS-X","external_id":"a-1001","external_version":"v1","observed_at":"2026-08-01T09:00:00Z","attributes":{"tag":"P-302","name":"Centrifugal Pump P-302","asset_class":"PUMP","manufacturer":"Grundfos","model":"CR-95","status":"ACTIVE"}}
{"kind":"ASSET","organization_id":"REPLACED_BY_PRINCIPAL","site_id":"REPLACED_BY_SITE","external_system":"CMMS-X","external_id":"a-1002","external_version":"v1","observed_at":"2026-08-01T09:01:00Z","attributes":{"tag":"M-101","name":"Motor M-101","asset_class":"MOTOR","manufacturer":"Siemens","model":"1LE5","status":"ACTIVE"}}
```

The CLI rewrites `organization_id`/`site_id` placeholders in memory before import (see Step 3) so the fixture is tenant-neutral.

- [ ] **Step 2: Write the failing CLI test**

Create `cmd/import/main_test.go` (skip unless `TEST_DATABASE_URL`): seeds an org/site via raw SQL, writes the fixture with real IDs to a temp file, runs `runImport(ctx, cfg, args)` (the extracted core of the CLI — see Step 3), and asserts:
- exit-style error is nil;
- the two assets exist with `source_of_truth = 'EXTERNAL_REFERENCE'` and `external_version = 'v1'`;
- a second run with the same snapshot reports zero new/updated rows (idempotent).

- [ ] **Step 3: Implement the CLI**

Create `cmd/import/main.go`:

```go
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/postgres"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/ndjson"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

type importArgs struct {
	Snapshot       string
	SiteID         string
	DatabaseURL    string
	Cursor         string
	Limit          int
	ExternalSystem string
	ExternalSubject string
}

func main() {
	args := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := runImport(ctx, logger, args); err != nil {
		logger.Error("import failed", "error", err)
		os.Exit(1)
	}
}

func parseFlags() importArgs {
	args := importArgs{}
	flag.StringVar(&args.Snapshot, "snapshot", "", "NDJSON snapshot path (required)")
	flag.StringVar(&args.SiteID, "site-id", "", "target site UUID (required)")
	flag.StringVar(&args.DatabaseURL, "database-url", os.Getenv("DATABASE_URL"), "PostgreSQL URL")
	flag.StringVar(&args.Cursor, "cursor", "", "resume cursor from a previous run")
	flag.IntVar(&args.Limit, "limit", 100, "records per page (1-500)")
	flag.StringVar(&args.ExternalSystem, "external-system", "CMMS", "external system identity")
	flag.StringVar(&args.ExternalSubject, "external-subject", "import-admin", "administrator principal external subject")
	flag.Parse()
	return args
}

func runImport(ctx context.Context, logger *slog.Logger, args importArgs) error {
	if strings.TrimSpace(args.Snapshot) == "" || strings.TrimSpace(args.SiteID) == "" {
		return fmt.Errorf("-snapshot and -site-id are required")
	}
	if args.Limit < 1 || args.Limit > 500 {
		return fmt.Errorf("-limit must be between 1 and 500")
	}
	pool, err := database.Open(ctx, database.Config{
		URL: args.DatabaseURL, APIMaxConns: 5, WorkerMaxConns: 5,
		MaxBudget: 10, StatementTimeout: 15 * time.Second, LockTimeout: 3 * time.Second,
	}, config.RoleAPI)
	if err != nil {
		return err
	}
	defer pool.Close()
	principal, err := loadImportPrincipal(ctx, pool, args.ExternalSubject, args.SiteID)
	if err != nil {
		return err
	}
	snapshot, err := os.ReadFile(args.Snapshot)
	if err != nil {
		return err
	}
	rewritten := strings.ReplaceAll(string(snapshot), "REPLACED_BY_PRINCIPAL", principal.OrganizationID)
	rewritten = strings.ReplaceAll(rewritten, "REPLACED_BY_SITE", principal.SiteIDs[0])
	tempFile, err := os.CreateTemp("", "import-*.ndjson")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.WriteString(rewritten); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	connector, err := ndjson.New(tempFile.Name(), os.TempDir(), integrationdomain.ConnectorIdentity{
		System: args.ExternalSystem, Instance: "import-cli", Version: "1.0",
		ReadOnly: true, Capabilities: []integrationdomain.Capability{
			integrationdomain.CapabilityReadAssets,
		},
	})
	if err != nil {
		return err
	}
	importer := integrationapp.Importer{
		Connector: connector,
		Sink: integrationpostgres.Sink{
			Pool: pool, IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
		},
	}
	cursor := args.Cursor
	total := 0
	for {
		page, err := importer.Pull(ctx, principal, cursor, args.Limit)
		if err != nil {
			return err
		}
		total += len(page.Records)
		logger.Info("import page", "records", len(page.Records), "next_cursor", page.NextCursor)
		if page.Complete {
			break
		}
		cursor = page.NextCursor
	}
	logger.Info("import complete", "total_records", total)
	return nil
}

// loadImportPrincipal resolves an administrator principal for the target site
// so the import runs under real tenant/site scope and RBAC.
func loadImportPrincipal(
	ctx context.Context,
	pool *pgxpool.Pool,
	subject string,
	siteID string,
) (identitydomain.Principal, error) {
	var principalID, organizationID string
	err := pool.QueryRow(ctx, `
		SELECT p.id::text, m.organization_id::text
		FROM principals p
		JOIN memberships m ON m.principal_id = p.id
		WHERE p.external_subject = $1 AND m.role = 'Administrator' AND m.site_id = $2::uuid
		LIMIT 1
	`, subject, siteID).Scan(&principalID, &organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return identitydomain.Principal{}, fmt.Errorf(
			"no administrator principal with subject %q in site %s", subject, siteID)
	}
	if err != nil {
		return identitydomain.Principal{}, err
	}
	permissions := make(map[identitydomain.Permission]struct{})
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		permissions[permission] = struct{}{}
	}
	return identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID,
		SiteIDs: []string{siteID}, Permissions: permissions,
	}, nil
}
```

No new config role is added: the CLI builds the pool from a `database.Config` literal (it needs only a database URL, not the S3/OIDC requirements `config.Load(RoleAPI)` validation enforces). Add the missing imports (`errors`, `github.com/jackc/pgx/v5`, `github.com/jackc/pgx/v5/pgxpool`, `integrationapp`, `integrationpostgres`, `integrationdomain`, `ndjson`, `database`, `config`, `audit`, `clock`, `id` are used above; `identitypostgres` is not needed).

- [ ] **Step 4: Run tests to verify they pass**

Run: `TEST_DATABASE_URL=... go test ./cmd/import/ -count=1`
Expected: PASS.

- [ ] **Step 5: Manual smoke test**

Run: `go run ./cmd/import -snapshot test/fixtures/import-p302.ndjson -site-id <uuid> -database-url "$TEST_DATABASE_URL"`
Expected: per-page log lines, exit 0; `psql` shows the two external assets.

- [ ] **Step 6: Document usage**

Add a short "Importing external EAM/CMMS data" section to `ReadMe.md` and a `cmd/import` paragraph in `docs/runbooks/demo-data.md` (flags, example invocation, idempotency note).

- [ ] **Step 7: Commit**

```bash
git add cmd/import test/fixtures/import-p302.ndjson ReadMe.md docs/runbooks/demo-data.md
git commit -m "feat: add EAM/CMMS snapshot import CLI"
```

### Task B3: POST /api/v1/integrations/imports route

**Files:**
- Modify: `internal/platform/httpserver/router.go` (Dependencies struct, `mountIntegrationRoutes`)
- Create: `internal/platform/httpserver/integrations.go`
- Create: `internal/platform/httpserver/integrations_test.go`
- Modify: `cmd/api/main.go` (wire `IntegrationSink`)
- Modify: `api/openapi.yaml` (new path)

**Interfaces:**
- Consumes: `integrationpostgres.Sink` (Task B1), `ndjson.New`, `integrationapp.Importer`, `integrationdomain` constants, `principalFromRequest`.
- Produces: `httpserver.Dependencies.IntegrationSink integrationapp.ProjectionSink`; route `POST /api/v1/integrations/imports`.

- [ ] **Step 1: Write the failing handler test**

Create `internal/platform/httpserver/integrations_test.go`:

```go
package httpserver

import (
	"bytes"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type recordSink struct {
	applied int
}

func (s *recordSink) Apply(_ context.Context, _ identitydomain.Principal, records []integrationdomain.ExternalRecord) error {
	s.applied += len(records)
	return nil
}

func TestIntegrationImportEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExternalImport: {},
		},
	}
	sink := &recordSink{}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		IntegrationSink: sink,
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("snapshot", "snapshot.ndjson")
	if err != nil { t.Fatal(err) }
	part.Write([]byte(`{"kind":"ASSET",...}` + "\n"))
	writer.WriteField("system", "CMMS-X")
	writer.WriteField("instance", "import-test")
	writer.WriteField("version", "1.0")
	writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if sink.applied == 0 {
		t.Fatal("sink was not applied")
	}
}
```

Add a second test `TestIntegrationImportForbidden` with a principal lacking `PermissionExternalImport` asserting `403`. The record in the fixture must satisfy `ExternalRecord.Validate` (use a full record with `organization_id: "o1"`, `site_id: "11111111-1111-4111-8111-111111111111"`, `external_system: "CMMS-X"`, `external_id: "a-1"`, `observed_at`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestIntegrationImport -v`
Expected: FAIL — route does not exist (404).

- [ ] **Step 3: Implement the handler**

Create `internal/platform/httpserver/integrations.go`:

```go
package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/ndjson"
	"github.com/go-chi/chi/v5"
)

const maxImportBytes = 10 << 20

func mountIntegrationRoutes(router chi.Router, sink integrationapp.ProjectionSink) {
	router.Post("/integrations/imports", runIntegrationImport(sink))
}
```

(Finish the implementation: `runIntegrationImport` reads the multipart form (`r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)`, `r.ParseMultipartForm(maxImportBytes)`), gets `snapshot` via `r.FormFile`, copies to `os.CreateTemp` (defer remove), reads `system`/`instance`/`version` form values (reject when empty with a 400 `Problem`), builds `ndjson.New(tempPath, os.TempDir(), ConnectorIdentity{...ReadOnly: true, Capabilities: []{CapabilityReadAssets}})`, loops `Importer{Connector, Sink: sink}.Pull(ctx, principal, "", 100)` until `Complete`, accumulating `imported := len(page.Records)`, returns `200` with `{"imported": n, "next_cursor": ..., "complete": true}`. Map `integrationapp.ErrForbidden` → 403 and `integrationapp.ErrInvalid` + connector errors → 400 via `writeProblem`.)

- [ ] **Step 4: Wire the route and dependency**

In `internal/platform/httpserver/router.go`: add `IntegrationSink integrationapp.ProjectionSink` to `Dependencies` (import `integrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"`) and `mountIntegrationRoutes(api, dependencies.IntegrationSink)` in the `/api/v1` route group. In `cmd/api/main.go`, add `IntegrationSink: integrationpostgres.Sink{Pool: pool, IDs: idGenerator, Clock: systemClock, Audit: audit.Sink{}}` to the `Dependencies` literal and import `integrationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/integration/adapter/postgres"`.

- [ ] **Step 5: Update OpenAPI**

Add to `api/openapi.yaml`:

```yaml
  /api/v1/integrations/imports:
    post:
      summary: Import an EAM/CMMS NDJSON snapshot
      operationId: runIntegrationImport
      security:
        - sessionCookie: []
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              required: [snapshot, system, instance, version]
              properties:
                snapshot:
                  type: string
                  format: binary
                system: {type: string}
                instance: {type: string}
                version: {type: string}
      responses:
        '200':
          description: Import summary
          content:
            application/json:
              schema:
                type: object
                properties:
                  imported: {type: integer}
                  next_cursor: {type: string, nullable: true}
                  complete: {type: boolean}
        '400': {$ref: '#/components/responses/Problem'}
        '403': {$ref: '#/components/responses/Problem'}
```

Verify the exact `sessionCookie` security scheme name and `Problem` component reference in the existing spec and reuse them.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 7: Commit**

```bash
git add internal/platform/httpserver/router.go internal/platform/httpserver/integrations.go internal/platform/httpserver/integrations_test.go cmd/api/main.go api/openapi.yaml
git commit -m "feat: add admin API route for EAM/CMMS snapshot imports"
```

---

### Task B4: Extend CI with the import CLI end-to-end step

**Files:**
- Modify: `.github/workflows/ci.yml` (the "Test" step block at `:44-49`)

**Interfaces:**
- Consumes: `cmd/import` (Task B2) and the test DB/site seeded by the existing migration step.

- [ ] **Step 1: Add the CI step**

Insert a new step after the existing "Test" step (which already sets `TEST_DATABASE_URL` and `TEST_S3_ENDPOINT`):

```yaml
      - name: Import fixture snapshot through the CLI
        env:
          TEST_DATABASE_URL: postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable
        run: |
          SITE_ID=$(psql "$TEST_DATABASE_URL" -Atc "SELECT m.site_id FROM memberships m JOIN principals p ON p.id = m.principal_id WHERE p.external_subject = 'seed-admin' AND m.role = 'Administrator' LIMIT 1")
          go run ./cmd/import -snapshot test/fixtures/import-p302.ndjson \
            -site-id "$SITE_ID" -database-url "$TEST_DATABASE_URL" \
            -external-subject seed-admin
```

(If `psql` is not available in the runner image, replace the `SITE_ID` lookup with a `python3` one-liner using `psycopg` only if already installed; otherwise run the CLI inside the existing "Test" job by having the seed step print the site ID. Match the runner's available tooling — the repo already uses `docker compose`, `go`, and `python3`.)

- [ ] **Step 2: Verify the workflow YAML parses**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('OK')"`
Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: run the EAM/CMMS import CLI end-to-end"
```

---

## Part C — Env-selectable AI provider adapters

### Task C1: AI config section and validation

**Files:**
- Modify: `internal/platform/config/config.go` (`Config` struct at `:22`, `Load` at `:95`, validation at `:215`)
- Modify: `internal/platform/config/config_test.go` (existing tests may assert the struct; extend, don't break)

**Interfaces:**
- Consumes: existing `env()`, `envInt()` helpers.
- Produces: `config.AI{StructuredProvider, EmbeddingProvider, Endpoint, APIKey, Model, ModelVersion, AnthropicAPIKey, AnthropicModel, EmbeddingEndpoint, EmbeddingModel, EmbeddingModelVersion string}` field `AI` on `Config`; validation errors for bad values.

- [ ] **Step 1: Write the failing test**

Add to `internal/platform/config/config_test.go` (reuse the existing `validConfig(role Role) Config` helper at the bottom of that file):

```go
func TestValidateRejectsUnknownStructuredProvider(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected structured provider validation error")
	}
}

func TestValidateRejectsUnknownEmbeddingProvider(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.EmbeddingProvider = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected embedding provider validation error")
	}
}

func TestValidateRequiresOpenAIModel(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "openai"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing AI_MODEL error")
	}
}

func TestValidateRequiresAnthropicKeyAndModel(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "anthropic"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing ANTHROPIC_API_KEY error")
	}
}

func TestLoadDefaultsStructuredProviderToDeterministic(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("S3_BUCKET", "test-bucket")
	cfg, err := Load(RoleWorker)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AI.StructuredProvider != "deterministic" {
		t.Fatalf("StructuredProvider = %q, want deterministic", cfg.AI.StructuredProvider)
	}
	if cfg.AI.EmbeddingProvider != "deterministic" {
		t.Fatalf("EmbeddingProvider = %q, want deterministic", cfg.AI.EmbeddingProvider)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/config/ -run TestAI -v`
Expected: FAIL.

- [ ] **Step 3: Implement the config section**

Add to `Config`:

```go
	AI AI
```

Add to the `Load` literal (alongside `Transcription`) the section shown in Step 3, and add the validation to `func (c Config) Validate()` (the method at `config.go:164` that accumulates `errs` and returns `errors.Join(errs...)`; the transcription block sits at `config.go:215-228`).

Define:

```go
type AI struct {
	// StructuredProvider selects the structured-output capability:
	// deterministic | openai | anthropic. Default deterministic.
	StructuredProvider string
	// EmbeddingProvider selects the embedding capability:
	// deterministic | openai. Default deterministic.
	EmbeddingProvider string
	Endpoint              string
	APIKey                string
	Model                 string
	ModelVersion          string
	AnthropicAPIKey       string
	AnthropicModel        string
	AnthropicModelVersion string
	EmbeddingEndpoint     string
	EmbeddingModel        string
	EmbeddingModelVersion string
}
```

In `Load`, add defaults and env resolution:

```go
		AI: AI{
			StructuredProvider:     strings.ToLower(strings.TrimSpace(env("STRUCTURED_PROVIDER", "deterministic"))),
			EmbeddingProvider:      strings.ToLower(strings.TrimSpace(env("EMBEDDING_PROVIDER", "deterministic"))),
			Endpoint:               env("AI_ENDPOINT", ""),
			APIKey:                 env("AI_API_KEY", ""),
			Model:                  env("AI_MODEL", ""),
			ModelVersion:           env("AI_MODEL_VERSION", ""),
			AnthropicAPIKey:        env("ANTHROPIC_API_KEY", ""),
			AnthropicModel:         env("ANTHROPIC_MODEL", ""),
			AnthropicModelVersion:  env("ANTHROPIC_MODEL_VERSION", ""),
			EmbeddingEndpoint:      env("EMBEDDING_ENDPOINT", ""),
			EmbeddingModel:         env("EMBEDDING_MODEL", ""),
			EmbeddingModelVersion:  env("EMBEDDING_MODEL_VERSION", ""),
		},
```

(Place alongside the other sections in the `Load` literal; verify the exact insertion point and existing field names.) Add validation to `Validate()`, modeled on the `Transcription` block (`config.go:215-228`):

```go
	if c.AI.StructuredProvider != "deterministic" &&
		c.AI.StructuredProvider != "openai" &&
		c.AI.StructuredProvider != "anthropic" {
		errs = append(errs, errors.New("STRUCTURED_PROVIDER must be deterministic, openai, or anthropic"))
	}
	if c.AI.EmbeddingProvider != "deterministic" &&
		c.AI.EmbeddingProvider != "openai" {
		errs = append(errs, errors.New("EMBEDDING_PROVIDER must be deterministic or openai"))
	}
	if c.AI.StructuredProvider == "openai" {
		if strings.TrimSpace(c.AI.Model) == "" {
			errs = append(errs, errors.New("AI_MODEL is required when STRUCTURED_PROVIDER is openai"))
		}
		if c.AI.Endpoint != "" {
			if endpoint, err := url.Parse(c.AI.Endpoint); err != nil || !endpoint.IsAbs() ||
				(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
				errs = append(errs, errors.New("AI_ENDPOINT must be an absolute HTTP(S) URL"))
			}
		}
	}
	if c.AI.StructuredProvider == "anthropic" {
		if strings.TrimSpace(c.AI.AnthropicAPIKey) == "" {
			errs = append(errs, errors.New("ANTHROPIC_API_KEY is required when STRUCTURED_PROVIDER is anthropic"))
		}
		if strings.TrimSpace(c.AI.AnthropicModel) == "" {
			errs = append(errs, errors.New("ANTHROPIC_MODEL is required when STRUCTURED_PROVIDER is anthropic"))
		}
	}
	if c.AI.EmbeddingProvider == "openai" {
		if strings.TrimSpace(c.AI.EmbeddingModel) == "" {
			errs = append(errs, errors.New("EMBEDDING_MODEL is required when EMBEDDING_PROVIDER is openai"))
		}
		if c.AI.EmbeddingEndpoint != "" {
			if endpoint, err := url.Parse(c.AI.EmbeddingEndpoint); err != nil || !endpoint.IsAbs() ||
				(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
				errs = append(errs, errors.New("EMBEDDING_ENDPOINT must be an absolute HTTP(S) URL"))
			}
		}
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/platform/config/ && go vet ./... && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/config
git commit -m "feat: add env-selectable AI provider configuration"
```

### Task C2: OpenAI-compatible structured provider

**Files:**
- Create: `internal/skawld/http_structured_provider.go`
- Create: `internal/skawld/http_structured_provider_test.go`

**Interfaces:**
- Consumes: `GenerateRequest`, `GenerateResponse`, `ProviderMetadata`, `ErrInvalidOutput` (all in `internal/skawld/routing.go`).
- Produces: `HTTPStructuredConfig{Endpoint, APIKey, Provider, Model, ModelVersion string}`; `NewHTTPStructuredProvider(config HTTPStructuredConfig, client *http.Client) (*HTTPStructuredProvider, error)`; `(*HTTPStructuredProvider) Generate(context.Context, GenerateRequest) (GenerateResponse, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/skawld/http_structured_provider_test.go` (httptest server, modeled on `http_transcription_provider_test.go` — read it first and mirror its helper style):

- `TestHTTPStructuredProviderGenerate` — fake server returns `{"choices":[{"message":{"content":"{\"status\":\"INFORMATIONAL\"}"}}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`; assert `Output` decodes to `map[string]any` with `status == "INFORMATIONAL"` and `Metadata.Provider == "openai"`.
- `TestHTTPStructuredProviderRejectsEmptyContent` — content `""` → `ErrInvalidOutput`.
- `TestHTTPStructuredProviderRejectsNonJSON` — content `"not json"` → `ErrInvalidOutput`.
- `TestHTTPStructuredProviderRejectsNon2xx` — server returns 500 → error containing `status 500`.
- `TestHTTPStructuredProviderConfigValidation` — bad endpoint / missing model / nil client → constructor error.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skawld/ -run TestHTTPStructuredProvider -v`
Expected: FAIL — file does not exist.

- [ ] **Step 3: Implement the provider**

Create `internal/skawld/http_structured_provider.go`:

```go
package skawld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPStructuredConfig struct {
	Endpoint     string
	APIKey       string
	Provider     string
	Model        string
	ModelVersion string
}

type HTTPStructuredProvider struct {
	endpoint string
	apiKey   string
	metadata ProviderMetadata
	client   *http.Client
}

func NewHTTPStructuredProvider(
	config HTTPStructuredConfig,
	client *http.Client,
) (*HTTPStructuredProvider, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || !endpoint.IsAbs() ||
		(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("structured provider endpoint must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(config.Provider) == "" ||
		strings.TrimSpace(config.Model) == "" ||
		strings.TrimSpace(config.ModelVersion) == "" {
		return nil, errors.New("structured provider model metadata is required")
	}
	if client == nil {
		return nil, errors.New("structured provider HTTP client is required")
	}
	return &HTTPStructuredProvider{
		endpoint: endpoint.String(), apiKey: config.APIKey, client: client,
		metadata: ProviderMetadata{
			Provider: config.Provider, Model: config.Model,
			ModelVersion: config.ModelVersion,
		},
	}, nil
}

func (p *HTTPStructuredProvider) Generate(
	ctx context.Context,
	request GenerateRequest,
) (GenerateResponse, error) {
	systemPrompt := fmt.Sprintf(
		"Return exactly one JSON object for capability %q at schema version %s. "+
			"Do not include markdown fences or commentary.",
		request.Capability, request.SchemaVersion,
	)
	userPrompt := fmt.Sprintf(
		"Prompt version: %s\nEvidence:\n%s\nContext:\n%s",
		request.PromptVersion, joinEvidence(request.Evidence), string(request.Context),
	)
	payload := map[string]any{
		"model": p.metadata.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(httpRequest)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("structured provider request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return GenerateResponse{}, fmt.Errorf(
			"structured provider status %d: %s",
			response.StatusCode, strings.TrimSpace(string(limited)),
		)
	}
	var value struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&value); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode structured provider response: %w", err)
	}
	content := strings.TrimSpace(value.Choices[0].Message.Content)
	if len(value.Choices) == 0 || content == "" {
		return GenerateResponse{}, ErrInvalidOutput
	}
	if !json.Valid([]byte(content)) {
		return GenerateResponse{}, ErrInvalidOutput
	}
	return GenerateResponse{
		Output: json.RawMessage(content),
		Metadata: ProviderMetadata{
			Provider: p.metadata.Provider, Model: p.metadata.Model,
			ModelVersion: p.metadata.ModelVersion,
			TokensIn:     value.Usage.PromptTokens,
			TokensOut:    value.Usage.CompletionTokens,
		},
	}, nil
}
```

Note: `joinEvidence` already exists in `internal/skawld/deterministic_provider.go:157` (same package). Guard the `value.Choices[0]` access — check `len(value.Choices) == 0` *before* indexing (reorder so the length check comes first; the snippet above checks `content` after indexing, so fix the order in your implementation: `if len(value.Choices) == 0 { return ..., ErrInvalidOutput }` then read content).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/skawld/ -run TestHTTPStructuredProvider -v && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/skawld/http_structured_provider.go internal/skawld/http_structured_provider_test.go
git commit -m "feat: add OpenAI-compatible structured output provider"
```

### Task C3: OpenAI-compatible embedding provider

**Files:**
- Create: `internal/skawld/http_embedding_provider.go`
- Create: `internal/skawld/http_embedding_provider_test.go`

**Interfaces:**
- Consumes: `EmbeddingModel`, `EmbeddingProvider` (routing.go).
- Produces: `HTTPEmbeddingConfig{Endpoint, APIKey, Provider, Model, ModelVersion string}`; `NewHTTPEmbeddingProvider(config HTTPEmbeddingConfig, client *http.Client) (*HTTPEmbeddingProvider, error)`; `Model() EmbeddingModel`; `Embed(context.Context, []string) ([][]float32, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/skawld/http_embedding_provider_test.go`:
- `TestHTTPEmbeddingProviderEmbed` — fake server returns `{"data":[{"embedding":[0.1,0.2,0.3]}],"model":"text-embedding-3-small"}`; assert one vector of 3 floats and `Model().Metric == "COSINE"`, `Dimensions == 3`.
- `TestHTTPEmbeddingProviderRejectsEmptyInput` — `Embed(ctx, []string{})` → error.
- `TestHTTPEmbeddingProviderRejectsInconsistentDimensions` — two responses with different lengths → error (the provider returns `ErrInvalidOutput`).
- Config validation test (bad endpoint, nil client, missing metadata).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skawld/ -run TestHTTPEmbeddingProvider -v`
Expected: FAIL.

- [ ] **Step 3: Implement the provider**

Create `internal/skawld/http_embedding_provider.go` (mirror the structured provider's request/response plumbing):

- Constructor validates endpoint (absolute HTTP(S)), metadata, client — same shape as `NewHTTPStructuredProvider`.
- `Model()` returns `EmbeddingModel{Provider: config.Provider, Model: config.Model, ModelVersion: config.ModelVersion, Metric: "COSINE"}` (dimensions unknown until the first response; set `Dimensions: 0` in `Model()` and populate it from the first response — document this in the method comment; the knowledge store uses the returned vectors, not `Model().Dimensions`, for storage, so a 0 value is safe).
- `Embed` posts `{"input": inputs, "model": model}`; decodes `{"data":[{"embedding":[...]}],"model":...}`; rejects `len(inputs) == 0`; validates each embedding length matches the first (`ErrInvalidOutput` otherwise); rejects a `data` count that differs from `len(inputs)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/skawld/ -run TestHTTPEmbeddingProvider -v && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/skawld/http_embedding_provider.go internal/skawld/http_embedding_provider_test.go
git commit -m "feat: add OpenAI-compatible embedding provider"
```

### Task C4: Anthropic structured provider

**Files:**
- Create: `internal/skawld/anthropic_structured_provider.go`
- Create: `internal/skawld/anthropic_structured_provider_test.go`

**Interfaces:**
- Consumes: `StructuredProvider` contract (routing.go).
- Produces: `AnthropicStructuredConfig{APIKey, Model, ModelVersion string}`; `NewAnthropicStructuredProvider(config, client) (*AnthropicStructuredProvider, error)`; `Generate(context.Context, GenerateRequest) (GenerateResponse, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/skawld/anthropic_structured_provider_test.go`:
- `TestAnthropicStructuredProviderGenerate` — fake server asserts the `x-api-key` and `anthropic-version` headers and returns `{"content":[{"type":"text","text":"{\"status\":\"INFORMATIONAL\"}"}],"usage":{"input_tokens":5,"output_tokens":2}}`; assert output decodes and `Metadata.Provider == "anthropic"`.
- `TestAnthropicStructuredProviderRejectsEmptyContent` — `content` array with empty `text` → `ErrInvalidOutput`.
- `TestAnthropicStructuredProviderRejectsNonJSON` — text `"not json"` → `ErrInvalidOutput`.
- Constructor validation: empty API key/model → error; nil client → error.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skawld/ -run TestAnthropicStructuredProvider -v`
Expected: FAIL.

- [ ] **Step 3: Implement the provider**

Create `internal/skawld/anthropic_structured_provider.go`:

- Endpoint fixed at `https://api.anthropic.com/v1/messages` (constant `anthropicMessagesEndpoint`).
- Constructor requires non-empty API key + model + model version and a non-nil client; `Metadata.Provider = "anthropic"`.
- `Generate` posts:

```json
{
  "model": "<model>",
  "max_tokens": 2048,
  "system": "Return exactly one JSON object ... Do not include markdown fences.",
  "messages": [{"role": "user", "content": "<prompt with prompt version, evidence, context>"}]
}
```

Headers: `Content-Type: application/json`, `x-api-key: <key>`, `anthropic-version: 2023-06-01`.
- Response decode:

```go
	var value struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
```

- Validate: non-2xx → error with status; `len(value.Content) == 0` or first text block empty → `ErrInvalidOutput`; `!json.Valid([]byte(text))` → `ErrInvalidOutput`. Return `GenerateResponse{Output: json.RawMessage(text), Metadata: ProviderMetadata{Provider: "anthropic", Model, ModelVersion, TokensIn: value.Usage.InputTokens, TokensOut: value.Usage.OutputTokens}}`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/skawld/ -run TestAnthropicStructuredProvider -v && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/skawld/anthropic_structured_provider.go internal/skawld/anthropic_structured_provider_test.go
git commit -m "feat: add Anthropic structured output provider"
```

### Task C5: Env-driven composition helper and wiring

**Files:**
- Create: `internal/skawld/providers.go`
- Create: `internal/skawld/providers_test.go`
- Modify: `cmd/api/main.go:120-140` (router/modelRouter construction)
- Modify: `cmd/worker/main.go` (embedding provider + processor wiring)
- Modify: `cmd/seed/main.go:92,327-330` (embedding provider + router)

**Interfaces:**
- Consumes: `config.AI` (Task C1), `HTTPStructuredProvider`, `HTTPEmbeddingProvider`, `AnthropicStructuredProvider`, `DeterministicProvider`, `DeterministicEmbeddingProvider`.
- Produces: `AIConfig{StructuredProvider, EmbeddingProvider, StructuredEndpoint, StructuredAPIKey, StructuredModel, StructuredModelVersion, AnthropicAPIKey, AnthropicModel, EmbeddingEndpoint, EmbeddingModel, EmbeddingModelVersion string}`; `BuildProviders(cfg AIConfig, client *http.Client) (map[Capability]StructuredProvider, EmbeddingProvider, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/skawld/providers_test.go`:

- `TestBuildProvidersDeterministicDefaults` — empty `AIConfig` → router map has all three capabilities, embedding provider is a `DeterministicEmbeddingProvider`; `Generate` for `CapabilityRecommendation` with empty evidence returns `INSUFFICIENT_EVIDENCE` output.
- `TestBuildProvidersOpenAI` — config selects `openai` → provider is `*HTTPStructuredProvider`, embedding is `*HTTPEmbeddingProvider` (assert via type assertion).
- `TestBuildProvidersAnthropic` — config selects `anthropic` → provider is `*AnthropicStructuredProvider`.
- `TestBuildProvidersRejectsUnknown` — `StructuredProvider: "bogus"` → error; `EmbeddingProvider: "bogus"` → error.
- `TestBuildProvidersOpenAIWithoutModel` — `openai` with empty model → error (fail closed).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skawld/ -run TestBuildProviders -v`
Expected: FAIL — no `providers.go`.

- [ ] **Step 3: Implement the helper**

Create `internal/skawld/providers.go`:

```go
package skawld

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ProviderDeterministic = "deterministic"
	ProviderOpenAI        = "openai"
	ProviderAnthropic     = "anthropic"
)

type AIConfig struct {
	StructuredProvider     string
	EmbeddingProvider      string
	StructuredEndpoint     string
	StructuredAPIKey       string
	StructuredModel        string
	StructuredModelVersion string
	AnthropicAPIKey        string
	AnthropicModel         string
	AnthropicModelVersion  string
	EmbeddingEndpoint      string
	EmbeddingModel         string
	EmbeddingModelVersion  string
}

func BuildProviders(
	cfg AIConfig,
	client *http.Client,
) (map[Capability]StructuredProvider, EmbeddingProvider, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	structuredProvider, err := buildStructuredProvider(cfg, client)
	if err != nil {
		return nil, nil, err
	}
	embeddingProvider, err := buildEmbeddingProvider(cfg, client)
	if err != nil {
		return nil, nil, err
	}
	return map[Capability]StructuredProvider{
		CapabilityRecommendation: structuredProvider,
		CapabilityReportDraft:    structuredProvider,
		CapabilityShiftHandover:  structuredProvider,
	}, embeddingProvider, nil
}

func buildStructuredProvider(cfg AIConfig, client *http.Client) (StructuredProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.StructuredProvider)) {
	case "", ProviderDeterministic:
		return DeterministicProvider{}, nil
	case ProviderOpenAI:
		if strings.TrimSpace(cfg.StructuredModel) == "" ||
			strings.TrimSpace(cfg.StructuredModelVersion) == "" {
			return nil, errors.New("openai structured provider requires model and model version")
		}
		return NewHTTPStructuredProvider(HTTPStructuredConfig{
			Endpoint: cfg.StructuredEndpoint, APIKey: cfg.StructuredAPIKey,
			Provider: ProviderOpenAI, Model: cfg.StructuredModel,
			ModelVersion: cfg.StructuredModelVersion,
		}, client)
	case ProviderAnthropic:
		return NewAnthropicStructuredProvider(AnthropicStructuredConfig{
			APIKey: cfg.AnthropicAPIKey, Model: cfg.AnthropicModel,
			ModelVersion: cfg.AnthropicModelVersion,
		}, client)
	default:
		return nil, fmt.Errorf("unsupported structured provider %q", cfg.StructuredProvider)
	}
}

func buildEmbeddingProvider(cfg AIConfig, client *http.Client) (EmbeddingProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider)) {
	case "", ProviderDeterministic:
		return DeterministicEmbeddingProvider{Dimensions: 64}, nil
	case ProviderOpenAI:
		if strings.TrimSpace(cfg.EmbeddingModel) == "" ||
			strings.TrimSpace(cfg.EmbeddingModelVersion) == "" {
			return nil, errors.New("openai embedding provider requires model and model version")
		}
		return NewHTTPEmbeddingProvider(HTTPEmbeddingConfig{
			Endpoint: cfg.EmbeddingEndpoint, APIKey: cfg.StructuredAPIKey,
			Provider: ProviderOpenAI, Model: cfg.EmbeddingModel,
			ModelVersion: cfg.EmbeddingModelVersion,
		}, client)
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", cfg.EmbeddingProvider)
	}
}
```

`AnthropicStructuredConfig{APIKey, Model, ModelVersion string}` (Task C4) aligns with `AIConfig.AnthropicModelVersion`; `HTTPStructuredConfig{Endpoint, APIKey, Provider, Model, ModelVersion string}` and `HTTPEmbeddingConfig{Endpoint, APIKey, Provider, Model, ModelVersion string}` align with the fields used above.

- [ ] **Step 4: Wire cmd/api/main.go**

Replace the hard-coded construction at `cmd/api/main.go:125-130`:

```go
	embeddingProvider, structuredProviders, err := skawld.BuildProviders(
		skawld.AIConfig{
			StructuredProvider:     cfg.AI.StructuredProvider,
			EmbeddingProvider:      cfg.AI.EmbeddingProvider,
			StructuredEndpoint:     cfg.AI.Endpoint,
			StructuredAPIKey:       cfg.AI.APIKey,
			StructuredModel:        cfg.AI.Model,
			StructuredModelVersion: cfg.AI.ModelVersion,
			AnthropicAPIKey:        cfg.AI.AnthropicAPIKey,
			AnthropicModel:         cfg.AI.AnthropicModel,
			AnthropicModelVersion:  cfg.AI.AnthropicModelVersion,
			EmbeddingEndpoint:      cfg.AI.EmbeddingEndpoint,
			EmbeddingModel:         cfg.AI.EmbeddingModel,
			EmbeddingModelVersion:  cfg.AI.EmbeddingModelVersion,
		},
		&http.Client{Timeout: 30 * time.Second},
	)
	if err != nil {
		logger.Error("AI provider startup failed", "error", err)
		os.Exit(1)
	}
	modelRouter := skawld.Router{Providers: structuredProviders}
```

(Keep the variable names the surrounding code uses: `embeddingProvider` is passed into `knowledgeStore`, `modelRouter` into `Copilot`/`Reports`/`Handovers`. Adjust the wiring so the two return values land in the right places. The knowledgeStore currently receives `Embeddings: embeddingProvider`; the router receives the provider map.)

- [ ] **Step 5: Wire cmd/worker/main.go and cmd/seed/main.go**

Worker: replace `skawld.DeterministicEmbeddingProvider{Dimensions: 64}` (the `ingest.Processor.Embeddings` field) with `embeddingProvider, _, err := skawld.BuildProviders(...)` (embedding only; ignore the structured map) using the same `AIConfig` mapping, and fail startup on error. Seed: replace the `DeterministicEmbeddingProvider` literal and the `Router{Providers: ...}` literal with `BuildProviders` — for seed, pass an empty `AIConfig` so it always uses deterministic providers (the demo must stay credential-free); the seed keeps `skawld.DeterministicProvider{}` behavior through the helper defaults.

Deployment note (do not implement now): switching `EMBEDDING_PROVIDER` after documents are already embedded mixes vector dimensions in the same store, so a provider switch must be paired with re-ingestion/re-embedding. Document this in the `ReadMe` deployment section if you touch it; no code change is required in this task.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/skawld/ ./internal/platform/config/ && go vet ./... && go build ./...`
Expected: PASS; `gofmt -l .` prints nothing.

- [ ] **Step 7: Commit**

```bash
git add internal/skawld/providers.go internal/skawld/providers_test.go cmd/api/main.go cmd/worker/main.go cmd/seed/main.go
git commit -m "feat: select AI providers by environment with deterministic fallback"
```

---

## Verification (run before finishing)

- `gofmt -l .` → empty.
- `go vet ./...` → clean.
- `go build ./...` → clean.
- `go test ./...` → pass (unit + handler tests; DB tests skip without env).
- `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test -race -count=1 ./...` → pass (includes the new sink, CLI, and pagination-store paths).
- `go run ./cmd/eval -dataset test/evaldata/pilot-v1.json` → gates pass.
- OpenAPI still validates: `python3 -c "import yaml,sys; yaml.safe_load(open('api/openapi.yaml')); print('OK')"`.
- `git status` shows only the intended files; no SDK version change in `go.mod`.
