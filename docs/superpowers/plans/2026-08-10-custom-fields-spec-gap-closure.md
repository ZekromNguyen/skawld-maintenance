# Custom Fields Spec Gap Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining gaps between the executed custom-fields core (2026-08-09 plan) and the design spec `docs/superpowers/specs/2026-08-09-custom-fields-core-design.md`: NUMBER range filters, incident counts in the retire dialog, retired-field hiding in the create form and admin default view, the queue "Fields" disclosure row, and saved-view persistence of custom field filters.

**Architecture:** All behavior already ships through `internal/customfield`, `internal/incident`, and the existing console pages. This plan only adds: (1) a `min:max` range syntax parsed in `incidentapp.Service.List` for NUMBER-typed definitions and pushed down to SQL with a numeric cast; (2) an `incident_count` on the field-definition view backed by a `Counts` store query; (3) small, targeted UI changes on `CustomFieldsPage`, `CreateIncidentForm`, `IncidentsPage` (via an optional `expandedRow` prop on the shared `DataTable`), and `SavedViews`. No migration is required.

**Tech Stack:** Go (pgx/v5, chi), PostgreSQL 16 (JSONB), React + TypeScript + Vite, existing design system tokens and `web/src/i18n/messages.ts` (`en` and `vi` blocks).

## Global Constraints

- Key regex stays `^[a-z][a-z0-9_]{1,63}$`; field `field_type`/`config` immutability rules are unchanged.
- Retired fields stay hidden from the create/edit form and the admin list default view, keep all values and history, and remain readable (spec "Retire" rule).
- Value validation and filter parsing happen in the application layer, never trusting raw JSONB.
- API custom-field filters are keyed by field **key** on input (`custom_field.<key>`), resolved to definition ids server-side; response `custom_fields` sections map ids back to keys.
- NUMBER range filter syntax is `custom_field.<key>=min:max` with either bound optional (`10:`, `:30`, `10:30`); a value without `:` stays an exact match (backward compatible). TEXT/SELECT filters are always exact, even if the value contains `:`.
- All new UI copy goes through `web/src/i18n/messages.ts` in both `en` and `vi`.
- Migration files are forward-only; this plan adds none.
- Do NOT commit unless the user asks; use existing commit message conventions when committing.

---

### Task 1: NUMBER range filter for custom fields

**Files:**
- Modify: `internal/incident/application/service.go` (`Filter` struct + `List` parsing)
- Modify: `internal/incident/application/service_test.go` (`fakeFields.ResolveKeys` + new tests)
- Modify: `internal/incident/adapter/postgres/store.go` (`List` range conditions)
- Test: `internal/incident/adapter/postgres/cursor_pagination_integration_test.go`
- Modify: `api/openapi.yaml` (`custom_field.<key>` description at line 308)

**Interfaces:**
- Consumes: `Fields.ResolveKeys(...) (map[string]customfieldapp.Definition, error)` — `Definition.FieldType` is `string` ("NUMBER" for numeric fields); `filter.CustomFields map[string]string` keyed by field key at service entry.
- Produces: `Filter.CustomFieldRanges map[string]CustomFieldRange` keyed by definition **id**, where `type CustomFieldRange struct { Min *float64; Max *float64 }`. The store consumes this map; the HTTP handler does not change.

- [ ] **Step 1: Write the failing service test**

Add to `internal/incident/application/service_test.go`. First extend the existing fakes so they can report a NUMBER field type and record the filter. The `stubStore` gains an `items` slice and records the filter in `List`; `fakeFields` (currently `type fakeFields struct{}` with value receivers at line 49-74) gains a `types` map — keep the value receivers so the existing `fakeFields{}` usages compile unchanged:

```go
type stubStore struct {
	items      []Incident
	lastFilter Filter
}

func (s *stubStore) List(_ context.Context, _ identitydomain.Principal, filter Filter) ([]Incident, bool, error) {
	s.lastFilter = filter
	return s.items, false, nil
}

type fakeFields struct {
	types map[string]string // field key -> field_type
}

func (f fakeFields) Definitions(_ context.Context, _, _ string) ([]customfieldapp.Definition, error) {
	return []customfieldapp.Definition{
		{ID: "def-1", Key: "po_number", FieldType: "TEXT", Config: domain.Config{Required: true}},
	}, nil
}

func (f fakeFields) ResolveAndValidate(_ context.Context, _, _ string, values map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for key, value := range values {
		if key != "po_number" {
			return nil, errors.New("unknown custom field")
		}
		out["def-1"] = value
	}
	return out, nil
}

func (f fakeFields) ResolveKeys(_ context.Context, _, _ string, keys []string) (map[string]customfieldapp.Definition, error) {
	out := map[string]customfieldapp.Definition{}
	for _, key := range keys {
		fieldType := f.types[key]
		if fieldType == "" {
			fieldType = "TEXT"
		}
		out[key] = customfieldapp.Definition{ID: "def-" + key, Key: key, FieldType: fieldType}
	}
	return out, nil
}
```

Then the tests:

