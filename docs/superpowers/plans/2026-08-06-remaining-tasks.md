# Remaining Tasks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the engineering-closeable backlog from `docs/superpowers/specs/2026-08-06-remaining-tasks-design.md` (A), add secret-gated AI provider contract tests and deployment runbooks (B), run Flutter quality gates on native CI and add signing-gated release steps (C), and document the pilot-customer decision/drill checklist (D).

**Architecture:** Four independent workstreams, executed in order A → B → C → D. A fixes OpenAPI drift, hardens error-sentinel tests, adds negative/edge coverage, wires cursor "load more" into the six console list pages through a shared hook, and audits the fictional corpus. B adds env-gated live contract tests for the four HTTP provider adapters plus runbooks. C extends the native packaging job with Flutter quality gates and adds a secrets-gated signing workflow. D is a documentation-only checklist.

**Tech Stack:** Go 1.x + pgx (backend tests), Vitest/React (web load-more), GitHub Actions (native CI + signing), Flutter 3.44.8 (quality gates).

## Global Constraints

- skawld-sdk-go stays pinned to v0.2.0; no skawld-sdk-go import outside `internal/skawld` and the SDK contract tests.
- No new runtime dependencies; `go.mod` and `web/package.json` are unchanged. Signing steps use pinned GitHub Actions only.
- Env vars unprefixed (`AI_ENDPOINT`, `TRANSCRIPTION_ENDPOINT`, etc.); contract tests self-skip when credentials are absent, so CI stays green without them.
- The deterministic provider remains the default; CI and seed stay credential-free.
- `api/openapi.yaml` remains the single contract source; every serialized field is documented.
- Web: `api.ts` list functions keep `.then((list) => list.items)` callers working (ListPage extends ListResponse); the load-more logic is shared, not copy-pasted per page.
- TDD: failing test first, then implementation, then passing test, then commit. `gofmt -l .` empty, `go vet ./...` clean.
- No comments unless they explain why; no em dashes in code.
- Docs live under `docs/runbooks/`; follow the existing runbook style.

---

## Part A — Engineering-closeable backlog

### Task 1: Fix OpenAPI drift (Asset.created_at, Execution.updated_at)

**Files:**
- Modify: `api/openapi.yaml` (`Asset:` and `Execution:` component schemas)

**Interfaces:**
- `Asset` Go struct serializes `created_at` unconditionally (`internal/asset/application/service.go`).
- `Execution` Go struct serializes `updated_at` unconditionally (`internal/execution/application/service.go`).

- [ ] **Step 1: Locate the two schemas**

Run: `grep -n "^    Asset:" api/openapi.yaml && grep -n "^    Execution:" api/openapi.yaml`
Expected: two line numbers. The `Asset` schema's `required` is currently `[id, organization_id, site_id, tag, name, class, status, source_of_truth, version, components]` and its last property is `criticality`. The `Execution` schema (currently at line ~1919) has `required: [id, organization_id, site_id, asset_id, purpose, state, version, steps, prerequisites, measurements, observations, actions, decisions]`.

- [ ] **Step 2: Add created_at to the Asset schema**

In the `Asset:` schema, change `required` to end with `..., version, components, created_at]` and add after the `version` property:

```yaml
        created_at: {type: string, format: date-time}
```

- [ ] **Step 3: Add updated_at to the Execution schema**

In the `Execution:` schema, change `required` to end with `..., actions, decisions, updated_at]` and add after the `version` property:

```yaml
        updated_at: {type: string, format: date-time}
```

- [ ] **Step 4: Verify**

Run: `python3 -c "import yaml; d=yaml.safe_load(open('api/openapi.yaml')); s=d['components']['schemas']; assert 'created_at' in s['Asset']['properties']; assert 'updated_at' in s['Execution']['properties']; print('OK')"`
Expected: `OK`.

- [ ] **Step 5: Commit**

```bash
git add api/openapi.yaml
git commit -m "docs: document created_at and updated_at in the API contract"
```

### Task 2: Harden error-sentinel assertions in provider tests

**Files:**
- Modify: `internal/skawld/http_structured_provider_test.go`
- Modify: `internal/skawld/http_embedding_provider_test.go`
- Modify: `internal/skawld/anthropic_structured_provider_test.go`

**Interfaces:**
- `ErrInvalidOutput` (routing.go), `ErrInvalid` (integration/application).

- [ ] **Step 1: Write the failing assertions**

In `http_structured_provider_test.go`, change `TestHTTPStructuredProviderRejectsEmptyContent` and `TestHTTPStructuredProviderRejectsNonJSON` from `if err == nil { t.Fatal(...) }` to:

```go
	if _, err := provider.Generate(...); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error = %v, want ErrInvalidOutput", err)
	}
```

