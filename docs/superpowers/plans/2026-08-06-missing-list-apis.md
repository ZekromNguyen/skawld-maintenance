# Missing List APIs (GET /reports, GET /handovers) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/reports` and `GET /api/v1/handovers` with site/state filters and keyset cursor pagination so the pilot console's Reports, Handovers, and Dashboard pages work (they currently 404).

**Architecture:** Each endpoint follows the existing layered pattern: HTTP handler in `internal/platform/httpserver` → application service method in `internal/report/application` / `internal/handover/application` → postgres store in `.../adapter/postgres`. A new leaf package `internal/platform/keyset` provides the cursor codec. Keyset pagination rides the existing `shift_handovers_scope_idx` index for handovers and a new `maintenance_reports_list_idx` index for reports. Frontend pages consume a `{items, next_cursor, has_more}` envelope.

**Tech Stack:** Go 1.x (chi, pgx, google/uuid), PostgreSQL (goose migrations), TypeScript/React (vitest, react-testing-library), YAML OpenAPI 3.1.

## Global Constraints

- Module path is `github.com/ZekromNguyen/skawld-maintenance`; all internal imports use it.
- Envelope for both endpoints: `{"items": [...], "next_cursor": <string|null>, "has_more": <bool>}`.
- `page_size` default 25, maximum 100. Invalid query params → 400 `Problem` body.
- Cursor: base64url (RawURLEncoding) of JSON `{"t":"<RFC3339Nano UTC>","i":"<uuid>"}`.
- Report ordering: `created_at DESC, id DESC`. Handover ordering: `shift_start DESC, id DESC`.
- Authorization: reports list requires `report:write` OR `report:approve`; handovers list requires `handover:write` OR `handover:accept`. SQL always scopes `organization_id = principal.OrganizationID` plus `site_id = ANY(principal.SiteIDs)` when the principal's site list is non-empty.
- Report states: `DRAFT`, `SUBMITTED`, `APPROVED`, `REJECTED`. Handover states: `DRAFT`, `SUBMITTED`, `ACCEPTED`, `ACKNOWLEDGED`. Any other `state` value → 400.
- No new Go dependencies. Web: no new npm dependencies.
- Migrations use goose markers `-- +goose Up` / `-- +goose Down`; next number is `00012`. Run `make migrate-up` after creating it.
- TDD: write the failing test first, verify it fails, then implement, then verify it passes, then commit.

---

### Task 1: Keyset cursor package

**Files:**
- Create: `internal/platform/keyset/keyset.go`
- Test: `internal/platform/keyset/keyset_test.go`

**Interfaces:**
- Produces:
  - `type Key struct { Timestamp time.Time; ID string }`
  - `func Encode(timestamp time.Time, id string) string`
  - `func Decode(cursor string) (Key, error)` — returns `ErrInvalidCursor` (wrapped with reason) on any malformed input.

- [ ] **Step 1: Write the failing test**

Create `internal/platform/keyset/keyset_test.go`:

```go
package keyset

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 123456789, time.UTC)
	id := "3b0f3c1e-9f4a-4f0e-8b2a-123456789abc"
	cursor := Encode(now, id)
	if cursor == "" {
		t.Fatal("cursor must not be empty")
	}
	key, err := Decode(cursor)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !key.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %v, want %v", key.Timestamp, now)
	}
	if key.ID != id {
		t.Fatalf("id = %q, want %q", key.ID, id)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	cases := []string{"", "not-base64!!", "e30=" /* empty json {} */, "aGVsbG8=" /* "hello" */}
	for _, cursor := range cases {
		if _, err := Decode(cursor); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("Decode(%q) error = %v, want ErrInvalidCursor", cursor, err)
		}
	}
}

func TestEncodeProducesURLSafeBase64(t *testing.T) {
	cursor := Encode(time.Now().UTC(), "3b0f3c1e-9f4a-4f0e-8b2a-123456789abc")
	if strings.ContainsAny(cursor, "+/=") {
		t.Fatalf("cursor %q contains base64 URL-unsafe characters", cursor)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/keyset/ -v`
Expected: FAIL — package does not exist (compile error: no Go files).

- [ ] **Step 3: Write minimal implementation**

Create `internal/platform/keyset/keyset.go`:

```go
package keyset

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid keyset cursor")

type Key struct {
	Timestamp time.Time
	ID        string
}

type cursorPayload struct {
	T string `json:"t"`
	I string `json:"i"`
}

func Encode(timestamp time.Time, id string) string {
	payload, _ := json.Marshal(cursorPayload{
		T: timestamp.UTC().Format(time.RFC3339Nano),
		I: id,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func Decode(cursor string) (Key, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return Key{}, fmt.Errorf("%w: malformed encoding", ErrInvalidCursor)
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Key{}, fmt.Errorf("%w: malformed payload", ErrInvalidCursor)
	}
	if payload.T == "" || payload.I == "" {
		return Key{}, fmt.Errorf("%w: missing fields", ErrInvalidCursor)
	}
	parsed, err := time.Parse(time.RFC3339Nano, payload.T)
	if err != nil {
		return Key{}, fmt.Errorf("%w: bad timestamp", ErrInvalidCursor)
	}
	if _, err := uuid.Parse(payload.I); err != nil {
		return Key{}, fmt.Errorf("%w: bad id", ErrInvalidCursor)
	}
	return Key{Timestamp: parsed, ID: payload.I}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/keyset/ -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/platform/keyset/
git commit -m "feat: add keyset cursor codec for paginated list endpoints"
```

---

### Task 2: Report list — application layer

**Files:**
- Modify: `internal/report/application/service.go` (imports, filter/result types, `Store` interface, `List` method)
- Modify: `internal/report/application/service_test.go` (add `List` to the `editStore` fake; add new tests)

**Interfaces:**
- Consumes: `keyset.Encode`, `keyset.Decode`, `keyset.ErrInvalidCursor` from Task 1.
- Produces:
  - `type ReportFilter struct { SiteID string; States []string; Cursor string; PageSize int }`
  - `type ListResult struct { Items []Report; NextCursor string; HasMore bool }`
  - Store method: `List(ctx context.Context, principal identitydomain.Principal, filter ReportFilter) ([]Report, bool, error)` — returns `(items, hasMore, err)`; items already truncated to `PageSize`; `hasMore` true when a page_size+1th row existed.
  - Service method: `func (s Service) List(ctx context.Context, principal identitydomain.Principal, filter ReportFilter) (ListResult, error)`

- [ ] **Step 1: Write the failing tests**

Add to `internal/report/application/service_test.go` (keep the existing `editStore`; add `List` to it and add these tests):

```go
func (*editStore) List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error) {
	return nil, false, nil
}

type listReportStore struct{}

func (*listReportStore) LoadExecutionContext(context.Context, identitydomain.Principal, string) (ExecutionContext, error) {
	return ExecutionContext{}, nil
}
func (*listReportStore) SaveDraft(context.Context, identitydomain.Principal, string, Report, json.RawMessage) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Get(context.Context, identitydomain.Principal, string) (Report, error) {
	return Report{}, nil
}
func (*listReportStore) Edit(context.Context, identitydomain.Principal, string, string, Edit) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Submit(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Approve(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) RecordCall(context.Context, identitydomain.Principal, ExecutionContext, skawld.Generation, string, string) error {
	return nil
}
func (*listReportStore) List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error) {
	return []Report{{
		ID: "00000000-0000-0000-0000-00000000000a", CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func TestListReportsRequiresPermission(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	_, err := service.List(context.Background(), identitydomain.Principal{ID: "p1"}, ReportFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListReportsRejectsSiteOutsideScope(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	_, err := service.List(context.Background(), principal, ReportFilter{SiteID: "s2"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListReportsRejectsUnknownState(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	_, err := service.List(context.Background(), principal, ReportFilter{States: []string{"NOPE"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestListReportsComputesCursor(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	result, err := service.List(context.Background(), principal, ReportFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if result.NextCursor == "" {
		t.Fatal("NextCursor must be set when HasMore")
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(result.Items))
	}
}
```

