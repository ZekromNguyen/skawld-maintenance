# Design: Missing List APIs — GET /reports and GET /handovers

Date: 2026-08-06
Status: Approved
Branch: feature/maintenance-copilot-pilot-baseline

## Problem

The pilot console calls two list endpoints that do not exist in the backend:

- `GET /api/v1/reports` — `web/src/console/pages/ReportsPage.tsx:21`
- `GET /api/v1/handovers` — `web/src/console/pages/HandoverPage.tsx:20` and
  `web/src/console/pages/DashboardPage.tsx:19`

Both return 404 today. They are missing at every layer: HTTP handler,
application service method, postgres repository query, and OpenAPI spec path.
The detail/get, transition, and prepare endpoints exist and work; only the
list reads are absent.

## Goals

- Serve `GET /reports` and `GET /handovers` so the three existing pages work.
- Filters: `site_id`, `state` (repeated param allowed), plus keyset cursor
  pagination (`cursor`, `page_size`).
- Consistent with existing conventions: org + site-scoped authorization,
  `{items, next_cursor, has_more}` envelope, `Problem` error body.

## Non-Goals (follow-up items, out of scope this round)

- Standardizing the existing `{items}`-only list endpoints (executions,
  documents, incidents, assets, demonstrations, workflows) to the cursor
  envelope.
- A dedicated "latest handover per site" endpoint.
- Report summary/aggregate list variants.

## Architecture

### Application layer

Add one method to each service:

- `internal/report/application/service.go` — `List(ctx, principal, Filter, Cursor) ([]Report, string, bool, error)` returning `(items, nextCursor, hasMore, err)`.
- `internal/handover/application/service.go` — same shape for `Handover`.

Both services depend on a `Store` interface; extend each interface with the
corresponding `List` method:

```go
type ReportFilter struct {
    SiteID string   // "" = all authorized sites
    States []string // empty = all states
    Cursor string   // opaque, from previous response
    PageSize int
}
```

Service responsibilities: normalize/validate the filter (delegate cursor
decode to a small shared cursor package), pass through to the store, return
the envelope. The service returns an empty `next_cursor` when the page is
exhausted; the handler serializes empty as JSON `null`.
Empty result returns `{items: [], next_cursor: null, has_more: false}`.

### Cursor design (shared)

New small package `internal/platform/keyset` (or place in the existing
`internal/platform/` tree):

- Encode: base64url(JSON `{"t":"<RFC3339>","i":"<uuid>"}`).
- Decode returns `(time, id, error)`; bad input → `ErrInvalidCursor`.
- Handovers key on `(shift_start, id)`; reports key on `(created_at, id)`.
- WHERE predicate for the "next page" query uses the decoded values, e.g.
  for handovers `(shift_start, id) < ($t, $i)` with `ORDER BY shift_start DESC, id DESC`.

### Adapter layer (postgres)

- `internal/report/adapter/postgres/store.go` — `List` querying
  `maintenance_reports`, ordered `created_at DESC, id DESC`, joined scope
  predicate copied from `Get`:
  `organization_id = $org AND (COALESCE(cardinality($site_ids), 0) = 0 OR site_id = ANY($site_ids))`.
- `internal/handover/adapter/postgres/store.go` — `List` querying
  `shift_handovers`, ordered `shift_start DESC, id DESC`, same scope
  predicate (already present in `get()`).
- Both fetch `page_size + 1` rows to compute `has_more`; the extra row is
  discarded from `items`.

### Migration 00012

```sql
CREATE INDEX maintenance_reports_list_idx
    ON maintenance_reports(organization_id, created_at DESC, id DESC);
```

Handovers already have a matching index:
`shift_handovers_scope_idx(organization_id, site_id, shift_start DESC)`.

### HTTP layer

- `internal/platform/httpserver/reports.go` — add `router.Get("/reports", listReports(service))` inside the existing `mountReportRoutes`.
- `internal/platform/httpserver/handovers.go` — add `router.Get("/handovers", listHandovers(service))` inside the existing `mountHandoverRoutes`.
- Query params: `site_id` (UUID), `state` (repeated), `cursor`, `page_size`.
- Validation (400 `Problem` on failure): `site_id` must be a valid UUID;
  `state` must be one of the enum values for the resource
  (reports: DRAFT/SUBMITTED/APPROVED/REJECTED; handovers:
  DRAFT/SUBMITTED/ACCEPTED/ACKNOWLEDGED); `page_size` default 25, max 100;
  `cursor` must decode.
- Response: `200 {"items": [...], "next_cursor": "..."|null, "has_more": bool}`.
- Auth: existing principal middleware; store enforces org/site scope.

## Frontend

- `web/src/api.ts` — `reports(options?)` and `handovers(options?)` accept
  `{site_id, state?, cursor, page_size?}` and return the envelope.
- `web/src/types.ts` — extend `ListResponse<T>` with optional
  `next_cursor: string | null` and `has_more: boolean`.
- `HandoverPage` — pass `site_id` from `useSite()`; latest = first item;
  history list gets a "load more" control walking `next_cursor`.
- `ReportsPage` — the existing state filter becomes a server-side `state`
  param; add next-page control.
- `DashboardPage` — pending handover count: walk pages with
  `state=DRAFT&state=SUBMITTED&state=ACCEPTED` until `has_more` is false
  (small `fetchAll` helper in `api.ts`), then count non-ACKNOWLEDGED items.

## OpenAPI

- Add `GET /api/v1/reports` and `GET /api/v1/handovers` paths with
  `site_id`, `state`, `cursor`, `page_size` parameters and a `200` response
  containing `items`, `next_cursor`, `has_more`.
- Update the existing inline list response schemas used by these two
  endpoints (do not touch the other endpoints' schemas this round).

## Error handling

| Case | Status |
|---|---|
| Bad `site_id` / `state` / `page_size` / `cursor` | 400 `Problem` |
| Unauthenticated | 401 (existing middleware) |
| No access to org/site | 403 (existing middleware / store scope) |
| Empty result | 200 `{items: [], next_cursor: null, has_more: false}` |

## Testing

- Go unit tests: cursor codec round-trip and invalid-input cases; service
  `List` filter passthrough and envelope computation.
- Store integration tests (following existing `store_integration_test.go`
  patterns in both adapters): keyset correctness across multiple pages,
  tie-breaking on `id`, site-scope enforcement, `has_more` boundary at
  exactly `page_size` rows.
- Handler tests: param validation (400 cases), empty result, envelope shape.
- Contract tests: extend `test/contract/sdk/sdk_contract_test.go` with both
  list endpoints.
- Frontend: extend `web/src/api.test.ts`; page-level tests for filter
  param wiring and pagination control.
- Commands: `go test ./...`, `cd web && npm test`, `go vet ./...`.

## Data flow

1. Page calls `api.reports({site_id, state, page_size})`.
2. Handler validates params, calls `service.List`.
3. Service decodes cursor (if any), calls `store.List`.
4. Store runs keyset query with org/site scope, returns `page_size + 1` rows.
5. Service computes `next_cursor`/`has_more`, returns envelope.
6. Handler writes `200` JSON; frontend renders and offers "load more" when
   `has_more` is true.