Add `"errors"` to the test imports. Do the same in `http_embedding_provider_test.go` for `TestHTTPEmbeddingProviderRejectsInconsistentDimensions` and in `anthropic_structured_provider_test.go` for `TestAnthropicStructuredProviderRejectsEmptyContent` and `TestAnthropicStructuredProviderRejectsNonJSON`.

- [ ] **Step 2: Run the tests to verify they pass (and would fail on wrong sentinels)**

Run: `go test ./internal/skawld/ -run "TestHTTPStructuredProvider|TestHTTPEmbeddingProvider|TestAnthropicStructuredProvider" -v`
Expected: all PASS; each assertion now pins the sentinel (a regression returning a plain error would fail).

- [ ] **Step 3: Commit**

```bash
git add internal/skawld/*_test.go
git commit -m "test: pin provider error sentinels in rejection tests"
```

### Task 3: Provider edge-case tests

**Files:**
- Modify: `internal/skawld/http_structured_provider_test.go`
- Modify: `internal/skawld/anthropic_structured_provider_test.go`
- Modify: `internal/skawld/http_embedding_provider_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `http_structured_provider_test.go`:

```go
func TestHTTPStructuredProviderRejectsEmptyChoices(t *testing.T) {
	t.Parallel()
	provider := newTestStructuredProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices": []}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error = %v, want ErrInvalidOutput", err)
	}
}
```

Add to `anthropic_structured_provider_test.go`:

```go
func TestAnthropicStructuredProviderRejectsEmptyContentArray(t *testing.T) {
	t.Parallel()
	provider := newTestAnthropicProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": []}`))
	})
	if _, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
	}); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error = %v, want ErrInvalidOutput", err)
	}
}
```

Add to `http_embedding_provider_test.go`:

```go
func TestHTTPEmbeddingProviderRejectsCountMismatch(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"embedding": [0.1, 0.2, 0.3]}]}`)) // 1 result for 2 inputs
	})
	if _, err := provider.Embed(context.Background(), []string{"a", "b"}); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error = %v, want ErrInvalidOutput", err)
	}
}

func TestHTTPEmbeddingProviderRejectsZeroLengthEmbedding(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"embedding": []}]}`))
	})
	if _, err := provider.Embed(context.Background(), []string{"a"}); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("error = %v, want ErrInvalidOutput", err)
	}
}

func TestHTTPEmbeddingProviderRejectsNon2xx(t *testing.T) {
	t.Parallel()
	provider := newTestEmbeddingProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusServiceUnavailable)
	})
	if _, err := provider.Embed(context.Background(), []string{"a"}); err == nil ||
		!strings.Contains(err.Error(), "status 503") {
		t.Fatalf("error = %v, want status 503", err)
	}
}
```

Add `"errors"` and `"strings"` to the embedding test imports as needed.

- [ ] **Step 2: Run the tests to verify they pass**

Run: `go test ./internal/skawld/ -run "TestHTTPStructuredProviderRejectsEmptyChoices|TestAnthropicStructuredProviderRejectsEmptyContentArray|TestHTTPEmbeddingProviderRejects" -v`
Expected: 4/4 PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/skawld/*_test.go
git commit -m "test: cover provider empty-result and count-mismatch guards"
```

### Task 4: List negative-path handler tests (six endpoints)

**Files:**
- Modify (add tests): `internal/platform/httpserver/assets_list_test.go`, `incidents_list_test.go`, `executions_list_test.go`, `documents_list_test.go`, `demonstrations_list_test.go`, `workflows_list_test.go`

**Interfaces:**
- The existing fake stores/gateways from the gap-completion plan (e.g. `listAssetStore`, `listIncidentStore`) are reused; only the handler is exercised.

- [ ] **Step 1: Write the failing tests**

For each of the six endpoints, add three tests using the existing fake store and an admin-reader principal. Example for assets (`assets_list_test.go`):

```go
func TestListAssetsRejectsInvalidPageSize(t *testing.T) {
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
		Assets: assetapp.Service{Store: &listAssetStore{}, Authorities: fakeAuthorities{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/assets?page_size=0", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListAssetsRejectsMalformedCursor(t *testing.T) {
	// same handler; request "/api/v1/assets?cursor=%%%not-base64%%%"
	// assert 400 (ErrInvalid from keyset.Decode).
}

func TestListAssetsForbiddenWithoutPermission(t *testing.T) {
	// principal with only PermissionIncidentRead (no PermissionAssetRead)
	// assert 403.
}
```

Repeat for incidents (fake `listIncidentStore`, `PermissionIncidentRead`), executions (`listExecutionStore`, `PermissionExecutionRead`), documents (`listDocumentStore`, `PermissionKnowledgeRead`), demonstrations (`listDemonstrationGateway`, `PermissionDemonstrationRead`), workflows (`listWorkflowGateway`, `PermissionWorkflowRead`). Use the same principal ID/site constants already in each test file.