Update imports in `service_test.go`: add `"errors"` and `"time"` (check current imports; `context`, `encoding/json`, `testing` and the domain packages are already there).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/report/application/ -run TestList -v`
Expected: FAIL — `ReportFilter` / `List` undefined, and `editStore` no longer satisfies `Store`.

- [ ] **Step 3: Implement the service changes**

In `internal/report/application/service.go`:

Add imports `"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"` (the file already imports `time`).

Add after the `Transition` type:

```go
type ReportFilter struct {
	SiteID   string
	States   []string
	Cursor   string
	PageSize int
}

type ListResult struct {
	Items      []Report
	NextCursor string
	HasMore    bool
}
```

Add `List` to the `Store` interface:

```go
type Store interface {
	LoadExecutionContext(context.Context, identitydomain.Principal, string) (ExecutionContext, error)
	SaveDraft(context.Context, identitydomain.Principal, string, Report, json.RawMessage) (Report, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Report, error)
	List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error)
	Edit(context.Context, identitydomain.Principal, string, string, Edit) (Report, bool, error)
	Submit(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error)
	Approve(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error)
	RecordCall(context.Context, identitydomain.Principal, ExecutionContext, skawld.Generation, string, string) error
}
```

Add the `List` method right after `Get`:

```go
func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter ReportFilter,
) (ListResult, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) &&
		!principal.Has(identitydomain.PermissionReportApprove) {
		return ListResult{}, ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return ListResult{}, ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	for _, state := range filter.States {
		switch state {
		case "DRAFT", "SUBMITTED", "APPROVED", "REJECTED":
		default:
			return ListResult{}, ErrInvalid
		}
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, principal, filter)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Items: items, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		result.NextCursor = keyset.Encode(last.CreatedAt, last.ID)
	}
	return result, nil
}
```

> Note: the decoded cursor is passed to the store as `"<RFC3339Nano>|<uuid>"` in `filter.Cursor` to avoid adding a keyset type to the store signature. The store splits on the last `|`. This keeps the store free of codec imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/report/application/`
Expected: PASS (existing tests + 4 new).

- [ ] **Step 5: Commit**

```bash
git add internal/report/application/
git commit -m "feat: add report list query with filters and keyset cursor"
```

---

### Task 3: Report list — postgres store

**Files:**
- Modify: `internal/report/adapter/postgres/store.go` (add `fmt` import; add `List` method)
- Modify: `internal/report/adapter/postgres/store_integration_test.go` (add `TestReportListPagination`)

**Interfaces:**
- Consumes: `reportapp.ReportFilter` (with `Cursor` carrying `"<RFC3339Nano>|<uuid>"` when set), `reportapp.Report` from Task 2.
- Produces: `func (s Store) List(ctx context.Context, principal identitydomain.Principal, filter reportapp.ReportFilter) ([]reportapp.Report, bool, error)`

- [ ] **Step 1: Write the failing integration test**

Add to `internal/report/adapter/postgres/store_integration_test.go` (reuse the seed pattern from `TestReportDraftEditAndApprovalPersistence` — org, site, principal, asset, execution — then insert reports directly):