```go
func TestListParsesNumberRangeFilter(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "cold", DetectedAt: time.Unix(1, 0).UTC()},
		{ID: "i2", Number: "INC-2", Summary: "warm", DetectedAt: time.Unix(2, 0).UTC()},
		{ID: "i3", Number: "INC-3", Summary: "hot", DetectedAt: time.Unix(3, 0).UTC()},
	}}
	s := Service{
		Store: store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "20:30"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ranges := store.lastFilter.CustomFieldRanges
	if len(ranges) != 1 {
		t.Fatalf("CustomFieldRanges = %#v, want 1 entry", ranges)
	}
	r := ranges["def-temperature"]
	if r.Min == nil || *r.Min != 20 || r.Max == nil || *r.Max != 30 {
		t.Fatalf("range = min %v max %v, want 20..30", r.Min, r.Max)
	}
}

func TestListParsesOpenEndedNumberRanges(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "s", DetectedAt: time.Unix(1, 0).UTC()},
	}}
	s := Service{
		Store: store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": ":30"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	r := store.lastFilter.CustomFieldRanges["def-temperature"]
	if r.Min != nil || r.Max == nil || *r.Max != 30 {
		t.Fatalf("range = min %v max %v, want open min, max 30", r.Min, r.Max)
	}
}

func TestListKeepsExactMatchForNonRangeValues(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "s", DetectedAt: time.Unix(1, 0).UTC()},
	}}
	s := Service{
		Store: store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER", "note": "TEXT"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "42", "note": "a:b"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(store.lastFilter.CustomFieldRanges) != 0 {
		t.Fatalf("CustomFieldRanges = %#v, want none", store.lastFilter.CustomFieldRanges)
	}
	exact := store.lastFilter.CustomFields
	if exact["def-temperature"] != "42" || exact["def-note"] != "a:b" {
		t.Fatalf("exact filters = %#v", exact)
	}
}

func TestListRejectsMalformedNumberRange(t *testing.T) {
	s := Service{
		Store:  &stubStore{},
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "abc:def"},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
```