- [ ] **Step 2: Run the tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ -run "TestList(Assets|Incidents|Executions|Documents|Demonstrations|Workflows)(RejectsInvalidPageSize|RejectsMalformedCursor|ForbiddenWithoutPermission)" -v`
Expected: 18/18 PASS. (The malformed-cursor and forbidden cases pass because `parsePageSize`/`keyset.Decode` and the service permission checks already enforce them; the tests lock the contract.)

- [ ] **Step 3: Commit**

```bash
git add internal/platform/httpserver/*_list_test.go
git commit -m "test: lock list endpoint error mappings for all paginated lists"
```

### Task 5: Store cursor-predicate integration tests (five endpoints)

**Files:**
- Modify: `internal/execution/adapter/postgres/store_integration_test.go`
- Create (or extend): integration tests for asset, incident, knowledge (documents), demonstration stores, all gated on `TEST_DATABASE_URL`.

**Interfaces:**
- The stores' `List`/`ListDocuments`/gateway `List` methods already take `PageSize`/`Cursor` and return `(items, hasMore, err)`.

- [ ] **Step 1: Write the failing integration tests**

Add to `internal/execution/adapter/postgres/store_integration_test.go` (reuse its seeding helpers; the file already seeds an org/site/execution):

```go
func TestExecutionListCursorPagination(t *testing.T) {
	// seed 3 executions (the existing fixture creates 1; create 2 more via
	// executionStore.Create with distinct idempotency keys), then:
	//   page1, cursor, err := executionStore.List(ctx, admin, Filter{PageSize: 2})
	//   assert len(page1) == 2 && cursor != ""
	//   page2, cursor2, err := executionStore.List(ctx, admin, Filter{PageSize: 2, Cursor: cursor})
	//   assert len(page2) == 1 && cursor2 == ""
	//   assert 3 distinct IDs across the two pages
}
```

Write the same shape for:
- assets: seed 3 assets via `assetpostgres.Store.Create`, page with `Filter{PageSize: 2}`.
- incidents: seed 3 incidents via `incidentpostgres.Store.Create`, page with `Filter{PageSize: 2}`.
- documents: seed 3 documents via `knowledgepostgres.Store.CreateDocument` (each needs a unique title; ingestion is not required for listing), page with `Filter{PageSize: 2}`.
- demonstrations: seed 3 demonstrations via `skawld.DemonstrationGateway.Start` (using `demonstrationTestPool` from `internal/skawld/demonstrations_integration_test.go` — add the test to that package), page with `demonstrationapp.ListFilter{PageSize: 2}`.

Each test asserts `LIMIT page_size+1` truncation (first page has exactly `pageSize` items), `hasMore` correctness, and no skip/duplicate across the cursor boundary.

- [ ] **Step 2: Run the integration tests against the real database**

Run: `TEST_DATABASE_URL=postgres://skawld_app:skawld_app_dev@localhost:5432/skawld?sslmode=disable go test -count=1 -run "ListCursorPagination" ./internal/execution/... ./internal/asset/... ./internal/incident/... ./internal/knowledge/... ./internal/skawld/`
Expected: 5/5 PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/execution internal/asset internal/incident internal/knowledge internal/skawld
git commit -m "test: prove keyset pagination against real stores"
```

### Task 6: cmd/import negative tests

**Files:**
- Modify: `cmd/import/main_test.go`

**Interfaces:**
- `runImport(ctx, logger, args) error` (no DB needed for the flag/subject validation paths).

- [ ] **Step 1: Write the failing tests**

```go
func TestRunImportRequiresFlags(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	args := importArgs{Snapshot: "", SiteID: ""}
	if err := runImport(context.Background(), logger, args); err == nil {
		t.Fatal("expected missing-flag error")
	}
	args = importArgs{Snapshot: "x.ndjson", SiteID: "00000000-0000-0000-0000-000000000000", Limit: 999}
	if err := runImport(context.Background(), logger, args); err == nil {
		t.Fatal("expected out-of-range limit error")
	}
}

func TestRunImportRejectsNonAdministratorSubject(t *testing.T) {
	// gate on TEST_DATABASE_URL; seed org/site/principal WITHOUT an
	// Administrator membership; call runImport with -external-subject
	// matching the non-admin principal; assert the error contains
	// "no administrator principal".
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./cmd/import/ -v`
Expected: PASS (first test is DB-free; second skips without `TEST_DATABASE_URL`).

- [ ] **Step 3: Commit**

```bash
git add cmd/import/main_test.go
git commit -m "test: cover import CLI validation failures"
```

### Task 7: Web — paginated api client functions and shared hook

**Files:**
- Modify: `web/src/api.ts` (six list functions)
- Create: `web/src/console/usePaginatedList.ts`
- Test: `web/src/console/usePaginatedList.test.tsx`

**Interfaces:**
- `ListPage<T> = ListResponse<T> & { next_cursor: string | null; has_more: boolean }` (`web/src/types.ts:356-358`).
- Existing callers use `.then((list) => list.items)` — must keep compiling (ListPage extends ListResponse).
- Produces: `usePaginatedList<T>(fetcher: (params: {page_size: number; cursor?: string}) => Promise<ListPage<T>>, deps: unknown[])` returning `{ items, loading, error, hasMore, loadMore, refetch }`.

- [ ] **Step 1: Write the failing hook test**

Create `web/src/console/usePaginatedList.test.tsx`:

```tsx
import { renderHook, act } from "@testing-library/react";
import { usePaginatedList } from "./usePaginatedList";

describe("usePaginatedList", () => {
  it("walks pages via next_cursor until has_more is false", async () => {
    const pages = [
      { items: [{ id: "1" }], next_cursor: "c1", has_more: true },
      { items: [{ id: "2" }], next_cursor: null, has_more: false },
    ];
    let calls = 0;
    const fetcher = async ({ cursor }: { page_size: number; cursor?: string }) => {
      const page = pages[calls++];
      if (cursor !== (calls === 2 ? "c1" : undefined)) throw new Error("bad cursor");
      return page;
    };
    const { result } = renderHook(() => usePaginatedList(fetcher, []));
    expect(result.current.items).toEqual([{ id: "1" }]);
    expect(result.current.hasMore).toBe(true);
    await act(async () => { await result.current.loadMore(); });
    expect(result.current.items).toEqual([{ id: "1" }, { id: "2" }]);
    expect(result.current.hasMore).toBe(false);
  });
});
```

Run: `cd web && npx vitest run src/console/usePaginatedList.test.tsx`
Expected: FAIL (hook does not exist).

- [ ] **Step 2: Implement the hook**

Create `web/src/console/usePaginatedList.ts`:

```tsx
import { useCallback, useEffect, useRef, useState } from "react";
import type { ListPage } from "../types";

export interface PaginatedResult<T> {
  items: T[];
  loading: boolean;
  error: string | undefined;
  hasMore: boolean;
  loadMore: () => Promise<void>;
  refetch: () => Promise<void>;
}

/**
 * usePaginatedList: cursor-walking list state. Loads the first page,
 * appends subsequent pages on loadMore, and refetches from the start.
 */