```go
func TestReportListPagination(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID := uuid.NewString(), uuid.NewString()
	principalID, assetID, executionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, statement := range []string{
		`INSERT INTO organizations (id, name, source_of_truth, version, created_at, updated_at)
		 VALUES ($1::uuid, 'Report List', 'OWNED_BY_SKAWLD', 1, $2, $2)`,
		`INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, 'RL', 'Report List Site', 'UTC', 'ACTIVE', 1, $3, $3)`,
		`INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
		 VALUES ($1::uuid, $1, 'List Technician', 'ACTIVE', $2, $2)`,
		`INSERT INTO assets (id, organization_id, site_id, tag, name, asset_class, status, source_of_truth, attributes, version, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, 'P-LIST', 'List Pump', 'CENTRIFUGAL_PUMP', 'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4)`,
		`INSERT INTO maintenance_executions (id, organization_id, site_id, asset_id, purpose, state, assigned_to, version, created_by, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'List evidence report', 'ASSIGNED', $5::uuid, 1, $5::uuid, $6, $6)`,
	} {
		if _, err := pool.Exec(ctx, statement, organizationID, siteID, principalID, assetID, executionID, now); err != nil {
			t.Fatal(err)
		}
	}
	insertReport := func(id string, createdAt time.Time, state string) {
		statement := `INSERT INTO maintenance_reports
			(id, organization_id, execution_id, site_id, revision, version, state,
			 structured_content, evidence_snapshot, generated_by_kind, created_by,
			 created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 1, 1, $5,
			 '{}'::jsonb, '[]'::jsonb, 'HUMAN', $6::uuid, $7, $7)`
		if _, err := pool.Exec(ctx, statement, id, organizationID, executionID, siteID, state, principalID, createdAt); err != nil {
			t.Fatal(err)
		}
	}
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		id := uuid.NewString()
		ids = append(ids, id)
		insertReport(id, now.Add(-time.Duration(i)*time.Hour), "DRAFT")
	}

	store := reportpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Audit: audit.Sink{}, Idempotency: idempotency.Store{},
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionReportWrite: {},
		},
	}

	pageOne, hasMore, err := store.List(ctx, principal, reportapp.ReportFilter{PageSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !hasMore {
		t.Fatal("hasMore = false, want true for 5 rows at page_size 3")
	}
	if len(pageOne) != 3 {
		t.Fatalf("len(pageOne) = %d, want 3", len(pageOne))
	}
	if pageOne[0].ID != ids[0] {
		t.Fatalf("newest first: got %s, want %s", pageOne[0].ID, ids[0])
	}

	cursor := reportapp.ReportFilter{
		PageSize: 3,
		Cursor:   pageOne[2].CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + pageOne[2].ID,
	}
	pageTwo, hasMoreTwo, err := store.List(ctx, principal, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if hasMoreTwo {
		t.Fatal("hasMore = true, want false after exhausting 5 rows")
	}
	if len(pageTwo) != 2 {
		t.Fatalf("len(pageTwo) = %d, want 2", len(pageTwo))
	}

	scoped, _, err := store.List(ctx, principal, reportapp.ReportFilter{PageSize: 25, SiteID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 0 {
		t.Fatalf("len(scoped) = %d, want 0 for foreign site", len(scoped))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://... go test ./internal/report/adapter/postgres/ -run TestReportList -v`
Expected: FAIL — `store.List` undefined.

- [ ] **Step 3: Implement `Store.List`**

In `internal/report/adapter/postgres/store.go`:

Add `"fmt"` to imports. Add this method after `Get`:

```go
func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter reportapp.ReportFilter,
) ([]reportapp.Report, bool, error) {
	query := `
		SELECT id::text, organization_id::text, site_id::text, execution_id::text,
		       revision, version, state, structured_content, evidence_snapshot,
		       coalesce(provider, ''), coalesce(model, ''),
		       coalesce(model_version, ''), coalesce(prompt_version, ''),
		       coalesce(input_sha256, ''), coalesce(output_sha256, ''),
		       generated_by_kind, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(approved_by::text, ''), approved_at, created_at, updated_at
		FROM maintenance_reports
		WHERE organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR site_id = ANY($2::uuid[]))`
	args := []any{principal.OrganizationID, principal.SiteIDs}
	if filter.SiteID != "" {
		args = append(args, filter.SiteID)
		query += fmt.Sprintf(" AND site_id = $%d::uuid", len(args))
	}
	if len(filter.States) > 0 {
		args = append(args, filter.States)
		query += fmt.Sprintf(" AND state = ANY($%d::text[])", len(args))
	}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, reportapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]reportapp.Report, 0, filter.PageSize+1)
	for rows.Next() {
		var value reportapp.Report
		var content, evidence []byte
		if err := rows.Scan(
			&value.ID, &value.OrganizationID, &value.SiteID, &value.ExecutionID,
			&value.Revision, &value.Version, &value.State, &content, &evidence,
			&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
			&value.InputSHA256, &value.OutputSHA256, &value.GeneratedBy,
			&value.SubmittedBy, &value.SubmittedAt, &value.ApprovedBy,
			&value.ApprovedAt, &value.CreatedAt, &value.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(content, &value.Content); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
			return nil, false, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}
```

> `strings` is already imported in this file (used by existing code).

- [ ] **Step 4: Run test to verify it passes**

Run: `TEST_DATABASE_URL=postgres://... go test ./internal/report/adapter/postgres/ -run TestReportList -v`
Expected: PASS.
Also run `go test ./internal/report/...` to confirm the whole package compiles.

- [ ] **Step 5: Commit**

```bash
git add internal/report/adapter/postgres/
git commit -m "feat: paginate report list in postgres store"
```

---

### Task 4: Handover list — application layer

**Files:**
- Modify: `internal/handover/application/service.go`
- Modify: `internal/handover/application/service_test.go`

**Interfaces:**
- Consumes: `keyset.Encode`, `keyset.Decode`, `keyset.ErrInvalidCursor` from Task 1.
- Produces:
  - `type HandoverFilter struct { SiteID string; States []string; Cursor string; PageSize int }`
  - `type ListResult struct { Items []Handover; NextCursor string; HasMore bool }`
  - Store method: `List(ctx context.Context, principal identitydomain.Principal, filter HandoverFilter) ([]Handover, bool, error)`
  - Service method: `func (s Service) List(ctx context.Context, principal identitydomain.Principal, filter HandoverFilter) (ListResult, error)`

- [ ] **Step 1: Write the failing tests**

`internal/handover/application/service_test.go` currently contains only `TestDecodeContentRejectsUnknownEvidence` and imports `encoding/json`, `errors`, `testing`, `skawld`. Extend its imports with `context`, `time`, `identitydomain`, and `knowledgedomain`, and add a complete fake store plus the tests. The `Handover` struct requires `ID` and `ShiftStart` fields only for the List test.

```go
type fakeStore struct{}

func (*fakeStore) LoadWindowContext(context.Context, identitydomain.Principal, PrepareDraft) (WindowContext, error) {
	return WindowContext{}, nil
}
func (*fakeStore) SaveDraft(context.Context, identitydomain.Principal, string, Handover) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Get(context.Context, identitydomain.Principal, string) (Handover, error) {
	return Handover{}, nil
}
func (*fakeStore) Edit(context.Context, identitydomain.Principal, string, string, Edit) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Submit(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Accept(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Acknowledge(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) RecordCall(context.Context, identitydomain.Principal, WindowContext, skawld.Generation, string, string) error {
	return nil
}
func (*fakeStore) List(context.Context, identitydomain.Principal, HandoverFilter) ([]Handover, bool, error) {
	return []Handover{{
		ID: "00000000-0000-0000-0000-00000000000a", ShiftStart: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func TestListHandoversRequiresPermission(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	_, err := service.List(context.Background(), identitydomain.Principal{ID: "p1"}, HandoverFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListHandoversRejectsSiteOutsideScope(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	_, err := service.List(context.Background(), principal, HandoverFilter{SiteID: "s2"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListHandoversRejectsUnknownState(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	_, err := service.List(context.Background(), principal, HandoverFilter{States: []string{"NOPE"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestListHandoversComputesCursor(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	result, err := service.List(context.Background(), principal, HandoverFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if result.NextCursor == "" {
		t.Fatal("NextCursor must be set when HasMore")
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(result.Items))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handover/application/ -run TestList -v`
Expected: FAIL — types/method undefined and fake no longer satisfies `Store`.

- [ ] **Step 3: Implement the service changes**

In `internal/handover/application/service.go`:

Add import `"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"`.

Add after the `Transition` type:

```go
type HandoverFilter struct {
	SiteID   string
	States   []string
	Cursor   string
	PageSize int
}

type ListResult struct {
	Items      []Handover
	NextCursor string
	HasMore    bool
}
```

Add `List` to the `Store` interface and add the method after `Get`:

```go
func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter HandoverFilter,
) (ListResult, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) &&
		!principal.Has(identitydomain.PermissionHandoverAccept) {
		return ListResult{}, ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return ListResult{}, ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	for _, state := range filter.States {
		switch state {
		case "DRAFT", "SUBMITTED", "ACCEPTED", "ACKNOWLEDGED":
		default:
			return ListResult{}, ErrInvalid
		}
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, principal, filter)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Items: items, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		result.NextCursor = keyset.Encode(last.ShiftStart, last.ID)
	}
	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handover/application/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handover/application/
git commit -m "feat: add handover list query with filters and keyset cursor"
```

---

### Task 5: Handover list — postgres store

**Files:**
- Modify: `internal/handover/adapter/postgres/store.go` (add `fmt` import; add `List` method)
- Modify: `internal/handover/adapter/postgres/store_integration_test.go` (add `TestHandoverListPagination`)

**Interfaces:**
- Consumes: `handoverapp.HandoverFilter`, `handoverapp.Handover` from Task 4.
- Produces: `func (s Store) List(ctx context.Context, principal identitydomain.Principal, filter handoverapp.HandoverFilter) ([]handoverapp.Handover, bool, error)`

- [ ] **Step 1: Write the failing integration test**

Add to `internal/handover/adapter/postgres/store_integration_test.go`, mirroring the seed pattern from the existing test (`TestPrepareHandoverPersistsExactProvenance`): seed org, site, principal, asset, execution, then insert shift_handovers directly:

```go
func TestHandoverListPagination(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID := uuid.NewString(), uuid.NewString()
	principalID, assetID, executionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, statement := range []string{
		`INSERT INTO organizations (id, name, source_of_truth, version, created_at, updated_at)
		 VALUES ($1::uuid, 'Handover List', 'OWNED_BY_SKAWLD', 1, $2, $2)`,
		`INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, 'HL', 'Handover List Site', 'UTC', 'ACTIVE', 1, $3, $3)`,
		`INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
		 VALUES ($1::uuid, $1, 'List Supervisor', 'ACTIVE', $2, $2)`,
		`INSERT INTO assets (id, organization_id, site_id, tag, name, asset_class, status, source_of_truth, attributes, version, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, 'P-HLIST', 'List Pump', 'CENTRIFUGAL_PUMP', 'ACTIVE', 'OWNED_BY_SKAWLD', '{}'::jsonb, 1, $4, $4)`,
		`INSERT INTO maintenance_executions (id, organization_id, site_id, asset_id, purpose, state, assigned_to, version, created_by, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'List handover evidence', 'ASSIGNED', $5::uuid, 1, $5::uuid, $6, $6)`,
	} {
		if _, err := pool.Exec(ctx, statement, organizationID, siteID, principalID, assetID, executionID, now); err != nil {
			t.Fatal(err)
		}
	}
	insertHandover := func(id string, shiftStart time.Time) {
		statement := `INSERT INTO shift_handovers
			(id, organization_id, site_id, shift_start, shift_end, state,
			 structured_content, evidence_snapshot, provider, model, model_version,
			 prompt_version, input_sha256, output_sha256, version, created_by, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, 'DRAFT',
			 '{}'::jsonb, '[]'::jsonb, 'skawld-copilot', 'copilot-v1', '1', 'p1',
			 'a', 'b', 1, $6::uuid, $4, $4)`
		if _, err := pool.Exec(ctx, statement, id, organizationID, siteID, shiftStart, shiftStart.Add(12*time.Hour), principalID); err != nil {
			t.Fatal(err)
		}
	}
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		id := uuid.NewString()
		ids = append(ids, id)
		insertHandover(id, now.Add(-time.Duration(i)*time.Hour))
	}

	store := handoverpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Audit: audit.Sink{}, Idempotency: idempotency.Store{},
	}
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionHandoverWrite: {},
		},
	}

	pageOne, hasMore, err := store.List(ctx, principal, handoverapp.HandoverFilter{PageSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !hasMore {
		t.Fatal("hasMore = false, want true for 5 rows at page_size 3")
	}
	if len(pageOne) != 3 {
		t.Fatalf("len(pageOne) = %d, want 3", len(pageOne))
	}
	if pageOne[0].ID != ids[0] {
		t.Fatalf("newest shift first: got %s, want %s", pageOne[0].ID, ids[0])
	}

	pageTwo, hasMoreTwo, err := store.List(ctx, principal, handoverapp.HandoverFilter{
		PageSize: 3,
		Cursor:   pageOne[2].ShiftStart.UTC().Format(time.RFC3339Nano) + "|" + pageOne[2].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasMoreTwo {
		t.Fatal("hasMore = true, want false after exhausting 5 rows")
	}
	if len(pageTwo) != 2 {
		t.Fatalf("len(pageTwo) = %d, want 2", len(pageTwo))
	}

	scoped, _, err := store.List(ctx, principal, handoverapp.HandoverFilter{PageSize: 25, SiteID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 0 {
		t.Fatalf("len(scoped) = %d, want 0 for foreign site", len(scoped))
	}
}
```

Check the existing test's import aliases (e.g., `handoverpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/handover/adapter/postgres"`, `handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"`) and match them.

- [ ] **Step 2: Run test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://... go test ./internal/handover/adapter/postgres/ -run TestHandoverList -v`
Expected: FAIL — `store.List` undefined.

- [ ] **Step 3: Implement `Store.List`**

In `internal/handover/adapter/postgres/store.go`:

Add `"fmt"` to imports. Add this method after `get`:

```go
func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter handoverapp.HandoverFilter,
) ([]handoverapp.Handover, bool, error) {
	query := `
		SELECT id::text, organization_id::text, site_id::text,
		       shift_start, shift_end, state, structured_content,
		       evidence_snapshot, provider, model, model_version,
		       prompt_version, input_sha256, output_sha256, version,
		       created_by::text, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(accepted_by::text, ''), accepted_at,
		       coalesce(acknowledged_by::text, ''), acknowledged_at,
		       created_at, updated_at
		FROM shift_handovers
		WHERE organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR site_id = ANY($2::uuid[]))`
	args := []any{principal.OrganizationID, principal.SiteIDs}
	if filter.SiteID != "" {
		args = append(args, filter.SiteID)
		query += fmt.Sprintf(" AND site_id = $%d::uuid", len(args))
	}
	if len(filter.States) > 0 {
		args = append(args, filter.States)
		query += fmt.Sprintf(" AND state = ANY($%d::text[])", len(args))
	}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, handoverapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (shift_start, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY shift_start DESC, id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]handoverapp.Handover, 0, filter.PageSize+1)
	for rows.Next() {
		var value handoverapp.Handover
		var content, evidence []byte
		if err := rows.Scan(
			&value.ID, &value.OrganizationID, &value.SiteID,
			&value.ShiftStart, &value.ShiftEnd, &value.State, &content, &evidence,
			&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
			&value.InputSHA256, &value.OutputSHA256, &value.Version, &value.CreatedBy,
			&value.SubmittedBy, &value.SubmittedAt, &value.AcceptedBy, &value.AcceptedAt,
			&value.AcknowledgedBy, &value.AcknowledgedAt, &value.CreatedAt, &value.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(content, &value.Content); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
			return nil, false, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `TEST_DATABASE_URL=postgres://... go test ./internal/handover/adapter/postgres/ -run TestHandoverList -v`
Expected: PASS.
Also run `go test ./internal/handover/...` to confirm the package compiles.

- [ ] **Step 5: Commit**

```bash
git add internal/handover/adapter/postgres/
git commit -m "feat: paginate handover list in postgres store"
```

---

### Task 6: HTTP handlers for the list endpoints

**Files:**
- Modify: `internal/platform/httpserver/reports.go` (route + `listReports` handler)
- Modify: `internal/platform/httpserver/handovers.go` (route + `listHandovers` handler)
- Modify: `internal/platform/httpserver/maintenance_helpers.go` (`parsePageSize`, `nullableString`; add `strconv` import)
- Create: `internal/platform/httpserver/reports_list_test.go`
- Create: `internal/platform/httpserver/handovers_list_test.go`

**Interfaces:**
- Consumes: `reportapp.Service.List`, `handoverapp.Service.List`, `reportapp.ReportFilter`, `handoverapp.HandoverFilter` from Tasks 2/4.
- Produces: `GET /api/v1/reports` and `GET /api/v1/handovers` handlers writing `{"items","next_cursor","has_more"}`.

- [ ] **Step 1: Write the failing handler tests**

Create `internal/platform/httpserver/reports_list_test.go`:

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

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
)

type listReportStore struct{}

func (*listReportStore) LoadExecutionContext(context.Context, identitydomain.Principal, string) (reportapp.ExecutionContext, error) {
	return reportapp.ExecutionContext{}, nil
}
func (*listReportStore) SaveDraft(context.Context, identitydomain.Principal, string, reportapp.Report, json.RawMessage) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Get(context.Context, identitydomain.Principal, string) (reportapp.Report, error) {
	return reportapp.Report{}, nil
}
func (*listReportStore) Edit(context.Context, identitydomain.Principal, string, string, reportapp.Edit) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Submit(context.Context, identitydomain.Principal, string, string, reportapp.Transition) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Approve(context.Context, identitydomain.Principal, string, string, reportapp.Transition) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) RecordCall(context.Context, identitydomain.Principal, reportapp.ExecutionContext, skawld.Generation, string, string) error {
	return nil
}
func (*listReportStore) List(context.Context, identitydomain.Principal, reportapp.ReportFilter) ([]reportapp.Report, bool, error) {
	return []reportapp.Report{{
		ID: "00000000-0000-0000-0000-00000000000a", State: "DRAFT",
		CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func listReportsHandler() http.Handler {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionReportWrite: {},
		},
	}
	return New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Reports: reportapp.Service{
			Store: &listReportStore{},
		},
	})
}