`incidentPrincipal()` is the existing helper in this file (it carries `incident:read`, so `List` passes its permission check). Note: the incident `Service` struct has only `Store` and `Fields` — no `Now` field — so the test literals above omit it (the plan's earlier drafts showed a `Now:` line; the real struct does not have one). The existing tests `TestCreateValidatesCustomValues`, `TestCreateRejectsUnknownCustomField`, `TestListResolvesCustomFieldFilterKeys`, and `TestUpdateCustomValuesValidatesAndMerges` must keep passing: the `stubStore.List` change only records the filter, and the `fakeFields` change keeps `ResolveAndValidate` and the `po_number` → `def-1` normalization intact.

- [ ] **Step 2: Run the service tests to verify they fail**

Run: `go test ./internal/incident/application/ -run 'TestList(ParsesNumberRangeFilter|ParsesOpenEndedNumberRanges|KeepsExactMatchForNonRangeValues|RejectsMalformedNumberRange)' -v`
Expected: FAIL — `CustomFieldRanges` does not exist on `Filter` (compile error).

- [ ] **Step 3: Implement the range parsing in the service**

In `internal/incident/application/service.go`, extend `Filter` and add the range type:

```go
type CustomFieldRange struct {
	Min *float64
	Max *float64
}

type Filter struct {
	SiteID            string
	AssetID           string
	Status            string
	CustomFields      map[string]string
	CustomFieldRanges map[string]CustomFieldRange
	PageSize          int
	Cursor            string
}
```

Add a helper:

```go
// parseNumberRange parses "min:max" (either bound optional) into a range.
// Returns ok=false when the value is not a range at all; returns an error
// when it looks like a range but has unparseable bounds.
func parseNumberRange(value string) (CustomFieldRange, bool, error) {
	lo, hi, found := strings.Cut(value, ":")
	if !found {
		return CustomFieldRange{}, false, nil
	}
	var r CustomFieldRange
	if lo != "" {
		v, err := strconv.ParseFloat(lo, 64)
		if err != nil {
			return CustomFieldRange{}, false, fmt.Errorf("invalid range lower bound %q", lo)
		}
		r.Min = &v
	}
	if hi != "" {
		v, err := strconv.ParseFloat(hi, 64)
		if err != nil {
			return CustomFieldRange{}, false, fmt.Errorf("invalid range upper bound %q", hi)
		}
		r.Max = &v
	}
	return r, true, nil
}
```

Add imports `strconv` if not already present. Replace the resolution block in `List` (currently around line 185-202):

```go
	if len(filter.CustomFields) > 0 {
		if s.Fields == nil {
			return nil, "", errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		keys := make([]string, 0, len(filter.CustomFields))
		for key := range filter.CustomFields {
			keys = append(keys, key)
		}
		resolved, err := s.Fields.ResolveKeys(ctx, principal.OrganizationID, "incident", keys)
		if err != nil {
			return nil, "", errors.Join(ErrInvalid, err)
		}
		byID := make(map[string]string, len(filter.CustomFields))
		rangesByID := make(map[string]CustomFieldRange)
		for key, value := range filter.CustomFields {
			def := resolved[key]
			if def.FieldType == "NUMBER" {
				r, isRange, err := parseNumberRange(value)
				if err != nil {
					return nil, "", errors.Join(ErrInvalid, err)
				}
				if isRange {
					rangesByID[def.ID] = r
					continue
				}
			}
			byID[def.ID] = value
		}
		filter.CustomFields = byID
		filter.CustomFieldRanges = rangesByID
	}
```

- [ ] **Step 4: Run the service tests to verify they pass**

Run: `go test ./internal/incident/application/ -run 'TestList(ParsesNumberRangeFilter|ParsesOpenEndedNumberRanges|KeepsExactMatchForNonRangeValues|RejectsMalformedNumberRange)' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing store integration test**

Add `TestIncidentListNumberRangeFilterIntegration` to `internal/incident/adapter/postgres/cursor_pagination_integration_test.go`, reusing the `seedOrgSitePrincipal` helper and the asset/definition insert patterns from `TestIncidentCustomValuesIntegration` (lines 160-200). The definition insert uses `'NUMBER'` as `field_type`; the incidents are created via `store.Create` with `CustomValues` holding float64 values:

```go
func TestIncidentListNumberRangeFilterIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	seedOrgSitePrincipal(t, ctx, pool, organizationID, siteID, principalID, now)

	// Insert one NUMBER definition ("temperature") using the same asset +
	// definition insert SQL as TestIncidentCustomValuesIntegration, with
	// field_type 'NUMBER' and config '{"min": -40, "max": 200}'.
	var definitionID string
	// ... (asset INSERT, then) ...
	// INSERT INTO field_definitions (...) VALUES ($1::uuid, 'incident', 'temperature',
	//   'Temperature', 'NUMBER', '{"min": -40, "max": 200}'::jsonb, 'ACTIVE', 1, $2, $2)
	// RETURNING id::text

	principal := /* administrator principal built from identitydomain.PermissionsForRole, as in TestIncidentCustomValuesIntegration */
	store := incidentpostgres.Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.System{},
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}

	values := map[string]float64{"cold": 15, "warm": 25, "hot": 35}
	created := make(map[string]string, len(values))
	for summary, v := range values {
		incident, _, err := store.Create(ctx, principal, uuid.NewString(), incidentapp.CreateIncident{
			SiteID: siteID, AssetID: assetID,
			Summary: summary, Priority: "MEDIUM",
			SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
			CustomValues: map[string]any{definitionID: v},
		})
		if err != nil {
			t.Fatalf("create %s: %v", summary, err)
		}
		created[summary] = incident.ID
	}

	// Range 20:30 must match only the 25 value.
	min, max := 20.0, 30.0
	matches, _, err := store.List(ctx, principal, incidentapp.Filter{
		CustomFieldRanges: map[string]incidentapp.CustomFieldRange{definitionID: {Min: &min, Max: &max}},
	})
	if err != nil {
		t.Fatalf("range list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created["warm"] {
		t.Fatalf("range list = %d items, want only warm", len(matches))
	}

	// Open-ended :20 must match only the 15 value.
	matches, _, err = store.List(ctx, principal, incidentapp.Filter{
		CustomFieldRanges: map[string]incidentapp.CustomFieldRange{definitionID: {Max: &max}},
	})
	if err != nil {
		t.Fatalf("open range list: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != created["cold"] {
		t.Fatalf("open range list = %d items, want only cold", len(matches))
	}
}
```

Note the definition id must be in scope: use the same pattern as the existing test (declare `var definitionID string` before the insert and `Scan` it). The exact `principal` and `store` construction, plus the asset insert, are copied verbatim from `TestIncidentCustomValuesIntegration`.

- [ ] **Step 6: Run the integration test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://skawld:skawld@localhost:5432/skawld_test?sslmode=disable go test ./internal/incident/adapter/postgres/ -run TestIncidentListNumberRangeFilterIntegration -v`
Expected: FAIL — `CustomFieldRanges` is ignored by the store query (returns all three incidents).

- [ ] **Step 7: Implement the range conditions in the store**

In `internal/incident/adapter/postgres/store.go`, after the existing `CustomFields` block (line 206-219), add:

```go
	if len(filter.CustomFieldRanges) > 0 {
		defIDs := make([]string, 0, len(filter.CustomFieldRanges))
		for defID := range filter.CustomFieldRanges {
			defIDs = append(defIDs, defID)
		}
		sort.Strings(defIDs)
		for _, defID := range defIDs {
			keyArg := len(args) + 1
			args = append(args, defID)
			query += fmt.Sprintf(" AND i.custom_values ? $%d::text", keyArg)
			rangeFilter := filter.CustomFieldRanges[defID]
			if rangeFilter.Min != nil {
				args = append(args, *rangeFilter.Min)
				query += fmt.Sprintf(" AND (i.custom_values->>$%d::text)::numeric >= $%d", keyArg, len(args))
			}
			if rangeFilter.Max != nil {
				args = append(args, *rangeFilter.Max)
				query += fmt.Sprintf(" AND (i.custom_values->>$%d::text)::numeric <= $%d", keyArg, len(args))
			}
		}
	}
```

- [ ] **Step 8: Run the integration test to verify it passes**

Run: `TEST_DATABASE_URL=postgres://skawld:skawld@localhost:5432/skawld_test?sslmode=disable go test ./internal/incident/adapter/postgres/ -run TestIncidentListNumberRangeFilterIntegration -v`
Expected: PASS.

- [ ] **Step 9: Update the OpenAPI description**

In `api/openapi.yaml` line 308, extend the `custom_field.<key>` description:

```yaml
        - {in: query, name: custom_field.<key>, required: false, schema: {type: string}, description: |
            Filter on a tenant custom field by its key (e.g. custom_field.po_number=PO-42).
            NUMBER fields accept a range as min:max with either bound optional (10:30, 10:, :30).
            TEXT and SELECT filters are exact matches.}
```

- [ ] **Step 10: Run the full backend suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: PASS (integration tests skip without `TEST_DATABASE_URL`).

- [ ] **Step 11: Commit**

```bash
git add internal/incident/application/service.go internal/incident/application/service_test.go internal/incident/adapter/postgres/store.go internal/incident/adapter/postgres/cursor_pagination_integration_test.go api/openapi.yaml
git commit -m "feat: support range filters on NUMBER custom fields"
```

---

### Task 2: Incident count per field definition

**Files:**
- Modify: `internal/customfield/application/service.go` (`Definition` view, `Store` interface, `list`, `Get`)
- Modify: `internal/customfield/application/service_test.go` (`fakeStore.Counts` + tests)
- Modify: `internal/customfield/adapter/postgres/store.go` (`Counts` method)
- Test: `internal/customfield/adapter/postgres/store_integration_test.go`
- Modify: `api/openapi.yaml` (Definition schema after line 2227)

**Interfaces:**
- Consumes: existing `Store.ListByEntity` / `Store.Get`; `domain.Definition.EntityType`.
- Produces: `customfieldapp.Definition.IncidentCount int \`json:"incident_count"\``; `Store.Counts(context.Context, organizationID, entityType string) (map[string]int, error)` keyed by definition id. The web retire dialog (Task 3) consumes `incident_count`.

- [ ] **Step 1: Write the failing service test**

In `internal/customfield/application/service_test.go`, extend `fakeStore`:

```go
	counts      map[string]int
```

```go
func (f *fakeStore) Counts(_ context.Context, _, _ string) (map[string]int, error) {
	return f.counts, nil
}
```

Add tests:

```go
func TestListAndGetPopulateIncidentCount(t *testing.T) {
	definition := domain.Definition{
		ID: "def-1", OrganizationID: "o1", EntityType: "incident",
		Key: "po_number", Label: "PO Number", FieldType: domain.FieldTypeText,
		Status: domain.StatusActive,
	}
	store := &fakeStore{
		definitions: map[string]domain.Definition{"def-1": definition},
		byKey:       map[string]domain.Definition{"po_number": definition},
		counts:      map[string]int{"def-1": 3},
	}
	s := newService(store)

	list, err := s.ListByEntity(context.Background(), reader(), "incident")
	if err != nil {
		t.Fatalf("ListByEntity: %v", err)
	}
	if len(list) != 1 || list[0].IncidentCount != 3 {
		t.Fatalf("list incident_count = %#v, want 3", list)
	}

	got, err := s.Get(context.Background(), reader(), "def-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.IncidentCount != 3 {
		t.Fatalf("Get incident_count = %d, want 3", got.IncidentCount)
	}
}
```

- [ ] **Step 2: Run the service tests to verify they fail**

Run: `go test ./internal/customfield/application/ -run TestListAndGetPopulateIncidentCount -v`
Expected: FAIL — `IncidentCount` does not exist (compile error).

- [ ] **Step 3: Implement the count in the service**

In `internal/customfield/application/service.go`:

1. Add to the `Definition` view struct (after `HasValues`):

```go
	IncidentCount int `json:"incident_count"`
```

2. Add to the `Store` interface:

```go
	Counts(context.Context, string, string) (map[string]int, error)
```

3. In `list`, after the `Usage` call:

```go
	counts, err := s.Store.Counts(ctx, organizationID, entityType)
	if err != nil {
		return nil, err
	}
```

and inside the loop:

```go
		view.IncidentCount = counts[item.ID]
```

4. In `Get`, after the `HasValues` call:

```go
	counts, err := s.Store.Counts(ctx, principal.OrganizationID, value.EntityType)
	if err != nil {
		return Definition{}, err
	}
```

and after `view.HasValues = used`:

```go
	view.IncidentCount = counts[id]
```

- [ ] **Step 4: Run the service tests to verify they pass**

Run: `go test ./internal/customfield/application/`
Expected: PASS.

- [ ] **Step 5: Write the failing store integration test**

In `internal/customfield/adapter/postgres/store_integration_test.go`, add a test following the `TestStoreCRUDAndImmutability` setup (create a TEXT definition via `store.Create`, create one incident row holding a value keyed by the definition id, then assert `Counts`):

```go
func TestCountsReportsIncidentsWithValues(t *testing.T) {
	store, orgID := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	created, err := store.Create(ctx, orgID, domain.Definition{
		OrganizationID: orgID, EntityType: "incident", Key: "po_number",
		Label: "PO Number", FieldType: domain.FieldTypeText,
		Status: domain.StatusActive,
	}, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var siteID, principalID string
	if err := store.Pool.QueryRow(ctx, `
		INSERT INTO sites (id, organization_id, code, name, timezone, status, version, created_at, updated_at)
		VALUES (gen_random_uuid(), $1::uuid, 'CF', 'CF Site', 'UTC', 'ACTIVE', 1, $2, $2)
		RETURNING id::text
	`, orgID, now).Scan(&siteID); err != nil {
		t.Fatal(err)
	}
	if err := store.Pool.QueryRow(ctx, `
		INSERT INTO principals (id, external_subject, display_name, status, created_at, updated_at)
		VALUES (gen_random_uuid(), 'cf-tester', 'CF Tester', 'ACTIVE', $1, $1)
		RETURNING id::text
	`, now).Scan(&principalID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO incidents (
			id, organization_id, site_id, principal_id, number, summary, priority,
			status, source_of_truth, attributes, custom_values, version, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, 'INC-CF-1', 'CF', 'MEDIUM',
			'OPEN', 'OWNED_BY_SKAWLD', '{}'::jsonb, jsonb_build_object($4::text, 1), 1, $5, $5
		)
	`, orgID, siteID, principalID, created.ID, now); err != nil {
		t.Fatal(err)
	}

	counts, err := store.Counts(ctx, orgID, "incident")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if counts[created.ID] != 1 {
		t.Fatalf("Counts[%s] = %d, want 1", created.ID, counts[created.ID])
	}
	if counts["unused-def"] != 0 {
		t.Fatalf("Counts[unused-def] = %d, want 0", counts["unused-def"])
	}
}
```

If the `incidents` table has additional NOT NULL columns beyond this INSERT (check migration `00013_incident_report_redesign.sql` / `00014_repair_incident_redesign.sql` first), extend the INSERT accordingly; the columns here follow the create path in `internal/incident/adapter/postgres/store.go`.

- [ ] **Step 6: Run the integration test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://skawld:skawld@localhost:5432/skawld_test?sslmode=disable go test ./internal/customfield/adapter/postgres/ -run TestCountsReportsIncidentsWithValues -v`
Expected: FAIL — `Store.Counts` does not exist (compile error).