export function usePaginatedList<T>(
  fetcher: (params: { page_size: number; cursor?: string }) => Promise<ListPage<T>>,
  deps: unknown[],
  pageSize = 25,
): PaginatedResult<T> {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);
  const [hasMore, setHasMore] = useState(false);
  const cursorRef = useRef<string | undefined>(undefined);
  const seqRef = useRef(0);

  const run = useCallback(async (fromStart: boolean) => {
    const seq = ++seqRef.current;
    setLoading(true);
    setError(undefined);
    try {
      const cursor = fromStart ? undefined : cursorRef.current;
      const page = await fetcher({ page_size: pageSize, cursor });
      if (seq !== seqRef.current) return;
      cursorRef.current = page.next_cursor ?? undefined;
      setHasMore(page.has_more);
      setItems((previous) => (fromStart ? page.items : [...previous, ...page.items]));
    } catch (err) {
      if (seq !== seqRef.current) return;
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (seq === seqRef.current) setLoading(false);
    }
  }, deps); // eslint-disable-line react-hooks/exhaustive-deps

  const loadMore = useCallback(async () => { await run(false); }, [run]);
  const refetch = useCallback(async () => { await run(true); }, [run]);

  useEffect(() => { void run(true); }, [run]);

  return { items, loading, error, hasMore, loadMore, refetch };
}
```

- [ ] **Step 3: Update the six api list functions**

In `web/src/api.ts`, change the six list functions to accept optional paging and return `ListPage<T>`. Add one helper that assembles the full query string from all params so no duplicate `?` can occur:

```ts
function listQuery(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") search.set(key, String(value));
  }
  const suffix = search.toString();
  return suffix ? `?${suffix}` : "";
}
```

```ts
  assets: (paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<Asset>>(`/assets${listQuery({ ...paging })}`),
  incidents: (paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<Incident>>(`/incidents${listQuery({ ...paging })}`),
  documents: (siteID?: string, paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<KnowledgeDocument>>(
      `/documents${listQuery({ site_id: siteID, ...paging })}`
    ),
  listExecutions: (paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<Execution>>(`/executions${listQuery({ ...paging })}`),
  demonstrations: (siteID?: string, paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<Demonstration>>(
      `/demonstrations${listQuery({ site_id: siteID, ...paging })}`
    ),
  workflows: (paging?: { page_size?: number; cursor?: string }) =>
    request<ListPage<WorkflowVersion>>(`/workflows${listQuery({ ...paging })}`),
```

- [ ] **Step 4: Run the web suite**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS. Existing `.then((list) => list.items)` callers keep compiling because `ListPage<T>` includes `items`.

- [ ] **Step 5: Commit**

```bash
git add web/src/api.ts web/src/types.ts web/src/console/usePaginatedList.ts web/src/console/usePaginatedList.test.tsx
git commit -m "feat(web): add paginated list client and cursor-walking hook"
```

### Task 8: Web — load more on the six console list pages

**Files:**
- Modify: `web/src/console/pages/AssetsPage.tsx`, `IncidentsPage.tsx`, `ExecutionsPage.tsx`, `KnowledgePage.tsx`, `DemonstrationsPage.tsx`, `WorkflowsPage.tsx`
- Modify: their `.test.tsx` files where list rendering is asserted

**Interfaces:**
- Consumes: `usePaginatedList` (Task 7) and the updated `api` list functions.
- Produces: a "Load more" button visible when `hasMore` is true, appended items on click.

- [ ] **Step 1: Rewrite AssetsPage to use the hook**

Replace `const assets = useQuery(() => api.assets().then((list) => list.items));` with:

```tsx
  const assets = usePaginatedList((params) => api.assets(params), []);
```

and pass `assets.items`, `assets.loading`, `assets.error` into `AssetsTable`; add after `AssetsTable`:

```tsx
        {assets.hasMore ? (
          <button
            className="secondary-button"
            onClick={() => void assets.loadMore()}
            disabled={assets.loading}
          >
            {t("common.loadMore")}
          </button>
        ) : null}
```

Update `onRetry` to `() => void assets.refetch()`. Add the `common.loadMore` key to `web/src/i18n/messages.ts` (the console i18n source) with a value such as `"Load more"`.

- [ ] **Step 2: Apply the same pattern to the other five pages**

For each of IncidentsPage, ExecutionsPage, KnowledgePage (documents), DemonstrationsPage, WorkflowsPage: replace the `useQuery(() => api.<list>().then((list) => list.items))` with the hook, render items from `items`, and add the same Load-more button. Where a page renders an empty/loading/error state, keep it and source it from the hook fields.

- [ ] **Step 3: Update tests**

For each affected page's `.test.tsx`, the mocked `api` list function must now return `ListPage`-shaped values (add `next_cursor: null, has_more: false` to existing `mockResolvedValue({ items: [...] })` objects, or the pagination assertion fails). Add one test per page asserting the Load-more button appears when `has_more: true` and disappears after loading the final page.

- [ ] **Step 4: Run the web suite**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src
git commit -m "feat(web): load more paginated list items in the console"
```

### Task 9: Fictional corpus verification (docs)

**Files:**
- Modify: `docs/runbooks/demo-data.md`

- [ ] **Step 1: Audit what the seed creates**

Run: `grep -n "Step [0-9]" cmd/seed/main.go | head -20`
Expected: the seed steps (identity, asset + criticality, incident, execution, measurements, observations, recommendation + feedback, report, handover, demonstrations, workflow).

- [ ] **Step 2: Add the verification checklist**

Append to `docs/runbooks/demo-data.md` a "Demo corpus verification" section: for each seed artifact, the UI/API query to confirm it (e.g. `GET /api/v1/documents` shows the approved SOP with ingestion_state READY; `GET /api/v1/incidents` shows the P-302 incident; `GET /api/v1/workflows` shows the published workflow), plus the cleanup SQL already in the runbook. Note any artifact that the audit cannot confirm and report it in the plan's final summary rather than inventing coverage.

- [ ] **Step 3: Commit**

```bash
git add docs/runbooks/demo-data.md
git commit -m "docs: add demo corpus verification checklist"
```

---

## Part B — Deployment gates

### Task 10: Secret-gated provider contract tests

**Files:**
- Create: `internal/skawld/http_structured_provider_contract_test.go`
- Create: `internal/skawld/http_embedding_provider_contract_test.go`
- Create: `internal/skawld/anthropic_structured_provider_contract_test.go`
- Create: `internal/skawld/http_transcription_provider_contract_test.go`

**Interfaces:**
- Adapters already exist (`HTTPStructuredProvider`, `HTTPEmbeddingProvider`, `AnthropicStructuredProvider`, `HTTPTranscriptionProvider`); the tests exercise the real endpoints.

- [ ] **Step 1: Write the failing contract tests**

`http_structured_provider_contract_test.go`:

```go
package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestStructuredProviderContract requires AI_ENDPOINT, AI_API_KEY, AI_MODEL,
// AI_MODEL_VERSION; it performs one real generation against the configured
// deployment adapter. It skips when the credentials are absent so CI stays
// green without them.
func TestStructuredProviderContract(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	key := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")
	version := os.Getenv("AI_MODEL_VERSION")
	if endpoint == "" || key == "" || model == "" || version == "" {
		t.Skip("AI provider credentials not configured")
	}
	provider, err := NewHTTPStructuredProvider(HTTPStructuredConfig{
		Endpoint: endpoint, APIKey: key, Provider: "openai",
		Model: model, ModelVersion: version,
	}, &http.Client{Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	response, err := provider.Generate(context.Background(), GenerateRequest{
		Capability: CapabilityRecommendation, SchemaVersion: "v1",
		PromptVersion: "contract-v1",
		Context:       json.RawMessage(`{"incident":"P-302 high vibration"}`),
		Evidence: []Evidence{{
			ID: "00000000-0000-0000-0000-0000000000aa", Kind: "document",
			SourceID: "doc-1", Locator: "sop://p-302", Authority: "SITE_APPROVED",
			Content: "Isolate energy before intrusive work.",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Status     string   `json:"status"`
		EvidenceIDs []string `json:"evidence_ids"`
		RiskLevel  string   `json:"risk_level"`
		Confidence float64  `json:"confidence"`
	}
	if err := json.Unmarshal(response.Output, &output); err != nil {
		t.Fatalf("output is not the recommendation shape: %v", err)
	}
	if output.Status == "" || output.Confidence < 0 || output.Confidence > 1 {
		t.Fatalf("malformed recommendation output: %+v", output)
	}
	for _, id := range output.EvidenceIDs {
		if id != "00000000-0000-0000-0000-0000000000aa" {
			t.Fatalf("evidence id %q not in the request evidence set", id)
		}
	}
	if output.RiskLevel != "INFORMATIONAL" && output.RiskLevel != "ADVISORY" {
		t.Fatalf("risk_level %q outside the closed set accepted by the copilot decoder", output.RiskLevel)
	}
}
```

`http_embedding_provider_contract_test.go` (env: `EMBEDDING_ENDPOINT`,
`AI_API_KEY`, `EMBEDDING_MODEL`, `EMBEDDING_MODEL_VERSION`): embed two fixed
texts, assert `len(vectors) == 2`, equal dimensions, and
`provider.Model().Metric == "COSINE"`.

`anthropic_structured_provider_contract_test.go` (env: `ANTHROPIC_API_KEY`,
`ANTHROPIC_MODEL`, `ANTHROPIC_MODEL_VERSION`): same shape as the structured
contract test but with `NewAnthropicStructuredProvider`; metadata provider
must be `anthropic`.

`http_transcription_provider_contract_test.go` (env: `TRANSCRIPTION_ENDPOINT`,
`TRANSCRIPTION_API_KEY`, `TRANSCRIPTION_MODEL`, `TRANSCRIPTION_MODEL_VERSION`):
build a small silent WAV in memory (44-byte RIFF header + 0.25 s of silence at
8 kHz mono 16-bit) with a helper, call `Transcribe(ctx, reader, "audio/wav")`,
and assert either non-empty text or an explicit provider error (a deployment
endpoint may reject the synthetic audio; both outcomes are contract-valid —
assert that no panic/empty-mime crash occurs and the returned error is not
`ErrInvalidOutput` when text is empty).

- [ ] **Step 2: Run the tests without credentials (must skip)**

Run: `go test ./internal/skawld/ -run "Contract" -v`
Expected: 4 tests SKIPPED (self-skip), suite PASS.

- [ ] **Step 3: Add the CI job**

Add to `.github/workflows/ci.yml` after the `checks` job:

```yaml
  provider-contracts:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version: "1.25.12"
          cache: true
      - name: Download dependencies
        run: go mod download
      - name: Run AI provider contract tests
        run: go test ./internal/skawld/ -run Contract -v
```

The job runs on every push but the tests self-skip without secrets; when the
repository gains provider credentials, add them to the job's `env` (or use
repository secrets) and the contracts execute.

- [ ] **Step 4: Verify the workflow YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('OK')"`
Expected: `OK`.

- [ ] **Step 5: Commit**

```bash
git add internal/skawld/*_contract_test.go .github/workflows/ci.yml
git commit -m "test: add secret-gated live contract tests for AI providers"
```

### Task 11: Provider selection and transcription runbooks (docs)

**Files:**
- Create: `docs/runbooks/ai-providers.md`

- [ ] **Step 1: Write the runbook**

`docs/runbooks/ai-providers.md` must document, following the style of the
existing runbooks:
- The env contract: `STRUCTURED_PROVIDER=deterministic|openai|anthropic`,
  `EMBEDDING_PROVIDER=deterministic|openai`, and every credential env var
  (`AI_ENDPOINT`, `AI_API_KEY`, `AI_MODEL`, `AI_MODEL_VERSION`,
  `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_MODEL_VERSION`,
  `EMBEDDING_ENDPOINT`, `EMBEDDING_MODEL`, `EMBEDDING_MODEL_VERSION`).
- Endpoint formats for OpenAI-compatible (chat completions with
  `response_format: json_object`; embeddings) and Anthropic (Messages API).
- The embedding-dimension warning: switching `EMBEDDING_PROVIDER` after
  documents are embedded mixes vector dimensions in one store; pair any
  switch with re-ingestion (`delete from embeddings` + re-run ingestion).
- How to run the secret-gated contract tests locally
  (`go test ./internal/skawld/ -run Contract -v` with the env set).
- Transcription deployment: `TRANSCRIPTION_ENDPOINT` etc., the multipart
  contract, and that transcripts remain unverified candidates until a human
  verifies or rejects them.

- [ ] **Step 2: Commit**

```bash
git add docs/runbooks/ai-providers.md
git commit -m "docs: document AI provider and transcription deployment"
```

---

## Part C — Native packaging and signing

### Task 12: Flutter quality gates on native CI runners

**Files:**
- Modify: `.github/workflows/ci.yml` (`desktop-app-packages` job)

**Interfaces:**
- The `desktop-app-packages` job already runs on windows-latest and
  macos-latest with `flutter-version: "3.44.8"`.

- [ ] **Step 1: Add the quality-gate steps**

The `desktop-app-packages` job currently runs (in order): checkout,
flutter-action, "Generate platform host project", "Configure macOS
entitlements" (macOS only), `flutter pub get`, `dart run build_runner build`,
the platform build steps, the package steps, upload-artifact. Replace the
standalone `dart run build_runner build` step with a combined quality-gate
step placed in the same position (after `flutter pub get`, before the
platform builds):

```yaml
      - name: Format, analyze, and test on the native runner
        run: |
          dart format --output=none --set-exit-if-changed lib test
          dart run build_runner build
          flutter analyze
          flutter test
```

It runs on both runners (the job's `run` defaults to
`working-directory: mobile`). No other step is duplicated or removed.

- [ ] **Step 2: Verify the workflow YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('OK')"`
Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: run Flutter quality gates on native packaging runners"
```

### Task 13: Signing-gated release workflow and runbook

**Files:**
- Create: `.github/workflows/release.yml`
- Create: `docs/runbooks/desktop-signing.md`

**Interfaces:**
- Runs after `desktop-app-packages`; consumes its packaged artifacts; signing
  steps activate only when the required secrets exist.

- [ ] **Step 1: Write the workflow**

The signing job rebuilds and packages on the same runner (mirroring the
`desktop-app-packages` steps: checkout, flutter-action, platform host
generation, entitlements on macOS, `flutter pub get`, the combined quality
gate, platform build, package via `scripts/package-desktop-windows.ps1` /
`scripts/package-desktop-macos.sh`), then signs. `.github/workflows/release.yml`:

```yaml
name: Release signing

on:
  workflow_run:
    workflows: [CI]
    types: [completed]
    branches: [main]

permissions:
  contents: read

jobs:
  sign:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    strategy:
      fail-fast: false
      matrix:
        include:
          - platform: windows
            runner: windows-latest
          - platform: macos
            runner: macos-latest
    runs-on: ${{ matrix.runner }}
    timeout-minutes: 30
    defaults:
      run:
        working-directory: mobile
    steps:
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
      - uses: subosito/flutter-action@1a449444c387b1966244ae4d4f8c696479add0b2 # v2
        with:
          flutter-version: "3.44.8"
          cache: true
      - name: Generate platform host project
        run: flutter create --platforms=${{ matrix.platform }} --org com.skawld --project-name skawld_maintenance_mobile .
      - name: Configure macOS entitlements
        if: runner.os == 'macOS'
        run: ../scripts/configure-flutter-macos.sh
      - run: flutter pub get
      - name: Quality gates
        run: |
          dart format --output=none --set-exit-if-changed lib test
          dart run build_runner build
          flutter analyze
          flutter test
      - name: Build and package Windows
        if: runner.os == 'Windows'
        shell: pwsh
        run: |
          flutter build windows --release
          ../scripts/package-desktop-windows.ps1 -Version "signed-ci"
      - name: Build and package macOS
        if: runner.os == 'macOS'
        run: |
          flutter build macos --release
          ../scripts/package-desktop-macos.sh "signed-ci"
      - name: Import Windows signing certificate
        if: runner.os == 'Windows' && secrets.WINDOWS_CERT_BASE64 != ''
        shell: pwsh
        run: |
          $bytes = [Convert]::FromBase64String($env:WINDOWS_CERT_BASE64)
          [IO.File]::WriteAllBytes("$env:RUNNER_TEMP\cert.pfx", $bytes)
        env:
          WINDOWS_CERT_BASE64: ${{ secrets.WINDOWS_CERT_BASE64 }}
      - name: Sign Windows package
        if: runner.os == 'Windows' && secrets.WINDOWS_CERT_BASE64 != ''
        uses: skyleedev/signtool-sign # pin a reviewed SHA at implementation time (repo convention)
        with:
          file: dist/desktop/**/*.exe
          certificate: ${{ runner.temp }}/cert.pfx
          password: ${{ secrets.WINDOWS_CERT_PASSWORD }}
      - name: Sign and notarize macOS package
        if: runner.os == 'macOS' && secrets.MACOS_SIGNING_IDENTITY != ''
        env:
          MACOS_SIGNING_IDENTITY: ${{ secrets.MACOS_SIGNING_IDENTITY }}
          MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
          MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
          MACOS_NOTARY_PRIVATE_KEY: ${{ secrets.MACOS_NOTARY_PRIVATE_KEY }}
        run: |
          codesign --deep --force --options runtime \
            --sign "$MACOS_SIGNING_IDENTITY" dist/desktop/*.app
          ditto -c -k --keepParent dist/desktop/*.app dist/desktop/skawld-macos.zip
          xcrun notarytool submit dist/desktop/skawld-macos.zip \
            --key-id "$MACOS_NOTARY_KEY_ID" \
            --issuer "$MACOS_NOTARY_ISSUER_ID" \
            --key "$MACOS_NOTARY_PRIVATE_KEY" --wait
      - uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4
        if: always()
        with:
          name: signed-${{ matrix.platform }}-${{ github.run_number }}
          path: mobile/dist/desktop/
          if-no-files-found: ignore
```

All signing steps are guarded by `secrets.XXX != ''`, so the workflow is
green before credentials exist and performs signing once they are added.
Pin the `skyleedev/signtool-sign` action to a reviewed commit SHA following
the repo convention (`actions/...@<full-sha> # <major>`) at implementation
time.

- [ ] **Step 2: Write the signing runbook**

`docs/runbooks/desktop-signing.md`: how to provision the Windows code-signing
certificate (export PFX + password, store as `WINDOWS_CERT_BASE64` /
`WINDOWS_CERT_PASSWORD`), the macOS Developer ID Application identity and
notary key/issuer/private key secrets, and the clean-machine install smoke
test procedure (fresh VM: install the signed package, launch, sign in via the
demo accounts, load demo data, verify no SmartScreen/Gatekeeper warnings).

- [ ] **Step 3: Verify the workflow YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); print('OK')"`
Expected: `OK`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yml docs/runbooks/desktop-signing.md
git commit -m "ci: add secrets-gated release signing workflow"
```

---

## Part D — Pilot-customer package

### Task 14: Pilot-readiness decision and drill runbook (docs)

**Files:**
- Create: `docs/runbooks/pilot-readiness.md`

- [ ] **Step 1: Write the runbook**

`docs/runbooks/pilot-readiness.md` documents the blocked checklist (no code
changes):
- Agreed topology (single-server/private deployment) and the compose pilot
  profile (`deployments/pilot/compose.yaml`).
- Data classification and retention targets.
- IdP selection and the OIDC/SSO deployment guide (reference
  `docs/runbooks/` for the existing OIDC runbook).
- S3/on-prem object-storage selection.
- Support contacts and escalation.
- RPO/RTO targets, then the restore drill procedure using
  `scripts/restore-postgres.sh` + `scripts/verify-restore.sh`, measured
  against the agreed targets, and an object-storage backup/restore drill for
  the selected implementation.
- Safety review and customer acceptance sign-off.
- A status table with `[ ]` checkboxes so the pilot owner can track
  completion; each item names the evidence required (a completed drill log,
  a signed acceptance, a config value).

- [ ] **Step 2: Commit**

```bash
git add docs/runbooks/pilot-readiness.md
git commit -m "docs: add pilot-readiness decision and drill checklist"
```

---

## Verification (run before finishing)

- `gofmt -l .` empty; `go vet ./...` clean; `go build ./...` clean.
- `go test ./...` green; `TEST_DATABASE_URL=... go test -race -count=1 ./...`
  green (includes the new cursor-pagination integration tests and CLI
  negative tests).
- `go run ./cmd/eval -dataset test/evaldata/pilot-v1.json` gates pass.
- `python3 -c "import yaml; yaml.safe_load(open('api/openapi.yaml'))"` and
  both workflow YAMLs parse.
- `cd web && npm run lint && npm test && npm run build` green (load-more
  wiring covered by the Vitest suite).
- `go test ./internal/skawld/ -run Contract -v` shows 4 skipped (no
  credentials) or executed (with credentials).
- No dependency changes in `go.mod` or `web/package.json`.
- `git status` shows only the intended files.