func TestListReportsEnvelope(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?site_id=s1&page_size=25", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Code, response.Body.String())
	}
	var body struct {
		Items      []reportapp.Report `json:"items"`
		NextCursor *string            `json:"next_cursor"`
		HasMore    bool               `json:"has_more"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(body.Items))
	}
	if body.NextCursor == nil {
		t.Fatal("next_cursor must be present (non-null) when has_more")
	}
	if !body.HasMore {
		t.Fatal("has_more = false, want true")
	}
}

func TestListReportsValidatesPageSize(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?page_size=500", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestListReportsRejectsBadSite(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?site_id=not-a-uuid", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
```

Create `internal/platform/httpserver/handovers_list_test.go` with the same structure, using a fake store implementing every `handoverapp.Store` method — `LoadWindowContext`, `SaveDraft`, `Get`, `Edit`, `Submit`, `Accept`, `Acknowledge`, `RecordCall`, and `List` — where `List` returns one `handoverapp.Handover` with `ID`, `ShiftStart` (`time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)`), and `State: "DRAFT"` plus `hasMore = true`. The principal gets `PermissionHandoverWrite`. Tests:

- `TestListHandoversEnvelope`: `GET /api/v1/handovers?site_id=s1&page_size=25` → 200, envelope unmarshals into `struct { Items []handoverapp.Handover `json:"items"`; NextCursor *string `json:"next_cursor"`; HasMore bool `json:"has_more"` }`, `NextCursor` non-nil, `HasMore` true.
- `TestListHandoversValidatesPageSize`: `GET /api/v1/handovers?page_size=0` → 400.
- `TestListHandoversRejectsBadSite`: `GET /api/v1/handovers?site_id=not-a-uuid` → 400.

The imports mirror the reports test: `context`, `encoding/json`, `log/slog`, `net/http`, `net/http/httptest`, `testing`, `time`, `identitydomain`, `skawld`, and `handoverapp`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/httpserver/ -run TestList -v`
Expected: FAIL — routes return 404 (no handler registered).

- [ ] **Step 3: Implement the handlers**

In `internal/platform/httpserver/maintenance_helpers.go`:

Add `"strconv"` to imports, then add:

```go
const (
	defaultPageSize = 25
	maxPageSize     = 100
)

func parsePageSize(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("page_size"))
	if raw == "" {
		return defaultPageSize, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > maxPageSize {
		writeProblem(w, http.StatusBadRequest, "Invalid Request", "page_size must be an integer between 1 and 100")
		return 0, false
	}
	return value, true
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
```

In `internal/platform/httpserver/reports.go`:

Add `router.Get("/reports", listReports(service))` as the first line of `mountReportRoutes`, and add:

```go
func listReports(service reportapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		pageSize, ok := parsePageSize(w, r)
		if !ok {
			return
		}
		filter := reportapp.ReportFilter{PageSize: pageSize}
		if siteID := r.URL.Query().Get("site_id"); siteID != "" {
			if !validUUIDParam(w, siteID, "site ID") {
				return
			}
			filter.SiteID = siteID
		}
		filter.States = r.URL.Query()["state"]
		filter.Cursor = strings.TrimSpace(r.URL.Query().Get("cursor"))
		result, err := service.List(r.Context(), principal, filter)
		if err != nil {
			writeReportError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":       result.Items,
			"next_cursor": nullableString(result.NextCursor),
			"has_more":    result.HasMore,
		})
	}
}
```

Add `"strings"` to reports.go imports.

In `internal/platform/httpserver/handovers.go`:

Add `router.Get("/handovers", listHandovers(service))` as the first line of `mountHandoverRoutes`, and add:

```go
func listHandovers(service handoverapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		pageSize, ok := parsePageSize(w, r)
		if !ok {
			return
		}
		filter := handoverapp.HandoverFilter{PageSize: pageSize}
		if siteID := r.URL.Query().Get("site_id"); siteID != "" {
			if !validUUIDParam(w, siteID, "site ID") {
				return
			}
			filter.SiteID = siteID
		}
		filter.States = r.URL.Query()["state"]
		filter.Cursor = strings.TrimSpace(r.URL.Query().Get("cursor"))
		result, err := service.List(r.Context(), principal, filter)
		if err != nil {
			writeHandoverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":       result.Items,
			"next_cursor": nullableString(result.NextCursor),
			"has_more":    result.HasMore,
		})
	}
}
```

Add `"strings"` to handovers.go imports.

> Route-ordering note: `router.Get("/reports", ...)` is a static path; chi routes it independently of `/reports/{reportID}`, so no ordering hazard. Same for `/handovers` vs `/handovers/{handoverID}` and `/handovers/prepare-draft` (static wins over the `{handoverID}` param in chi).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/httpserver/ -run TestList -v`
Expected: PASS. Then `go vet ./internal/platform/httpserver/` and `go test ./internal/...` to confirm nothing else broke.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/httpserver/
git commit -m "feat: serve report and handover list endpoints"
```

---

### Task 7: Migration 00012 — report list index

**Files:**
- Create: `migrations/00012_report_list_index.sql`

**Interfaces:**
- Produces: index `maintenance_reports_list_idx (organization_id, created_at DESC, id DESC)`.

- [ ] **Step 1: Write the migration**

Create `migrations/00012_report_list_index.sql`:

```sql
-- +goose Up
CREATE INDEX maintenance_reports_list_idx
    ON maintenance_reports(organization_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS maintenance_reports_list_idx;
```

- [ ] **Step 2: Run the migration**

Run: `make migrate-up` (requires `MIGRATION_DATABASE_URL`).
Expected: `00012_report_list_index.sql` applied, no errors.

- [ ] **Step 3: Verify the plan uses the index**

Run in `psql`: `EXPLAIN SELECT id FROM maintenance_reports WHERE organization_id = '...' ORDER BY created_at DESC, id DESC LIMIT 26;`
Expected: index scan on `maintenance_reports_list_idx`. (Manual verification; if EXPLAIN shows a different plan for small tables, that is expected.)

- [ ] **Step 4: Commit**

```bash
git add migrations/00012_report_list_index.sql
git commit -m "feat: index report list ordering for paginated queries"
```

---

### Task 8: OpenAPI spec

**Files:**
- Modify: `api/openapi.yaml`

**Interfaces:**
- Produces: `GET /api/v1/reports` and `GET /api/v1/handovers` path definitions with `site_id`, `state`, `cursor`, `page_size` parameters and the `{items, next_cursor, has_more}` 200 response.

- [ ] **Step 1: Add the `GET /api/v1/reports` path**

Insert immediately before the `  /api/v1/reports/{reportID}:` line (line 899):

```yaml
  /api/v1/reports:
    get:
      summary: List reports with site/state filters and keyset pagination
      tags: [Reports]
      operationId: listMaintenanceReports
      security: [{sessionCookie: []}, {bearerAuth: []}]
      parameters:
        - {in: query, name: site_id, required: false, schema: {type: string, format: uuid}}
        - {in: query, name: state, required: false, schema: {type: string, enum: [DRAFT, SUBMITTED, APPROVED, REJECTED]}}
        - {in: query, name: cursor, required: false, schema: {type: string}}
        - {in: query, name: page_size, required: false, schema: {type: integer, minimum: 1, maximum: 100, default: 25}}
      responses:
        "200":
          description: Paginated report library scoped to the principal.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [items, next_cursor, has_more]
                properties:
                  items:
                    type: array
                    items:
                      type: object
                      additionalProperties: false
                      required: [id, execution_id, revision, version, state, structured_content, created_at]
                      properties:
                        id: {type: string, format: uuid}
                        execution_id: {type: string, format: uuid}
                        site_id: {type: string, format: uuid}
                        revision: {type: integer}
                        version: {type: integer, format: int64}
                        state: {type: string, enum: [DRAFT, SUBMITTED, APPROVED, REJECTED]}
                        structured_content: {$ref: "#/components/schemas/ReportContent"}
                        created_at: {type: string, format: date-time}
                  next_cursor: {type: [string, "null"]}
                  has_more: {type: boolean}
        "400": {$ref: "#/components/responses/Problem"}
        "401": {$ref: "#/components/responses/Problem"}
        "403": {$ref: "#/components/responses/Problem"}
```

- [ ] **Step 2: Add the `GET /api/v1/handovers` path**

Insert immediately before the `  /api/v1/handovers/prepare-draft:` line:

```yaml
  /api/v1/handovers:
    get:
      summary: List shift handovers with site/state filters and keyset pagination
      tags: [Handovers]
      operationId: listShiftHandovers
      security: [{sessionCookie: []}, {bearerAuth: []}]
      parameters:
        - {in: query, name: site_id, required: false, schema: {type: string, format: uuid}}
        - {in: query, name: state, required: false, schema: {type: string, enum: [DRAFT, SUBMITTED, ACCEPTED, ACKNOWLEDGED]}}
        - {in: query, name: cursor, required: false, schema: {type: string}}
        - {in: query, name: page_size, required: false, schema: {type: integer, minimum: 1, maximum: 100, default: 25}}
      responses:
        "200":
          description: Paginated handover library scoped to the principal.
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [items, next_cursor, has_more]
                properties:
                  items:
                    type: array
                    items:
                      type: object
                      additionalProperties: false
                      required: [id, site_id, shift_start, shift_end, state, version, structured_content]
                      properties:
                        id: {type: string, format: uuid}
                        site_id: {type: string, format: uuid}
                        shift_start: {type: string, format: date-time}
                        shift_end: {type: string, format: date-time}
                        state: {type: string, enum: [DRAFT, SUBMITTED, ACCEPTED, ACKNOWLEDGED]}
                        version: {type: integer, format: int64}
                        structured_content: {$ref: "#/components/schemas/HandoverContent"}
                  next_cursor: {type: [string, "null"]}
                  has_more: {type: boolean}
        "400": {$ref: "#/components/responses/Problem"}
        "401": {$ref: "#/components/responses/Problem"}
        "403": {$ref: "#/components/responses/Problem"}
```

- [ ] **Step 3: Validate the YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('api/openapi.yaml')); print('valid yaml')"` (or `ruby -e "require 'yaml'; YAML.load_file('api/openapi.yaml'); puts 'valid yaml'"`).
Expected: `valid yaml` — no indentation/tab errors.

- [ ] **Step 4: Commit**

```bash
git add api/openapi.yaml
git commit -m "docs: specify report and handover list endpoints"
```

---

### Task 9: Frontend client, types, and i18n

**Files:**
- Modify: `web/src/types.ts` (add `ListPage<T>`)
- Modify: `web/src/api.ts` (`ListOptions`, `listQuery`, `fetchAll`, `reports`, `handovers`, `pendingHandovers`)
- Modify: `web/src/api.test.ts` (query building + fetchAll tests)
- Modify: `web/src/i18n/messages.ts` (add `common.loadMore`, `report.state.all`)

**Interfaces:**
- Produces:
  - `export type ListPage<T> = ListResponse<T> & { next_cursor: string | null; has_more: boolean }`
  - `export interface ListOptions { site_id?: string; state?: string[]; cursor?: string; page_size?: number }`
  - `function listQuery(options: ListOptions): string`
  - `export async function fetchAll<T>(first: ListPage<T>, fetchPage: (cursor: string) => Promise<ListPage<T>>): Promise<T[]>`
  - `api.reports(options?: ListOptions): Promise<ListPage<MaintenanceReport>>`
  - `api.handovers(options?: ListOptions): Promise<ListPage<ShiftHandover>>`
  - `api.pendingHandovers(): Promise<ShiftHandover[]>` (walks all pages with `state=DRAFT,SUBMITTED,ACCEPTED`)

- [ ] **Step 1: Write the failing client tests**

Add to `web/src/api.test.ts`:

```ts
import type { ListPage } from "./types";

describe("list query building", () => {
  it("serializes site, repeated states, cursor, and page size", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [], next_cursor: null, has_more: false }), { status: 200 }),
    );
    await api.reports({ site_id: "s1", state: ["DRAFT", "SUBMITTED"], cursor: "abc", page_size: 50 });
    const [url] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/api/v1/reports?");
    expect(String(url)).toContain("site_id=s1");
    expect(String(url)).toContain("state=DRAFT");
    expect(String(url)).toContain("state=SUBMITTED");
    expect(String(url)).toContain("cursor=abc");
    expect(String(url)).toContain("page_size=50");
  });
});

describe("fetchAll", () => {
  it("walks next_cursor pages until exhausted", async () => {
    const first: ListPage<{ id: string }> = {
      items: [{ id: "a" }],
      next_cursor: "c1",
      has_more: true,
    };
    const second: ListPage<{ id: string }> = {
      items: [{ id: "b" }, { id: "c" }],
      next_cursor: null,
      has_more: false,
    };
    const fetchPage = vi.fn().mockResolvedValue(second);
    const all = await fetchAll(first, fetchPage);
    expect(all.map((item) => item.id)).toEqual(["a", "b", "c"]);
    expect(fetchPage).toHaveBeenCalledWith("c1");
  });

  it("stops when the first page has no cursor", async () => {
    const first: ListPage<{ id: string }> = { items: [{ id: "a" }], next_cursor: null, has_more: false };
    const fetchPage = vi.fn();
    const all = await fetchAll(first, fetchPage);
    expect(all).toHaveLength(1);
    expect(fetchPage).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/api.test.ts`
Expected: FAIL — `ListPage`, `fetchAll` undefined.

- [ ] **Step 3: Implement the client**

In `web/src/types.ts`, after `ListResponse`:

```ts
export type ListPage<T> = ListResponse<T> & {
  next_cursor: string | null;
  has_more: boolean;
};
```

In `web/src/api.ts`, add before `export const api`:

```ts
export interface ListOptions {
  site_id?: string;
  state?: string[];
  cursor?: string;
  page_size?: number;
}

function listQuery(options: ListOptions): string {
  const params = new URLSearchParams();
  if (options.site_id) params.set("site_id", options.site_id);
  for (const state of options.state ?? []) params.append("state", state);
  if (options.cursor) params.set("cursor", options.cursor);
  if (options.page_size) params.set("page_size", String(options.page_size));
  const query = params.toString();
  return query ? `?${query}` : "";
}

export async function fetchAll<T>(
  first: ListPage<T>,
  fetchPage: (cursor: string) => Promise<ListPage<T>>,
): Promise<T[]> {
  const items = [...first.items];
  let cursor = first.next_cursor;
  while (cursor) {
    const page = await fetchPage(cursor);
    items.push(...page.items);
    cursor = page.next_cursor;
  }
  return items;
}
```

Replace the `reports:` and `handovers:` entries in the `api` object:

```ts
  reports: (options: ListOptions = {}) =>
    request<ListPage<MaintenanceReport>>(`/reports${listQuery(options)}`),
  handovers: (options: ListOptions = {}) =>
    request<ListPage<ShiftHandover>>(`/handovers${listQuery(options)}`),
```

Add a `pendingHandovers` entry (place it next to `handovers`):

```ts
  pendingHandovers: async () => {
    const options = { state: ["DRAFT", "SUBMITTED", "ACCEPTED"], page_size: 100 };
    const first = await request<ListPage<ShiftHandover>>(`/handovers${listQuery(options)}`);
    return fetchAll(first, (cursor) =>
      request<ListPage<ShiftHandover>>(`/handovers${listQuery({ ...options, cursor })}`)
    );
  },
```

In `web/src/i18n/messages.ts`, add near the other common keys:

```ts
  "common.loadMore": "Load more",
```

and near the report keys:

```ts
  "report.state.all": "All states",
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run src/api.test.ts`
Expected: PASS. Then `cd web && npx tsc --noEmit` to confirm types compile.

- [ ] **Step 5: Commit**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.test.ts web/src/i18n/messages.ts
git commit -m "feat: add paginated report and handover client with page walking"
```

---

### Task 10: HandoverPage — server-side site filter and load more

**Files:**
- Modify: `web/src/console/pages/HandoverPage.tsx`
- Modify: `web/src/console/pages/HandoverPage.test.tsx`

**Interfaces:**
- Consumes: `api.handovers(options)`, `ListPage<ShiftHandover>`, `useSite()` from Task 9.
- Produces: HandoverPage fetches the first 25 handovers for the selected site (ordered `shift_start DESC` server-side), renders the first as `latest`, shows a "Load more" button (`common.loadMore`) that appends the next page, and resets accumulation when the site changes.

- [ ] **Step 1: Write the failing tests**

Update the `handovers` mock in `web/src/console/pages/HandoverPage.test.tsx` to the new envelope:

```ts
    handovers: vi.fn().mockResolvedValue({ items: [fixtures.older, fixtures.latest], next_cursor: null, has_more: false }),
```

Add tests (inside the existing `describe`):

```tsx
  it("loads more history when present", async () => {
    vi.mocked(api.handovers)
      .mockResolvedValueOnce({
        items: [fixtures.latest],
        next_cursor: "c1",
        has_more: true,
      })
      .mockResolvedValueOnce({
        items: [fixtures.older],
        next_cursor: null,
        has_more: false,
      });
    renderPage();
    const button = await screen.findByRole("button", { name: /load more/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(api.handovers).toHaveBeenLastCalledWith({
        site_id: "s1",
        page_size: 25,
        cursor: "c1",
      });
    });
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/HandoverPage.test.tsx`
Expected: FAIL — no "Load more" button.

- [ ] **Step 3: Implement the page changes**

In `web/src/console/pages/HandoverPage.tsx`:

Replace the `handovers` query and the `items` memo (the block from `const handovers = useQuery(...)` through the `const history = items.slice(1);` line) with page accumulation. `siteId` is already declared above via `const { siteId } = useSite();` — do not redeclare it:

```tsx
  const firstPage = useQuery(
    () =>
      api.handovers(
        siteId ? { site_id: siteId, page_size: 25 } : { page_size: 25 },
      ),
    [siteId],
  );
  const [history, setHistory] = useState<ShiftHandover[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);

  useEffect(() => {
    setHistory([]);
    setNextCursor(firstPage.data?.next_cursor ?? null);
  }, [firstPage.data, siteId]);

  const items = useMemo(
    () => [...(firstPage.data?.items ?? []), ...history],
    [firstPage.data, history],
  );
  const latest = items[0];
  const historyList = items.slice(1);
  const loadMore = async () => {
    if (!nextCursor) return;
    const page = await api.handovers(
      siteId
        ? { site_id: siteId, cursor: nextCursor, page_size: 25 }
        : { cursor: nextCursor, page_size: 25 },
    );
    setHistory((prev) => [...prev, ...page.items]);
    setNextCursor(page.next_cursor);
  };
```

Update imports: add `useEffect`, `useState` from `"react"`; add `ShiftHandover` to the types import.

Replace every reference to the old `handovers` query result:
- `handovers.error` → `firstPage.error`
- `handovers.loading && !handovers.data` → `firstPage.loading && !firstPage.data`
- `handovers.refetch()` → `firstPage.refetch()` (two call sites: the error-state retry button and the `refetch` passed to the submit/accept/acknowledge/prepare `useCommand` `onSuccess` handlers)
- `history.length > 0` → `historyList.length > 0`
- `history.map(...)` → `historyList.map(...)`
- The `latest`/`history` consts in the query block are replaced by the accumulator versions from Step 3, so `transitions` keeps using `latest` unchanged.

In the render, add the load-more control immediately after the history panel's closing `</section>` (the one containing `handover.noHistory`) and before the `</>` fragment close:

```tsx
        {nextCursor ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void loadMore()}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
          </button>
        ) : null}
```

> The page already computes `latest`/`history` client-side from `items`; server ordering (`shift_start DESC`) makes `items[0]` the latest shift. The old client-side `scoped` filter is removed because the server now filters by `site_id`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run src/console/pages/HandoverPage.test.tsx`
Expected: PASS (existing tests + new load-more test).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/pages/HandoverPage.tsx web/src/console/pages/HandoverPage.test.tsx
git commit -m "feat: paginate handover history with server-side site filter"
```

---

### Task 11: ReportsPage — state filter and load more

**Files:**
- Modify: `web/src/console/pages/ReportsPage.tsx`
- Modify: `web/src/console/pages/ReportsPage.test.tsx`

**Interfaces:**
- Consumes: `api.reports(options)`, `ListPage<MaintenanceReport>`, `reportStateLabelKey`/`reportStateTone` (already imported).
- Produces: ReportsPage fetches paginated reports; a state `<select>` (All/Draft/Submitted/Approved) refetches with `state=[...]`; a "Load more" button appends the next page.

- [ ] **Step 1: Write the failing tests**

Update the `reports` mock in `web/src/console/pages/ReportsPage.test.tsx`:

```ts
    reports: vi.fn().mockResolvedValue({
      items: [ /* keep existing two fixture reports */ ],
      next_cursor: null,
      has_more: false
    }),
```

Add tests:

```tsx
  it("applies the state filter server-side", async () => {
    renderPage();
    const select = await screen.findByRole("combobox", { name: /state/i });
    fireEvent.change(select, { target: { value: "APPROVED" } });
    await waitFor(() => {
      expect(api.reports).toHaveBeenLastCalledWith({
        state: ["APPROVED"],
        page_size: 25,
      });
    });
  });

  it("loads more pages when available", async () => {
    vi.mocked(api.reports)
      .mockResolvedValueOnce({
        items: [{ id: "r1", execution_id: "e1", revision: 1, version: 1, state: "DRAFT", structured_content: { summary: "Pump inspection", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] }],
        next_cursor: "c1",
        has_more: true,
      })
      .mockResolvedValueOnce({
        items: [{ id: "r2", execution_id: null, revision: 2, version: 1, state: "APPROVED", structured_content: { summary: "Bearing replacement", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] }],
        next_cursor: null,
        has_more: false,
      });
    renderPage();
    const button = await screen.findByRole("button", { name: /load more/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText("Bearing replacement")).toBeTruthy();
    });
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/ReportsPage.test.tsx`
Expected: FAIL — no combobox / load more button.

- [ ] **Step 3: Implement the page changes**

In `web/src/console/pages/ReportsPage.tsx`:

Add to imports: `useEffect, useMemo, useState` from `"react"`; `ListOptions` from `"../../api"`.

Replace the query block:

```tsx
  const [stateFilter, setStateFilter] = useState<string>("ALL");
  const [history, setHistory] = useState<MaintenanceReport[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);

  const options = useMemo<ListOptions>(
    () => ({
      ...(stateFilter !== "ALL" ? { state: [stateFilter] } : {}),
      page_size: 25
    }),
    [stateFilter],
  );
  const reports = useQuery(() => api.reports(options), [options]);
  const rows = useMemo(
    () => [...(reports.data?.items ?? []), ...history],
    [reports.data, history],
  );

  useEffect(() => {
    setHistory([]);
    setNextCursor(reports.data?.next_cursor ?? null);
  }, [reports.data, stateFilter]);

  const loadMore = async () => {
    if (!nextCursor) return;
    const page = await api.reports({ ...options, cursor: nextCursor });
    setHistory((prev) => [...prev, ...page.items]);
    setNextCursor(page.next_cursor);
  };
```

Update the render:
- Count span: `{rows.length}`
- Add the state filter select inside the `panel-heading` div, before the count span:

```tsx
            <select
              aria-label="State"
              className="filter-select"
              value={stateFilter}
              onChange={(event) => setStateFilter(event.target.value)}
            >
              <option value="ALL">{t("report.state.all")}</option>
              <option value="DRAFT">{t("report.state.draft")}</option>
              <option value="SUBMITTED">{t("report.state.submitted")}</option>
              <option value="APPROVED">{t("report.state.approved")}</option>
            </select>
```

- `rows={reports.data ?? []}` → `rows={rows}`
- Add the load-more button after the `</DataTable>`:

```tsx
        {nextCursor ? (
          <button
            type="button"
            className="secondary-button"
            onClick={() => void loadMore()}
            style={{ marginTop: 12 }}
          >
            {t("common.loadMore")}
          </button>
        ) : null}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run src/console/pages/ReportsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/console/pages/ReportsPage.tsx web/src/console/pages/ReportsPage.test.tsx
git commit -m "feat: paginate report library with server-side state filter"
```

---

### Task 12: DashboardPage — pending handover count via page walking

**Files:**
- Modify: `web/src/console/pages/DashboardPage.tsx`
- Modify: `web/src/console/pages/DashboardPage.test.tsx`

**Interfaces:**
- Consumes: `api.pendingHandovers()` from Task 9.
- Produces: `pendingHandoverCount` = `handovers.data?.length ?? 0` (all returned handovers are non-ACKNOWLEDGED).

- [ ] **Step 1: Write the failing test**

Update the mock in `web/src/console/pages/DashboardPage.test.tsx`: replace the `handovers` entry with

```ts
    pendingHandovers: vi.fn().mockResolvedValue([
      {
        id: "h1",
        site_id: "s1",
        shift_start: new Date().toISOString(),
        shift_end: new Date().toISOString(),
        state: "SUBMITTED",
        version: 1,
        structured_content: {},
        evidence: [],
        provider: "skawld-copilot"
      }
    ]),
```

Add a test (asserts via the MetricCard link, whose text contains the label and the value):

```tsx
  it("counts pending handovers from the paginated list", async () => {
    renderDashboard();
    const pendingLink = await screen.findByRole("link", { name: /pending handovers/i });
    expect(pendingLink.textContent).toContain("1");
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: FAIL — `pendingHandovers` undefined (mock module missing the key) or count renders 0.

- [ ] **Step 3: Implement the page changes**

In `web/src/console/pages/DashboardPage.tsx`:

Replace:

```tsx
  const handovers = useQuery(() => api.handovers());
```

with:

```tsx
  const handovers = useQuery(() => api.pendingHandovers());
```

Replace:

```tsx
  const pendingHandovers =
    handovers.data?.items.filter((handover) => handover.state !== "ACKNOWLEDGED").length ?? 0;
```

with:

```tsx
  const pendingHandovers = handovers.data?.length ?? 0;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/console/pages/DashboardPage.tsx web/src/console/pages/DashboardPage.test.tsx
git commit -m "feat: count pending handovers across paginated pages"
```

---

### Task 13: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Run the Go suite**

Run: `make check` (runs `gofmt`, `go vet ./...`, `go test ./...`).
Expected: all pass.

- [ ] **Step 2: Run the web suite**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: all pass (lint clean, all vitest suites green, TypeScript build succeeds).

- [ ] **Step 3: Smoke-test the endpoints against a live stack**

With `compose up` running and a seeded principal that has `report:write` and `handover:write`:

```bash
# 200 envelope, newest first
curl -s 'http://localhost:8080/api/v1/reports?page_size=25' -H 'Cookie: <session>'
curl -s 'http://localhost:8080/api/v1/handovers?site_id=<site>&page_size=25' -H 'Cookie: <session>'
# 400 on bad params
curl -s -o /dev/null -w '%{http_code}\n' 'http://localhost:8080/api/v1/reports?page_size=500' -H 'Cookie: <session>'
```

Expected: first two return 200 with `next_cursor`/`has_more`; last returns 400.

- [ ] **Step 4: Confirm the two pages load in the console**

Open `/reports`, `/handovers`, and `/` (dashboard) in the pilot console. Expected: no "Something went wrong" error states; reports/handovers lists render; dashboard shows the pending-handover KPI.

---

## Self-Review Notes

- **Spec coverage:** design doc requirements — filters (site_id, state), cursor + page_size, envelope, org/site scoping, migration, OpenAPI, frontend wiring for all three pages — map 1:1 to Tasks 1-13.
- **Adaptation vs design doc:** the design's "extend `test/contract/sdk/sdk_contract_test.go`" is not applicable — that file pins the Go SDK module version, not the HTTP API. Wire-contract coverage is provided instead by the handler tests (Task 6) which assert status codes and the `{items, next_cursor, has_more}` envelope.
- **Cursor plumbing:** the application layer decodes the base64url cursor and hands the store a plain `"<RFC3339Nano>|<uuid>"` string to keep the store free of codec imports; Task 3 and Task 5 both split on the last `|` (UUIDs never contain `|`).
- **Type consistency:** `ReportFilter`/`HandoverFilter` + `ListResult` are defined per application package (no shared generics needed); the store signatures return `([]T, bool, error)` consistently.
- **Existing fakes:** adding `List` to both `Store` interfaces breaks the fakes in `service_test.go`; each task updates its own fake first (Step 1) so the suite compiles at every commit.