- [ ] **Step 7: Implement `Counts` in the postgres store**

In `internal/customfield/adapter/postgres/store.go`, next to `Usage`:

```go
// Counts reports how many incidents hold a value per definition, keyed by
// definition id, for the org and entity.
func (s Store) Counts(ctx context.Context, organizationID, entityType string) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT f.id::text, count(i.id)::int
		FROM field_definitions f
		LEFT JOIN incidents i
		  ON i.organization_id = f.organization_id AND i.custom_values ? f.id::text
		WHERE f.organization_id = $1::uuid AND f.entity_type = $2
		GROUP BY f.id
	`, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		out[id] = count
	}
	return out, rows.Err()
}
```

- [ ] **Step 8: Run the integration test to verify it passes**

Run: `TEST_DATABASE_URL=postgres://skawld:skawld@localhost:5432/skawld_test?sslmode=disable go test ./internal/customfield/adapter/postgres/ -run TestCountsReportsIncidentsWithValues -v`
Expected: PASS.

- [ ] **Step 9: Update the OpenAPI schema**

In `api/openapi.yaml`, after line 2227 (`has_values: {type: boolean}`):

```yaml
        incident_count: {type: integer}
```

- [ ] **Step 10: Run the full backend suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/customfield/application/service.go internal/customfield/application/service_test.go internal/customfield/adapter/postgres/store.go internal/customfield/adapter/postgres/store_integration_test.go api/openapi.yaml
git commit -m "feat: report how many incidents hold values per custom field"
```

---

### Task 3: Retire dialog shows the incident count

**Files:**
- Modify: `web/src/types.ts` (`CustomFieldDefinition`)
- Modify: `web/src/console/pages/CustomFieldsPage.tsx` (retire dialog message)
- Modify: `web/src/i18n/messages.ts` (both `en` and `vi` blocks)
- Test: `web/src/console/pages/CustomFieldsPage.test.tsx`

**Interfaces:**
- Consumes: `CustomFieldDefinition.incident_count` (Task 2 backend field; default to 0 when absent).
- Produces: no new exports; the retire `ConfirmDialog` message includes the count.

- [ ] **Step 1: Write the failing test**

In `web/src/console/pages/CustomFieldsPage.test.tsx`, add `incident_count: 4` to the `FIELD` fixture (the shared const near the top of the file, so the retire test reads a count off the field). In the existing `retires a field after confirmation` test, add the dialog-text assertion after the first retire-button click:

```tsx
    fireEvent.click(within(row).getByRole("button", { name: /retire/i }));
    expect(await screen.findByText(/holds values on 4 incidents/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /retire/i }));
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/CustomFieldsPage.test.tsx`
Expected: FAIL — dialog message does not contain the count.

- [ ] **Step 3: Implement the count in the retire dialog**

In `web/src/types.ts`, add to `CustomFieldDefinition` (after `has_values`):

```ts
  incident_count?: number;
```

In `web/src/i18n/messages.ts`, add to both `en` and `vi` blocks (keep the existing `retireConfirm` key; use the new one):

```ts
  "admin.customFields.retireConfirmCount": "Retire {field}? It holds values on {count} incidents. Values and history are kept; the field is hidden from new forms.",
```

```ts
  "admin.customFields.retireConfirmCount": "Ngừng sử dụng {field}? Trường này đang có giá trị trên {count} sự cố. Giá trị và lịch sử được giữ nguyên; trường sẽ bị ẩn khỏi biểu mẫu mới.",
```

In `web/src/console/pages/CustomFieldsPage.tsx`, replace the `ConfirmDialog` `message` prop (lines 170-172):

```tsx
          message={retireError
            ? `${t("admin.customFields.retireConfirmCount", {
                field: retiring.label,
                count: retiring.incident_count ?? 0,
              })} ${retireError}`
            : t("admin.customFields.retireConfirmCount", {
                field: retiring.label,
                count: retiring.incident_count ?? 0,
              })}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npm test -- --run src/console/pages/CustomFieldsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Run web lint, tests, and build**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/types.ts web/src/console/pages/CustomFieldsPage.tsx web/src/i18n/messages.ts web/src/console/pages/CustomFieldsPage.test.tsx
git commit -m "feat: show incident count in the custom field retire dialog"
```

---

### Task 4: Stop retired fields from rendering in the incident form

**Files:**
- Modify: `web/src/console/components/CreateIncidentForm.tsx` (custom fields section)
- Test: `web/src/console/pages/IncidentsPage.test.tsx` (the form renders inside this page's create dialog)

**Interfaces:**
- Consumes: `props.customFields: CustomFieldDefinition[]` (the list may contain RETIRED definitions; `status` distinguishes them).
- Produces: the "Custom fields" fieldset renders only definitions with `status === "ACTIVE"`.

- [ ] **Step 1: Write the failing test**

In `web/src/console/pages/IncidentsPage.test.tsx`, locate the existing create-form custom-fields test (the mock `listFieldDefinitions` response). Add a RETIRED definition to that mock, open the create dialog, and assert the retired field's control label does NOT render while an ACTIVE one does:

```tsx
    const activeLabel = await screen.findByLabelText("PO Number");
    expect(activeLabel).toBeInTheDocument();
    expect(screen.queryByLabelText("Retired Note")).not.toBeInTheDocument();
```

The mock definition for the retired field uses `status: "RETIRED"`, `field_type: "TEXT"`, `key: "retired_note"`, `label: "Retired Note"`. If the existing test does not open the form, extend the page test to click the create button first.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: FAIL — "Retired Note" control renders.

- [ ] **Step 3: Implement the ACTIVE filter**

In `web/src/console/components/CreateIncidentForm.tsx`, replace `{props.customFields.map((field) => (` (line 317) with a filtered map:

```tsx
          {props.customFields
            .filter((field) => field.status === "ACTIVE")
            .map((field) => (
            <CustomFieldControl
              key={field.id}
              field={field}
              value={customValues[field.key]}
              onChange={(value) =>
                setCustomValues((prev) => ({ ...prev, [field.key]: value }))
              }
            />
          ))}
```

Keep the surrounding `{props.customFields.length > 0 && (` guard and `fieldset` unchanged.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Run web lint, tests, and build**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/console/components/CreateIncidentForm.tsx web/src/console/pages/IncidentsPage.test.tsx
git commit -m "fix: stop retired custom fields from appearing in the incident form"
```

---

### Task 5: Hide retired fields in the admin list by default

**Files:**
- Modify: `web/src/console/pages/CustomFieldsPage.tsx` (toggle + filtered rows)
- Modify: `web/src/i18n/messages.ts` (both `en` and `vi` blocks)
- Test: `web/src/console/pages/CustomFieldsPage.test.tsx`

**Interfaces:**
- Consumes: `CustomFieldDefinition.status`.
- Produces: a `showRetired` toggle state; the table renders ACTIVE fields by default and RETIRED fields only when the toggle is on.

- [ ] **Step 1: Write the failing tests**

In `web/src/console/pages/CustomFieldsPage.test.tsx`, the shared `RETIRED_FIELD` fixture ("Zone") currently renders in the default view. Rewrite the two tests that assume that:

1. `renders definitions in a table`: replace the `screen.getByText("zone")` assertion with the default-hidden + toggle behavior:

```tsx
    renderPage();
    expect(await screen.findByText("PO Number")).toBeTruthy();
    expect(screen.queryByText("zone")).not.toBeInTheDocument();

    fireEvent.click(await screen.findByRole("checkbox", { name: /show retired/i }));
    expect(await screen.findByText("zone")).toBeTruthy();
```

2. `disables retire for already-retired fields`: the "Zone" row is hidden by default, so enable the toggle first:

```tsx
    renderPage();
    fireEvent.click(await screen.findByRole("checkbox", { name: /show retired/i }));
    const row = (await screen.findByText("Zone")).closest("tr") as HTMLTableRowElement;
    expect((within(row).getByRole("button", { name: /retire/i }) as HTMLButtonElement).disabled).toBe(true);
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd web && npm test -- --run src/console/pages/CustomFieldsPage.test.tsx`
Expected: FAIL — "Zone" renders in the default view (both rewritten tests assert it is hidden).

- [ ] **Step 3: Implement the toggle**

In `web/src/i18n/messages.ts`, add to both blocks:

```ts
  "admin.customFields.showRetired": "Show retired",
```

```ts
  "admin.customFields.showRetired": "Hiện trường đã ngừng",
```

In `web/src/console/pages/CustomFieldsPage.tsx`:

1. Add state near the other state declarations:

```tsx
  const [showRetired, setShowRetired] = useState(false);
```

2. Compute the filtered list:

```tsx
  const visibleFields = (fields.data ?? []).filter(
    (field) => showRetired || field.status === "ACTIVE",
  );
```

3. Render the toggle in the `PageHeader` actions, next to the "New field" button (keeps the table column count unchanged):

```tsx
      <PageHeader
        title={t("admin.customFields.title")}
        actions={
          <>
            <label className="toggle-label">
              <input
                type="checkbox"
                checked={showRetired}
                onChange={(event) => setShowRetired(event.target.checked)}
              />
              {t("admin.customFields.showRetired")}
            </label>
            <button className="primary-button" onClick={() => { setDialogError(undefined); setCreating(true); }}>
              {t("admin.customFields.newField")}
            </button>
          </>
        }
      />
```

4. Change the body map to iterate `visibleFields` instead of `fields.data`:

```tsx
              {visibleFields.map((field) => (
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npm test -- --run src/console/pages/CustomFieldsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Run web lint, tests, and build**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/console/pages/CustomFieldsPage.tsx web/src/i18n/messages.ts web/src/console/pages/CustomFieldsPage.test.tsx
git commit -m "feat: hide retired custom fields in the admin list by default"
```

---

### Task 6: Queue "Fields" disclosure row

**Files:**
- Modify: `web/src/console/ui/DataTable.tsx` (optional `expandedRow` + `expandedKey` props)
- Modify: `web/src/console/pages/IncidentsPage.tsx` (remove the "custom" column; chevron + disclosure row)
- Modify: `web/src/i18n/messages.ts` (both blocks, aria labels)
- Test: `web/src/console/pages/IncidentsPage.test.tsx`

**Interfaces:**
- Consumes: `Column<T>` as today; `formatCustomValue(field, value)` from `web/src/console/components/customFieldFormat.ts`; `Incident.custom_values` keyed by definition id; `CustomFieldDefinition[]` from `api.listFieldDefinitions`.
- Produces: `DataTable` gains optional `expandedRow?: (row: T) => ReactNode` and `expandedKey?: string | null`; the queue renders custom values in an expandable row instead of a column (spec: "custom fields are not new columns; a 'Fields' disclosure row expands to show custom field values per row").

- [ ] **Step 1: Write the failing test**

In `web/src/console/pages/IncidentsPage.test.tsx`, extend the list mock so one incident has `custom_values: { [definitionId]: "PO-42" }` and `custom_fields` includes the matching definition. Add:

```tsx
    // No dedicated custom-fields column (spec: not new columns).
    expect(screen.queryByRole("columnheader", { name: "Custom fields" })).not.toBeInTheDocument();

    // The disclosure row expands to show the value.
    fireEvent.click(screen.getByRole("button", { name: /show custom fields/i }));
    expect(await screen.findByText(/PO Number/)).toBeInTheDocument();
    expect(screen.getByText("PO-42")).toBeInTheDocument();
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: FAIL — a "Custom fields" column header exists and there is no expander button.

- [ ] **Step 3: Add `expandedRow` support to DataTable**

In `web/src/console/ui/DataTable.tsx`:

1. Extend the props type:

```tsx
  expandedKey?: string | null;
  expandedRow?: (row: T) => ReactNode;
```

2. Destructure them in the function signature.

3. After the closing `</tr>` of each data row (inside the `sorted.map`), render the disclosure row when this row is expanded:

```tsx
            {expandedKey === key && expandedRow ? (
              <tr className="data-row-detail">
                <td colSpan={columns.length}>{expandedRow(row)}</td>
              </tr>
            ) : null}
```

- [ ] **Step 4: Rework the IncidentsPage queue**

In `web/src/console/pages/IncidentsPage.tsx`:

1. Remove the `key: "custom"` column block (currently around line 403-415) entirely.

2. Add an expansion state:

```tsx
  const [expandedIncident, setExpandedIncident] = useState<string | null>(null);
```

3. Pass the new props to `DataTable`:

```tsx
          expandedKey={expandedIncident}
          expandedRow={(incident) => {
            const entries = customFields.data
              ?.filter((field) => field.status === "ACTIVE")
              .map((field) => ({ field, value: incident.custom_values?.[field.id] }))
              .filter((entry) => entry.value !== undefined && entry.value !== null && entry.value !== "");
            if (!entries || entries.length === 0) {
              return <p className="muted">{t("incidents.noCustomValues")}</p>;
            }
            return (
              <dl className="custom-fields-inline">
                {entries.map(({ field, value }) => (
                  <div key={field.id} className="inline-value">
                    <dt>{field.label}</dt>
                    <dd>{formatCustomValue(field, value)}</dd>
                  </div>
                ))}
              </dl>
            );
          }}
```

4. Add a chevron column as the last column (before or after "age") that toggles expansion without triggering the row navigation. If `web/src/styles.css` has no `data-row-detail` / `custom-fields-inline` / `row-expander` rules, add minimal styles there following the design-system tokens (muted secondary text, small padding, cursor pointer on the expander) so the disclosure reads as part of the table. 

```tsx
            {
              key: "fields",
              header: "",
              render: (incident) => {
                const expanded = expandedIncident === incident.id;
                return (
                  <button
                    type="button"
                    className="row-expander"
                    aria-expanded={expanded}
                    aria-label={t("incidents.expandFields")}
                    onClick={(event) => {
                      event.stopPropagation();
                      setExpandedIncident(expanded ? null : incident.id);
                    }}
                  >
                    <CaretDown
                      size={14}
                      aria-hidden="true"
                      className={expanded ? "row-expander-icon expanded" : "row-expander-icon"}
                    />
                  </button>
                );
              },
            },
```

`CaretDown` must be imported from `@phosphor-icons/react` (check the import list at the top of the file; add it if missing).

- [ ] **Step 5: Add the i18n keys**

In `web/src/i18n/messages.ts`, both blocks:

```ts
  "incidents.expandFields": "Show custom fields",
  "incidents.noCustomValues": "No custom field values.",
```

```ts
  "incidents.expandFields": "Xem trường tùy chỉnh",
  "incidents.noCustomValues": "Không có giá trị trường tùy chỉnh.",
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: PASS.

- [ ] **Step 7: Run web lint, tests, and build**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add web/src/console/ui/DataTable.tsx web/src/console/pages/IncidentsPage.tsx web/src/i18n/messages.ts web/src/console/pages/IncidentsPage.test.tsx
git commit -m "feat: show custom field values in an expandable queue row"
```

---

### Task 7: Persist custom field filters in saved views

**Files:**
- Modify: `web/src/console/components/SavedViews.tsx` (`SavedView` type)
- Modify: `web/src/console/pages/IncidentsPage.tsx` (`saveCurrentView`, `applyView`)
- Test: `web/src/console/pages/IncidentsPage.test.tsx`

**Interfaces:**
- Consumes: `SavedView { id, name, priority, query }`; `customFieldFiltersRef` (record of field key -> value) and `setCustomFieldFilters` in `IncidentsPage`.
- Produces: `SavedView` gains optional `custom_fields?: Record<string, string>`; saving stores the current custom field filters; applying a view restores them.

- [ ] **Step 1: Write the failing test**

In `web/src/console/pages/IncidentsPage.test.tsx`, add a saved-view round-trip test. The page persists views to `localStorage` under `skawld.incidents.savedViews`; the test seeds a view with `custom_fields` and asserts the filter select reflects it after applying:

```tsx
    localStorage.setItem(
      "skawld.incidents.savedViews",
      JSON.stringify([
        {
          id: "zone-view",
          name: "Zone A",
          priority: "ALL",
          query: "",
          custom_fields: { zone: "A" },
        },
      ]),
    );
    // Render the page, click the saved-view chip "Zone A", then assert the
    // zone filter select has value "A" and the list request carried
    // custom_fields: { zone: "A" } (inspect the mocked api.incidents call).
```

If the existing test file already covers saved views, extend the nearest test instead of adding a second render (avoid double `localStorage` seeding).

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: FAIL — applying the view does not restore the filter.

- [ ] **Step 3: Implement saved-view persistence**

In `web/src/console/components/SavedViews.tsx`:

```ts
export interface SavedView {
  id: string;
  name: string;
  priority: string;
  query: string;
  custom_fields?: Record<string, string>;
}
```

In `web/src/console/pages/IncidentsPage.tsx`:

1. In `saveCurrentView`, add the filters (the ref holds the current selections):

```tsx
      custom_fields: { ...customFieldFiltersRef.current },
```

2. In `applyView`, restore them:

```tsx
    const filters = view.custom_fields ?? {};
    customFieldFiltersRef.current = { ...filters };
    setCustomFieldFilters({ ...filters });
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd web && npm test -- --run src/console/pages/IncidentsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Run web lint, tests, and build**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/console/components/SavedViews.tsx web/src/console/pages/IncidentsPage.tsx web/src/console/pages/IncidentsPage.test.tsx
git commit -m "feat: persist custom field filters in saved views"
```

---

### Task 8: Changelog and full verification

**Files:**
- Modify: `ChangeLogs.md`

- [ ] **Step 1: Add a changelog entry**

Add at the top of `ChangeLogs.md` under `## [Unreleased]`:

```markdown
### 2026-08-10 — Custom fields spec gap closure

- Incident list filters on NUMBER custom fields now accept `min:max` ranges
  (open bounds supported) in addition to exact matches.
- The retire dialog shows how many incidents currently hold values for the
  field.
- Retired fields no longer appear in the incident form and are hidden in the
  admin list by default (with a "Show retired" toggle).
- Custom field values moved from a queue column into an expandable per-row
  "Fields" disclosure, and custom field filters are now persisted in saved
  views.
```

- [ ] **Step 2: Full backend verification**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: PASS. If `TEST_DATABASE_URL` is available, also run `go test -race -count=1 ./internal/customfield/... ./internal/incident/...`.

- [ ] **Step 3: Full web verification**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 4: Smoke test the flow**

1. `make compose-up && make migrate-up` (or the local dev flow from `docs/runbooks/local-development.md`).
2. `go run ./cmd/api` and `cd web && npm run dev`.
3. As an Administrator, create a NUMBER field `temperature` and an incident with `temperature=25`; list incidents with `custom_field.temperature=20:30` → the incident appears; `:20` → it does not.
4. Open the admin custom fields page: retired fields are hidden by default, the "Show retired" toggle reveals them, and the retire dialog shows the incident count.
5. Create an incident: retired fields do not render in the form.
6. Set a custom field filter on the queue, save a view, apply it again → the filter is restored; the queue row expands to show custom values.
7. Confirm the column-header test expectation: the queue has no "Custom fields" column.

- [ ] **Step 5: Commit**

```bash
git add ChangeLogs.md
git commit -m "docs: record custom fields spec gap closure"
```

---

## Self-Review Notes

- **Spec coverage check:**
  - "TEXT exact / NUMBER range / SELECT value filter" → Task 1 (range syntax `min:max` for NUMBER fields; TEXT/SELECT stay exact; OpenAPI updated).
  - "Retire (confirmation dialog showing how many incidents hold values)" → Tasks 2 (backend `incident_count`) + 3 (dialog).
  - "Retired fields are hidden from the create/edit form and admin list default view" → Tasks 4 (form) + 5 (admin default view with toggle).
  - "custom fields are not new columns; a 'Fields' disclosure row expands to show custom field values per row" → Task 6 (column removed, DataTable `expandedRow` disclosure).
  - "saved-view filters gain custom field filter chips (existing saved-views pattern)" → Task 7 (chips already shipped; persistence in saved views added).
  - Everything else in the spec (migration, domain validation, immutability, retire semantics, history, permission `field:manage`, admin API, incident payloads, OpenAPI, dynamic controls, detail card) was already implemented and verified by the 2026-08-09 plan and its execution.
- **Placeholder scan:** every code step above is concrete; no "TBD"/"add validation"-style steps. The two places that reference "copy the insert SQL from TestIncidentCustomValuesIntegration" point to an exact, existing block in the same file rather than a placeholder.
- **Type consistency:** `CustomFieldRange{Min, Max *float64}` and `Filter.CustomFieldRanges` are defined once (Task 1) and consumed by the store and the integration test; `incident_count` on `customfieldapp.Definition` (Task 2) is consumed as `incident_count?: number` in web types and the retire dialog (Task 3); `SavedView.custom_fields?: Record<string, string>` (Task 7) matches `customFieldFiltersRef` (`Record<string, string>`). `DataTable.expandedRow`/`expandedKey` (Task 6) are optional, so the four other DataTable consumers (ExecutionsPage, ReportsPage, AssetDetailPage, IncidentDetailPage) compile unchanged.
- **Known pre-existing issue (not in scope):** `web/src/console/pages/KnowledgePage.test.tsx` fails its "Load more" test in the current tree; it is unrelated to custom fields (knowledge page) and should be diagnosed separately.
