# Custom Fields Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let tenant administrators define custom fields on incidents (TEXT/NUMBER/DATE/SELECT/MULTI_SELECT), store validated values per incident, and surface them in the create form, detail page, queue, and filters.

**Architecture:** Config is data, not code. A per-tenant `field_definitions` table describes field shape; values live in the `incidents.custom_values` JSONB column keyed by field definition ID (API input is keyed by field *key*, resolved server-side). Values are validated in the Go application layer. Field definitions that have ever held data become immutable except label/description/sort_order and can only be retired, never deleted. A new `field:manage` permission (Administrator only) gates admin mutations; reading definitions rides on `incident:read`.

**Tech Stack:** Go (pgx/v5, chi), PostgreSQL 16 (JSONB, GIN), React + TypeScript + Vite, existing design system.

## Global Constraints

- Migration files are forward-only, embedded via `migrations/embed.go` (`//go:embed *.sql`), numbered `00015_...`.
- Key regex: `^[a-z][a-z0-9_]{1,63}$`; keys are immutable after creation.
- Field `field_type` and `config` are mutable ONLY while no incident holds a value (`HasValues == false`); afterwards locked. Label/description/sort_order always mutable.
- Retired fields are hidden from forms and default admin list, keep all values and history, remain readable.
- Value validation happens in the application layer, never trusting raw JSONB.
- New permission `field:manage` = `"field:manage"`; granted ONLY to `RoleAdministrator` in `internal/identity/domain/authorization.go` and mirrored in `web/src/console/rolePermissions.ts`.
- API custom-field values are keyed by field **key** on input, stored keyed by field definition **id**; list/detail responses include a `custom_fields` array of definitions so the UI can map ids back to keys.
- All new UI follows `docs/contributing/web-ui-design-system.md` tokens and i18n via `web/src/i18n/messages.ts` (both `en` and `vi` blocks).
- Do NOT commit unless the user asks; use existing commit message conventions when committing.

---

### Task 1: Migration `00015_custom_fields.sql`

**Files:**
- Create: `migrations/00015_custom_fields.sql`
- Modify: none (embed is automatic)

**Interfaces:**
- Produces: tables `field_definitions`, `custom_value_history`; column `incidents.custom_values jsonb NOT NULL DEFAULT '{}'`; indexes `field_definitions_org_idx`, `incidents_custom_values_gin`, `custom_value_history_incident_idx`.

- [ ] **Step 1: Write the migration**

```sql
CREATE TABLE field_definitions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    entity_type     text NOT NULL,
    key             text NOT NULL,
    label           text NOT NULL,
    description     text NOT NULL DEFAULT '',
    field_type      text NOT NULL,
    config          jsonb NOT NULL DEFAULT '{}'::jsonb,
    status          text NOT NULL DEFAULT 'ACTIVE',
    sort_order      integer NOT NULL DEFAULT 0,
    version         integer NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    retired_at      timestamptz,
    UNIQUE (organization_id, entity_type, key),
    CHECK (field_type IN ('TEXT', 'NUMBER', 'DATE', 'SELECT', 'MULTI_SELECT')),
    CHECK (status IN ('ACTIVE', 'RETIRED'))
);

ALTER TABLE incidents
    ADD COLUMN custom_values jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE custom_value_history (
    id                 bigserial PRIMARY KEY,
    incident_id        uuid NOT NULL REFERENCES incidents(id),
    field_definition_id uuid NOT NULL REFERENCES field_definitions(id),
    principal_id       uuid NOT NULL REFERENCES principals(id),
    value_before       jsonb,
    value_after        jsonb NOT NULL,
    changed_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX field_definitions_org_idx
    ON field_definitions (organization_id, entity_type, sort_order);
CREATE INDEX incidents_custom_values_gin
    ON incidents USING gin (custom_values);
CREATE INDEX custom_value_history_incident_idx
    ON custom_value_history (incident_id, changed_at DESC);
```

- [ ] **Step 2: Verify migration applies**

Run: `make migrate-up` against the dev database (see `docs/runbooks/local-development.md`).
Expected: applies cleanly; `\d incidents` shows `custom_values jsonb NOT NULL DEFAULT '{}'::jsonb`.

- [ ] **Step 3: Commit**

```bash
git add migrations/00015_custom_fields.sql
git commit -m "feat: add per-tenant custom field definitions and incident values"
```

---

### Task 2: Custom field domain — types and value validation

**Files:**
- Create: `internal/customfield/domain/definition.go`
- Create: `internal/customfield/domain/definition_test.go`

**Interfaces:**
- Produces (used by Tasks 3–7):
  - `type FieldType string` with consts `FieldTypeText`, `FieldTypeNumber`, `FieldTypeDate`, `FieldTypeSelect`, `FieldTypeMultiSelect`.
  - `type Status string` with consts `StatusActive`, `StatusRetired`.
  - `type Option struct { Label, Value string }`
  - `type Config struct { Required bool; MaxLength int; Regex string; Min, Max *float64; Options []Option }`
  - `type Definition struct { ID, OrganizationID, EntityType, Key, Label, Description string; FieldType FieldType; Config Config; Status Status; SortOrder int; Version int64; CreatedAt, UpdatedAt time.Time; RetiredAt *time.Time }`
  - `func New(value Definition) (Definition, error)`
  - `func (d Definition) Validate() error`
  - `func ValidateValue(d Definition, value any) error` — returns human-readable per-field errors.

- [ ] **Step 1: Write the failing tests**

`internal/customfield/domain/definition_test.go`:

```go
package domain

import (
	"strings"
	"testing"
)

func TestNewValidatesKey(t *testing.T) {
	for _, key := range []string{"po_number", "a", "shutdown_time"} {
		if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: key, Label: "L", FieldType: FieldTypeText}); err != nil {
			t.Errorf("key %q should be valid: %v", key, err)
		}
	}
	for _, key := range []string{"", "PO_Number", "1abc", "with space", strings.Repeat("x", 65), "has-dash"} {
		if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: key, Label: "L", FieldType: FieldTypeText}); err == nil {
			t.Errorf("key %q should be invalid", key)
		}
	}
}

func TestNewRequiresSelectOptions(t *testing.T) {
	_, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: "zone", Label: "Zone", FieldType: FieldTypeSelect, Config: Config{Options: nil}})
	if err == nil {
		t.Fatal("SELECT without options must fail")
	}
	if _, err := New(Definition{OrganizationID: "o1", EntityType: "incident", Key: "zone", Label: "Zone", FieldType: FieldTypeSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}}}}); err != nil {
		t.Fatalf("SELECT with options must pass: %v", err)
	}
}

func TestValidateValueText(t *testing.T) {
	d := Definition{FieldType: FieldTypeText, Config: Config{Required: true, MaxLength: 10}}
	if err := ValidateValue(d, nil); err == nil {
		t.Error("required nil must fail")
	}
	if err := ValidateValue(d, "toolongvalue"); err == nil {
		t.Error("over max length must fail")
	}
	if err := ValidateValue(d, "ok"); err != nil {
		t.Errorf("valid text must pass: %v", err)
	}
	if err := ValidateValue(d, 42); err == nil {
		t.Error("non-string must fail")
	}
}

func TestValidateValueNumber(t *testing.T) {
	min, max := 0.0, 100.0
	d := Definition{FieldType: FieldTypeNumber, Config: Config{Min: &min, Max: &max}}
	if err := ValidateValue(d, 50.0); err != nil {
		t.Errorf("in-range must pass: %v", err)
	}
	if err := ValidateValue(d, 101.0); err == nil {
		t.Error("above max must fail")
	}
	if err := ValidateValue(d, "fifty"); err == nil {
		t.Error("non-number must fail")
	}
}

func TestValidateValueDate(t *testing.T) {
	d := Definition{FieldType: FieldTypeDate}
	if err := ValidateValue(d, "2026-08-09"); err != nil {
		t.Errorf("valid date must pass: %v", err)
	}
	if err := ValidateValue(d, "not-a-date"); err == nil {
		t.Error("invalid date must fail")
	}
}

func TestValidateValueSelect(t *testing.T) {
	d := Definition{FieldType: FieldTypeSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}}}
	if err := ValidateValue(d, "a"); err != nil {
		t.Errorf("known option must pass: %v", err)
	}
	if err := ValidateValue(d, "zzz"); err == nil {
		t.Error("unknown option must fail")
	}
}

func TestValidateValueMultiSelect(t *testing.T) {
	d := Definition{FieldType: FieldTypeMultiSelect, Config: Config{Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}}}
	if err := ValidateValue(d, []any{"a", "b"}); err != nil {
		t.Errorf("known options must pass: %v", err)
	}
	if err := ValidateValue(d, []any{"a", "zzz"}); err == nil {
		t.Error("unknown option must fail")
	}
	if err := ValidateValue(d, []any{}); err == nil {
		t.Error("empty array must fail")
	}
}

func TestValidateValueRegex(t *testing.T) {
	d := Definition{FieldType: FieldTypeText, Config: Config{Regex: `^PO-\d+$`}}
	if err := ValidateValue(d, "PO-123"); err != nil {
		t.Errorf("matching value must pass: %v", err)
	}
	if err := ValidateValue(d, "nope"); err == nil {
		t.Error("non-matching value must fail")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/customfield/domain/...`
Expected: compile failure ("package customfield/domain is not in std" / undefined).

- [ ] **Step 3: Write the implementation**

`internal/customfield/domain/definition.go`:

```go
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FieldType string

const (
	FieldTypeText        FieldType = "TEXT"
	FieldTypeNumber      FieldType = "NUMBER"
	FieldTypeDate        FieldType = "DATE"
	FieldTypeSelect      FieldType = "SELECT"
	FieldTypeMultiSelect FieldType = "MULTI_SELECT"
)

type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusRetired Status = "RETIRED"
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Config struct {
	Required  bool     `json:"required,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
	Regex     string   `json:"regex,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Options   []Option `json:"options,omitempty"`
}

type Definition struct {
	ID             string
	OrganizationID string
	EntityType     string
	Key            string
	Label          string
	Description    string
	FieldType      FieldType
	Config         Config
	Status         Status
	SortOrder      int
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RetiredAt      *time.Time
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

func New(value Definition) (Definition, error) {
	value.Key = strings.TrimSpace(value.Key)
	value.Label = strings.TrimSpace(value.Label)
	value.EntityType = strings.TrimSpace(value.EntityType)
	if value.OrganizationID == "" || value.EntityType == "" {
		return Definition{}, errors.New("field definition scope is required")
	}
	if !keyPattern.MatchString(value.Key) {
		return Definition{}, errors.New("field key must match ^[a-z][a-z0-9_]{1,63}$")
	}
	if value.Label == "" {
		return Definition{}, errors.New("field label is required")
	}
	switch value.FieldType {
	case FieldTypeText, FieldTypeNumber, FieldTypeDate, FieldTypeSelect, FieldTypeMultiSelect:
	default:
		return Definition{}, fmt.Errorf("unsupported field type %q", value.FieldType)
	}
	if (value.FieldType == FieldTypeSelect || value.FieldType == FieldTypeMultiSelect) && len(value.Config.Options) == 0 {
		return Definition{}, errors.New("select fields require at least one option")
	}
	if value.FieldType == FieldTypeSelect || value.FieldType == FieldTypeMultiSelect {
		seen := make(map[string]struct{}, len(value.Config.Options))
		for _, option := range value.Config.Options {
			option.Value = strings.TrimSpace(option.Value)
			if option.Value == "" {
				return Definition{}, errors.New("select option values are required")
			}
			if _, dup := seen[option.Value]; dup {
				return Definition{}, fmt.Errorf("duplicate select option value %q", option.Value)
			}
			seen[option.Value] = struct{}{}
		}
	}
	if value.Status == "" {
		value.Status = StatusActive
	}
	if value.Status != StatusActive && value.Status != StatusRetired {
		return Definition{}, errors.New("unsupported field status")
	}
	value.Version = 1
	return value, nil
}

func (d Definition) Validate() error {
	_, err := New(d)
	return err
}

func ValidateValue(d Definition, value any) error {
	if value == nil {
		if d.Config.Required {
			return fmt.Errorf("%s is required", d.Label)
		}
		return nil
	}
	switch d.FieldType {
	case FieldTypeText:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be text", d.Label)
		}
		text = strings.TrimSpace(text)
		if d.Config.Required && text == "" {
			return fmt.Errorf("%s is required", d.Label)
		}
		if d.Config.MaxLength > 0 && len(text) > d.Config.MaxLength {
			return fmt.Errorf("%s must be at most %d characters", d.Label, d.Config.MaxLength)
		}
		if d.Config.Regex != "" {
			matched, err := regexp.MatchString(d.Config.Regex, text)
			if err != nil {
				return fmt.Errorf("%s has an invalid pattern", d.Label)
			}
			if !matched {
				return fmt.Errorf("%s does not match the required pattern", d.Label)
			}
		}
	case FieldTypeNumber:
		number, ok := toFloat(value)
		if !ok {
			return fmt.Errorf("%s must be a number", d.Label)
		}
		if d.Config.Min != nil && number < *d.Config.Min {
			return fmt.Errorf("%s must be at least %v", d.Label, *d.Config.Min)
		}
		if d.Config.Max != nil && number > *d.Config.Max {
			return fmt.Errorf("%s must be at most %v", d.Label, *d.Config.Max)
		}
	case FieldTypeDate:
		date, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a date", d.Label)
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return fmt.Errorf("%s must be a date in YYYY-MM-DD format", d.Label)
		}
	case FieldTypeSelect:
		selected, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a single option", d.Label)
		}
		if !hasOption(d, selected) {
			return fmt.Errorf("%s has an unknown option", d.Label)
		}
	case FieldTypeMultiSelect:
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be a list of options", d.Label)
		}
		if len(items) == 0 {
			return fmt.Errorf("%s must select at least one option", d.Label)
		}
		for _, item := range items {
			text, ok := item.(string)
			if !ok || !hasOption(d, text) {
				return fmt.Errorf("%s has an unknown option", d.Label)
			}
		}
	default:
		return fmt.Errorf("unsupported field type %q", d.FieldType)
	}
	return nil
}

func hasOption(d Definition, value string) bool {
	for _, option := range d.Config.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func toFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		return number.Float64()
	default:
		return 0, false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/customfield/domain/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/customfield/domain/
git commit -m "feat: add custom field domain types and value validation"
```

---

### Task 3: Custom field application service

**Files:**
- Create: `internal/customfield/application/service.go`
- Create: `internal/customfield/application/service_test.go`

**Interfaces:**
- Consumes: `customfield/domain` (Task 2).
- Produces (used by Tasks 4, 6, 7):
  - `ErrForbidden, ErrNotFound, ErrInvalid, ErrConflict, ErrValidation error`
  - `type CreateDefinition struct { EntityType, Key, Label, Description string; FieldType string; Config domain.Config; SortOrder int }`
  - `type UpdateDefinition struct { Label, Description string; FieldType string; Config domain.Config; SortOrder int; ExpectedVersion int64 }`
  - `type Definition` (app view with JSON tags mirroring domain.Definition; `field_type`, `config`, `status`, `sort_order`, `version`, `created_at`, `updated_at`, `retired_at`).
  - `type HistoryEntry struct { IncidentID string; PrincipalID string; ValueBefore, ValueAfter any; ChangedAt time.Time }`
  - `type Store interface { ListByEntity(ctx, orgID, entityType string) ([]domain.Definition, error); Get(ctx, orgID, id string) (domain.Definition, error); Create(ctx, orgID string, d domain.Definition, now time.Time) (domain.Definition, error); Update(ctx, orgID, id string, d domain.Definition, now time.Time) (domain.Definition, error); Retire(ctx, orgID, id string, now time.Time) (domain.Definition, error); History(ctx, orgID, id string) ([]HistoryEntry, error); HasValues(ctx, orgID, id string) (bool, error) }`
  - `type Service struct { Store Store; Now func() time.Time }`
  - Service methods (all take `identitydomain.Principal`):
    - `ListByEntity(ctx, principal, entityType string) ([]Definition, error)` — requires `incident:read`.
    - `Create(ctx, principal, command CreateDefinition) (Definition, error)` — requires `field:manage`.
    - `Get(ctx, principal, id string) (Definition, error)` — requires `incident:read`.
    - `Update(ctx, principal, id string, command UpdateDefinition) (Definition, error)` — requires `field:manage`; blocks type/config change when `HasValues`.
    - `Retire(ctx, principal, id string) (Definition, error)` — requires `field:manage`; idempotent-ish (already retired → `ErrConflict`).
    - `History(ctx, principal, id string) ([]HistoryEntry, error)` — requires `field:manage`.
    - `ResolveAndValidate(ctx, organizationID, entityType string, values map[string]any) (map[string]any, error)` — keys → definition IDs, validated; unknown key or invalid value → `ErrValidation` with message.
    - `Definitions(ctx, organizationID, entityType string) ([]Definition, error)` — no permission check (internal helper for incident responses).
    - `ResolveKeys(ctx, organizationID, entityType string, keys []string) (map[string]Definition, error)` — key → definition, unknown key → `ErrValidation`.

- [ ] **Step 1: Write the failing tests**

`internal/customfield/application/service_test.go` (uses a fake Store):

```go
package application

import (
	"context"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
)

type fakeStore struct {
	definitions map[string]domain.Definition
	byKey       map[string]domain.Definition
	hasValues   bool
	history     []HistoryEntry
}

func (f *fakeStore) ListByEntity(_ context.Context, _, entityType string) ([]domain.Definition, error) {
	var out []domain.Definition
	for _, d := range f.definitions {
		if d.EntityType == entityType {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeStore) Get(_ context.Context, _, id string) (domain.Definition, error) {
	d, ok := f.definitions[id]
	if !ok {
		return domain.Definition{}, ErrNotFound
	}
	return d, nil
}
func (f *fakeStore) Create(_ context.Context, _ string, d domain.Definition, now time.Time) (domain.Definition, error) {
	d.ID = "def-1"
	d.CreatedAt = now
	d.UpdatedAt = now
	f.definitions[d.ID] = d
	f.byKey[d.Key] = d
	return d, nil
}
func (f *fakeStore) Update(_ context.Context, _, id string, d domain.Definition, _ time.Time) (domain.Definition, error) {
	f.definitions[id] = d
	return d, nil
}
func (f *fakeStore) Retire(_ context.Context, _, id string, now time.Time) (domain.Definition, error) {
	d := f.definitions[id]
	d.Status = domain.StatusRetired
	d.RetiredAt = &now
	f.definitions[id] = d
	return d, nil
}
func (f *fakeStore) History(_ context.Context, _, _ string) ([]HistoryEntry, error) { return f.history, nil }
func (f *fakeStore) HasValues(_ context.Context, _, _ string) (bool, error)          { return f.hasValues, nil }

func admin() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
			"field:manage":                        {},
		},
	}
}

func reader() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionIncidentRead: {}},
	}
}

func newService(store *fakeStore) Service {
	return Service{Store: store, Now: func() time.Time { return time.Unix(0, 0).UTC() }}
}

func TestCreateRequiresManagePermission(t *testing.T) {
	s := newService(&fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}})
	_, err := s.Create(context.Background(), reader(), CreateDefinition{EntityType: "incident", Key: "po", Label: "PO", FieldType: "TEXT"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateAndResolveAndValidate(t *testing.T) {
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	s := newService(store)
	created, err := s.Create(context.Background(), admin(), CreateDefinition{EntityType: "incident", Key: "po_number", Label: "PO Number", FieldType: "TEXT", Config: domain.Config{Required: true}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	normalized, err := s.ResolveAndValidate(context.Background(), "o1", "incident", map[string]any{"po_number": "PO-42"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if normalized[created.ID] != "PO-42" {
		t.Fatalf("expected normalized keyed by definition id, got %#v", normalized)
	}
	if _, err := s.ResolveAndValidate(context.Background(), "o1", "incident", map[string]any{"nope": "x"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown key must be ErrValidation, got %v", err)
	}
}

func TestUpdateLocksConfigAfterUse(t *testing.T) {
	store := &fakeStore{
		definitions: map[string]domain.Definition{},
		byKey:       map[string]domain.Definition{},
		hasValues:   true,
	}
	d := domain.Definition{ID: "def-1", OrganizationID: "o1", EntityType: "incident", Key: "po", Label: "PO", FieldType: domain.FieldTypeText, Status: domain.StatusActive, Version: 1}
	store.definitions["def-1"] = d
	s := newService(store)
	_, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{FieldType: "NUMBER", Config: domain.Config{}, SortOrder: 1, ExpectedVersion: 1})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("changing type on a used field must be ErrConflict, got %v", err)
	}
	updated, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{Label: "Purchase Order", SortOrder: 5, ExpectedVersion: 1})
	if err != nil {
		t.Fatalf("label change must pass: %v", err)
	}
	if updated.Label != "Purchase Order" {
		t.Fatalf("label not updated: %+v", updated)
	}
}

func TestRetireConflictWhenAlreadyRetired(t *testing.T) {
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	now := time.Unix(0, 0).UTC()
	store.definitions["def-1"] = domain.Definition{ID: "def-1", Status: domain.StatusRetired, RetiredAt: &now}
	s := newService(store)
	if _, err := s.Retire(context.Background(), admin(), "def-1"); !errors.Is(err, ErrConflict) {
		t.Fatalf("double retire must be ErrConflict, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/customfield/application/...`
Expected: compile failure (package not found).

- [ ] **Step 3: Write the implementation**

`internal/customfield/application/service.go`:

```go
package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
)

var (
	ErrForbidden   = errors.New("field operation forbidden")
	ErrNotFound    = errors.New("field definition not found")
	ErrInvalid     = errors.New("invalid field definition")
	ErrConflict    = errors.New("field definition conflict")
	ErrValidation  = errors.New("custom field value validation failed")
)

type CreateDefinition struct {
	EntityType  string        `json:"entity_type"`
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Description string        `json:"description,omitempty"`
	FieldType   string        `json:"field_type"`
	Config      domain.Config `json:"config"`
	SortOrder   int           `json:"sort_order"`
}

type UpdateDefinition struct {
	Label           string        `json:"label"`
	Description     string        `json:"description,omitempty"`
	FieldType       string        `json:"field_type"`
	Config          domain.Config `json:"config"`
	SortOrder       int           `json:"sort_order"`
	ExpectedVersion int64         `json:"expected_version"`
}

type Definition struct {
	ID             string        `json:"id"`
	EntityType     string        `json:"entity_type"`
	Key            string        `json:"key"`
	Label          string        `json:"label"`
	Description    string        `json:"description,omitempty"`
	FieldType      string        `json:"field_type"`
	Config         domain.Config `json:"config"`
	Status         string        `json:"status"`
	SortOrder      int           `json:"sort_order"`
	Version        int64         `json:"version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	RetiredAt      *time.Time    `json:"retired_at,omitempty"`
}

type HistoryEntry struct {
	IncidentID   string    `json:"incident_id"`
	PrincipalID  string    `json:"principal_id"`
	ValueBefore  any       `json:"value_before,omitempty"`
	ValueAfter   any       `json:"value_after"`
	ChangedAt    time.Time `json:"changed_at"`
}

type Store interface {
	ListByEntity(context.Context, string, string) ([]domain.Definition, error)
	Get(context.Context, string, string) (domain.Definition, error)
	Create(context.Context, string, domain.Definition, time.Time) (domain.Definition, error)
	Update(context.Context, string, string, domain.Definition, time.Time) (domain.Definition, error)
	Retire(context.Context, string, string, time.Time) (domain.Definition, error)
	History(context.Context, string, string) ([]HistoryEntry, error)
	HasValues(context.Context, string, string) (bool, error)
}

type Service struct {
	Store Store
	Now   func() time.Time
}

func (s Service) ListByEntity(ctx context.Context, principal identitydomain.Principal, entityType string) ([]Definition, error) {
	if !principal.Has(identitydomain.PermissionIncidentRead) {
		return nil, ErrForbidden
	}
	items, err := s.Store.ListByEntity(ctx, principal.OrganizationID, entityType)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
	out := make([]Definition, 0, len(items))
	for _, item := range items {
		out = append(out, toView(item))
	}
	return out, nil
}

func (s Service) Create(ctx context.Context, principal identitydomain.Principal, command CreateDefinition) (Definition, error) {
	if !principal.Has(identitydomain.PermissionFieldManage) {
		return Definition{}, ErrForbidden
	}
	value, err := domain.New(domain.Definition{
		OrganizationID: principal.OrganizationID,
		EntityType:     command.EntityType,
		Key:            command.Key,
		Label:          command.Label,
		Description:    command.Description,
		FieldType:      domain.FieldType(command.FieldType),
		Config:         command.Config,
		SortOrder:      command.SortOrder,
	})
	if err != nil {
		return Definition{}, errors.Join(ErrInvalid, err)
	}
	created, err := s.Store.Create(ctx, principal.OrganizationID, value, s.now())
	if err != nil {
		return Definition{}, err
	}
	return toView(created), nil
}

func (s Service) Get(ctx context.Context, principal identitydomain.Principal, id string) (Definition, error) {
	if !principal.Has(identitydomain.PermissionIncidentRead) {
		return Definition{}, ErrForbidden
	}
	value, err := s.Store.Get(ctx, principal.OrganizationID, id)
	if err != nil {
		return Definition{}, err
	}
	return toView(value), nil
}

func (s Service) Update(ctx context.Context, principal identitydomain.Principal, id string, command UpdateDefinition) (Definition, error) {
	if !principal.Has(identitydomain.PermissionFieldManage) {
		return Definition{}, ErrForbidden
	}
	current, err := s.Store.Get(ctx, principal.OrganizationID, id)
	if err != nil {
		return Definition{}, err
	}
	if current.Version != command.ExpectedVersion {
		return Definition{}, ErrConflict
	}
	used, err := s.Store.HasValues(ctx, principal.OrganizationID, id)
	if err != nil {
		return Definition{}, err
	}
	fieldType := current.FieldType
	config := current.Config
	if string(fieldType) != command.FieldType || !configsEqual(config, command.Config) {
		if used {
			return Definition{}, ErrConflict
		}
		fieldType = domain.FieldType(command.FieldType)
		config = command.Config
	}
	value, err := domain.New(domain.Definition{
		ID: current.ID, OrganizationID: current.OrganizationID,
		EntityType: current.EntityType, Key: current.Key,
		Label: command.Label, Description: command.Description,
		FieldType: fieldType, Config: config, Status: current.Status,
		SortOrder: command.SortOrder, Version: current.Version + 1,
		CreatedAt: current.CreatedAt,
	})
	if err != nil {
		return Definition{}, errors.Join(ErrInvalid, err)
	}
	updated, err := s.Store.Update(ctx, principal.OrganizationID, id, value, s.now())
	if err != nil {
		return Definition{}, err
	}
	return toView(updated), nil
}

func (s Service) Retire(ctx context.Context, principal identitydomain.Principal, id string) (Definition, error) {
	if !principal.Has(identitydomain.PermissionFieldManage) {
		return Definition{}, ErrForbidden
	}
	current, err := s.Store.Get(ctx, principal.OrganizationID, id)
	if err != nil {
		return Definition{}, err
	}
	if current.Status == domain.StatusRetired {
		return Definition{}, ErrConflict
	}
	retired, err := s.Store.Retire(ctx, principal.OrganizationID, id, s.now())
	if err != nil {
		return Definition{}, err
	}
	return toView(retired), nil
}

func (s Service) History(ctx context.Context, principal identitydomain.Principal, id string) ([]HistoryEntry, error) {
	if !principal.Has(identitydomain.PermissionFieldManage) {
		return nil, ErrForbidden
	}
	if _, err := s.Store.Get(ctx, principal.OrganizationID, id); err != nil {
		return nil, err
	}
	return s.Store.History(ctx, principal.OrganizationID, id)
}

func (s Service) Definitions(ctx context.Context, organizationID, entityType string) ([]Definition, error) {
	items, err := s.Store.ListByEntity(ctx, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
	out := make([]Definition, 0, len(items))
	for _, item := range items {
		out = append(out, toView(item))
	}
	return out, nil
}

// ResolveAndValidate maps values keyed by field key to values keyed by
// definition id, validating each against its definition. Unknown keys or
// invalid values return ErrValidation.
func (s Service) ResolveAndValidate(ctx context.Context, organizationID, entityType string, values map[string]any) (map[string]any, error) {
	definitions, err := s.Store.ListByEntity(ctx, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]domain.Definition, len(definitions))
	for _, d := range definitions {
		if d.Status == domain.StatusActive {
			byKey[d.Key] = d
		}
	}
	normalized := make(map[string]any, len(values))
	for key, value := range values {
		d, ok := byKey[key]
		if !ok {
			return nil, errors.Join(ErrValidation, fmt.Errorf("unknown custom field %q", key))
		}
		if err := domain.ValidateValue(d, value); err != nil {
			return nil, errors.Join(ErrValidation, err)
		}
		normalized[d.ID] = value
	}
	return normalized, nil
}

// ResolveKeys maps field keys to definitions for list filtering.
func (s Service) ResolveKeys(ctx context.Context, organizationID, entityType string, keys []string) (map[string]Definition, error) {
	definitions, err := s.Store.ListByEntity(ctx, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Definition, len(keys))
	for _, d := range definitions {
		byKey[d.Key] = toView(d)
	}
	out := make(map[string]Definition, len(keys))
	for _, key := range keys {
		d, ok := byKey[key]
		if !ok {
			return nil, errors.Join(ErrValidation, fmt.Errorf("unknown custom field %q", key))
		}
		out[key] = d
	}
	return out, nil
}

func (s Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func toView(value domain.Definition) Definition {
	return Definition{
		ID: value.ID, EntityType: value.EntityType, Key: value.Key,
		Label: value.Label, Description: value.Description,
		FieldType: string(value.FieldType), Config: value.Config,
		Status: string(value.Status), SortOrder: value.SortOrder,
		Version: value.Version, CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt, RetiredAt: value.RetiredAt,
	}
}

func configsEqual(a, b domain.Config) bool {
	return a.Required == b.Required &&
		a.MaxLength == b.MaxLength &&
		a.Regex == b.Regex &&
		floatsEqual(a.Min, b.Min) &&
		floatsEqual(a.Max, b.Max) &&
		optionsEqual(a.Options, b.Options)
}

func floatsEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func optionsEqual(a, b []domain.Option) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validKey(key string) bool {
	return strings.TrimSpace(key) != ""
}
```

Remove the unused `validKey` helper at the end if the linter complains (`go vet`/`staticcheck` in CI). Keep imports tidy: drop `"strings"` and `validKey` if unused.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/customfield/application/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/customfield/application/
git commit -m "feat: add custom field application service with validation"
```

---

### Task 4: Custom field postgres adapter

**Files:**
- Create: `internal/customfield/adapter/postgres/store.go`
- Create: `internal/customfield/adapter/postgres/store_integration_test.go`

**Interfaces:**
- Consumes: `customfield/application` (Task 3), `migrations/00015` (Task 1).
- Produces: `type Store struct { Pool *pgxpool.Pool }` implementing `application.Store`.

- [ ] **Step 1: Write the failing integration test**

```go
package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	ctx := context.Background()
	var orgID string
	err = pool.QueryRow(ctx, `INSERT INTO organizations (name) VALUES ($1) RETURNING id::text`, "test-org-"+time.Now().Format("150405.000")).Scan(&orgID)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	return &Store{Pool: pool}, orgID
}

func TestStoreCRUDAndImmutability(t *testing.T) {
	store, orgID := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	created, err := store.Create(ctx, orgID, domain.Definition{
		OrganizationID: orgID, EntityType: "incident", Key: "po_number",
		Label: "PO Number", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 1,
	}, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated id")
	}

	listed, err := store.ListByEntity(ctx, orgID, "incident")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v items=%d", err, len(listed))
	}
	if listed[0].Key != "po_number" {
		t.Fatalf("unexpected definition: %+v", listed[0])
	}

	used, err := store.HasValues(ctx, orgID, created.ID)
	if err != nil {
		t.Fatalf("hasValues: %v", err)
	}
	if used {
		t.Fatal("expected no values yet")
	}

	updated, err := store.Update(ctx, orgID, created.ID, domain.Definition{
		ID: created.ID, OrganizationID: orgID, EntityType: "incident",
		Key: "po_number", Label: "Purchase Order Number", FieldType: domain.FieldTypeText,
		Config: domain.Config{Required: true}, SortOrder: 2, Version: 2, CreatedAt: created.CreatedAt,
	}, now)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Version != 2 || updated.Label != "Purchase Order Number" {
		t.Fatalf("unexpected updated definition: %+v", updated)
	}

	retired, err := store.Retire(ctx, orgID, created.ID, now)
	if err != nil {
		t.Fatalf("retire: %v", err)
	}
	if retired.Status != domain.StatusRetired || retired.RetiredAt == nil {
		t.Fatalf("unexpected retired definition: %+v", retired)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `TEST_DATABASE_URL=postgres://... go test ./internal/customfield/adapter/postgres/...`
Expected: compile failure (package not found).

- [ ] **Step 3: Write the implementation**

`internal/customfield/adapter/postgres/store.go`:

```go
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

const selectDefinition = `
	SELECT id::text, entity_type, key, label, coalesce(description, ''),
	       field_type, config, status, sort_order, version,
	       created_at, updated_at, retired_at
	FROM field_definitions
`

func scanDefinition(row scanner) (domain.Definition, error) {
	var value domain.Definition
	var rawConfig []byte
	err := row.Scan(
		&value.ID, &value.EntityType, &value.Key, &value.Label, &value.Description,
		&value.FieldType, &rawConfig, &value.Status, &value.SortOrder, &value.Version,
		&value.CreatedAt, &value.UpdatedAt, &value.RetiredAt,
	)
	if err != nil {
		return domain.Definition{}, err
	}
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &value.Config); err != nil {
			return domain.Definition{}, err
		}
	}
	return value, nil
}

type scanner interface {
	Scan(...any) error
}

func (s Store) ListByEntity(ctx context.Context, organizationID, entityType string) ([]domain.Definition, error) {
	rows, err := s.Pool.Query(ctx, selectDefinition+`
		WHERE organization_id = $1::uuid AND entity_type = $2
		ORDER BY sort_order, key
	`, organizationID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Definition
	for rows.Next() {
		value, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (s Store) Get(ctx context.Context, organizationID, id string) (domain.Definition, error) {
	value, err := scanDefinition(s.Pool.QueryRow(ctx, selectDefinition+`
		WHERE id = $1::uuid AND organization_id = $2::uuid
	`, id, organizationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Definition{}, application.ErrNotFound
	}
	return value, err
}

func (s Store) Create(ctx context.Context, organizationID string, d domain.Definition, now time.Time) (domain.Definition, error) {
	rawConfig, err := json.Marshal(d.Config)
	if err != nil {
		return domain.Definition{}, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO field_definitions (
			organization_id, entity_type, key, label, description, field_type,
			config, status, sort_order, version, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $11)
		RETURNING id::text, created_at, updated_at
	`, organizationID, d.EntityType, d.Key, d.Label, d.Description,
		string(d.FieldType), rawConfig, string(d.Status), d.SortOrder, d.Version, now)
	if err := row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return domain.Definition{}, err
	}
	return d, nil
}

func (s Store) Update(ctx context.Context, organizationID, id string, d domain.Definition, now time.Time) (domain.Definition, error) {
	rawConfig, err := json.Marshal(d.Config)
	if err != nil {
		return domain.Definition{}, err
	}
	tag, err := s.Pool.Exec(ctx, `
		UPDATE field_definitions
		SET label = $3, description = $4, field_type = $5, config = $6::jsonb,
		    sort_order = $7, version = $8, updated_at = $9
		WHERE id = $1::uuid AND organization_id = $2::uuid AND version = $10
	`, id, organizationID, d.Label, d.Description, string(d.FieldType),
		rawConfig, d.SortOrder, d.Version, now, d.Version-1)
	if err != nil {
		return domain.Definition{}, err
	}
	if tag.RowsAffected() != 1 {
		return domain.Definition{}, application.ErrConflict
	}
	return s.Get(ctx, organizationID, id)
}

func (s Store) Retire(ctx context.Context, organizationID, id string, now time.Time) (domain.Definition, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE field_definitions
		SET status = 'RETIRED', retired_at = $3, version = version + 1, updated_at = $3
		WHERE id = $1::uuid AND organization_id = $2::uuid AND status = 'ACTIVE'
	`, id, organizationID, now)
	if err != nil {
		return domain.Definition{}, err
	}
	if tag.RowsAffected() != 1 {
		return domain.Definition{}, application.ErrConflict
	}
	return s.Get(ctx, organizationID, id)
}

func (s Store) History(ctx context.Context, organizationID, id string) ([]application.HistoryEntry, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT h.incident_id::text, h.principal_id::text,
		       h.value_before, h.value_after, h.changed_at
		FROM custom_value_history h
		JOIN field_definitions f ON f.id = h.field_definition_id
		WHERE f.id = $1::uuid AND f.organization_id = $2::uuid
		ORDER BY h.changed_at DESC
		LIMIT 100
	`, id, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.HistoryEntry
	for rows.Next() {
		var entry application.HistoryEntry
		var before, after []byte
		if err := rows.Scan(&entry.IncidentID, &entry.PrincipalID, &before, &after, &entry.ChangedAt); err != nil {
			return nil, err
		}
		if len(before) > 0 {
			if err := json.Unmarshal(before, &entry.ValueBefore); err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(after, &entry.ValueAfter); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (s Store) HasValues(ctx context.Context, organizationID, id string) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM incidents
			WHERE organization_id = $1::uuid AND custom_values ? $2::text
		)
	`, organizationID, id).Scan(&exists)
	return exists, err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `TEST_DATABASE_URL=... go test ./internal/customfield/adapter/postgres/...`
Expected: PASS. Also run `go vet ./internal/customfield/...`.

- [ ] **Step 5: Commit**

```bash
git add internal/customfield/adapter/postgres/
git commit -m "feat: add custom field definition postgres store"
```

---

### Task 5: `field:manage` permission wiring (backend + web mirror)

**Files:**
- Modify: `internal/identity/domain/authorization.go` (permission const ~line 48, Administrator list ~line 127)
- Modify: `web/src/console/permissions.ts` (PermissionKey union ~line 24)
- Modify: `web/src/console/rolePermissions.ts` (Administrator mirror)
- Modify: `web/src/console/permissions.test.ts` (if it enumerates permissions)

**Interfaces:**
- Produces: `identitydomain.PermissionFieldManage Permission = "field:manage"`; Administrator-only.

- [ ] **Step 1: Write the failing tests (backend)**

`internal/identity/domain/authorization_test.go` — add:

```go
func TestFieldManageIsAdministratorOnly(t *testing.T) {
	admin := PermissionsForRole(RoleAdministrator)
	if !contains(admin, PermissionFieldManage) {
		t.Fatal("Administrator must hold field:manage")
	}
	for _, role := range []Role{RoleMaintenanceSupervisor, RoleSeniorTechnician, RoleTechnician, RoleManager} {
		if contains(PermissionsForRole(role), PermissionFieldManage) {
			t.Fatalf("role %s must not hold field:manage", role)
		}
	}
}

func contains(list []Permission, target Permission) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/identity/domain/... -run TestFieldManageIsAdministratorOnly`
Expected: FAIL (constant undefined).

- [ ] **Step 3: Implement**

In `internal/identity/domain/authorization.go`:

Add to the permission const block (after `PermissionExternalImport`):

```go
	PermissionFieldManage          Permission = "field:manage"
```

Add to the `RoleAdministrator` case in `PermissionsForRole` (after `PermissionExternalImport,`):

```go
			PermissionFieldManage,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/identity/domain/...`
Expected: PASS.

- [ ] **Step 5: Mirror in web**

In `web/src/console/permissions.ts`, add `| "field:manage"` to the `PermissionKey` union (alphabetical: after `"execution:write"`).

In `web/src/console/rolePermissions.ts`, add `"field:manage"` to the `Administrator` entry only (find the `ROLE_PERMISSIONS` object and its `Administrator: [...` array).

- [ ] **Step 6: Update web permission tests**

Run: `cd web && npm test -- permissions`
Inspect `web/src/console/permissions.test.ts` and `rolePermissions` tests; if any test asserts the full permission list for Administrator, add `"field:manage"` to the expected list. Fix the test first (RED), then the source (GREEN).

- [ ] **Step 7: Run web tests + lint**

Run: `cd web && npm run lint && npm test`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/identity/domain/authorization.go internal/identity/domain/authorization_test.go web/src/console/permissions.ts web/src/console/rolePermissions.ts web/src/console/permissions.test.ts
git commit -m "feat: add field:manage permission for tenant admins"
```

---

### Task 6: Admin field-definition HTTP routes

**Files:**
- Create: `internal/platform/httpserver/customfields.go`
- Create: `internal/platform/httpserver/customfields_test.go`
- Modify: `internal/platform/httpserver/router.go` (Dependencies + mount ~line 112)

**Interfaces:**
- Consumes: `customfield/application` (Task 3), `field:manage` (Task 5).
- Produces: routes under `/api/v1/admin/field-definitions`:
  - `GET /admin/field-definitions?entity_type=incident` — `incident:read`
  - `POST /admin/field-definitions` — `field:manage`
  - `GET /admin/field-definitions/{id}` — `incident:read`
  - `PATCH /admin/field-definitions/{id}` — `field:manage`
  - `POST /admin/field-definitions/{id}/retire` — `field:manage`
  - `GET /admin/field-definitions/{id}/history` — `field:manage`
  - Error mapping: `ErrValidation` → 422, `ErrConflict` → 409, `ErrForbidden` → 403, `ErrNotFound` → 404, `ErrInvalid` → 400.

- [ ] **Step 1: Write the failing handler test**

`internal/platform/httpserver/customfields_test.go`:

```go
package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/go-chi/chi/v5"
)

type stubFieldStore struct {
	definitions map[string]application.Definition
}

func (s *stubFieldStore) ListByEntity(_ context.Context, _, _ string) ([]domain.Definition, error) {
	var out []domain.Definition
	for _, d := range s.definitions {
		out = append(out, domain.Definition{ID: d.ID, OrganizationID: "o1", EntityType: d.EntityType, Key: d.Key, Label: d.Label, FieldType: domain.FieldType(d.FieldType), Status: domain.StatusActive, Version: d.Version})
	}
	return out, nil
}
func (s *stubFieldStore) Get(_ context.Context, _, id string) (domain.Definition, error) {
	d, ok := s.definitions[id]
	if !ok {
		return domain.Definition{}, application.ErrNotFound
	}
	return domain.Definition{ID: d.ID, OrganizationID: "o1", EntityType: d.EntityType, Key: d.Key, Label: d.Label, FieldType: domain.FieldType(d.FieldType), Status: domain.StatusActive, Version: d.Version}, nil
}
func (s *stubFieldStore) Create(_ context.Context, _ string, d domain.Definition, now time.Time) (domain.Definition, error) {
	d.ID = "def-1"
	d.CreatedAt = now
	d.UpdatedAt = now
	s.definitions[d.ID] = application.Definition{ID: d.ID, EntityType: d.EntityType, Key: d.Key, Label: d.Label, FieldType: string(d.FieldType), Status: string(d.Status), Version: 1, CreatedAt: now, UpdatedAt: now}
	return d, nil
}
func (s *stubFieldStore) Update(_ context.Context, _, id string, d domain.Definition, _ time.Time) (domain.Definition, error) {
	d.ID = id
	s.definitions[id] = application.Definition{ID: id, EntityType: d.EntityType, Key: d.Key, Label: d.Label, FieldType: string(d.FieldType), Status: string(d.Status), Version: d.Version}
	return d, nil
}
func (s *stubFieldStore) Retire(_ context.Context, _, id string, now time.Time) (domain.Definition, error) {
	d := s.definitions[id]
	d.Status = string(domain.StatusRetired)
	d.RetiredAt = &now
	s.definitions[id] = d
	return domain.Definition{ID: id, Status: domain.StatusRetired, RetiredAt: &now}, nil
}
func (s *stubFieldStore) History(_ context.Context, _, _ string) ([]application.HistoryEntry, error) {
	return nil, nil
}
func (s *stubFieldStore) HasValues(_ context.Context, _, _ string) (bool, error) { return false, nil }

func withFieldContext(r *http.Request, principal identitydomain.Principal) *http.Request {
	return r.WithContext(identitydomain.WithPrincipal(r.Context(), principal))
}

func adminPrincipal() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
			identitydomain.PermissionFieldManage:  {},
		},
	}
}

func TestCreateFieldDefinitionHandler(t *testing.T) {
	service := application.Service{Store: &stubFieldStore{definitions: map[string]application.Definition{}}, Now: func() time.Time { return time.Unix(0, 0).UTC() }}
	handler := createFieldDefinition(service)
	body := `{"entity_type":"incident","key":"po_number","label":"PO Number","field_type":"TEXT","config":{"required":true},"sort_order":1}`
	req := withFieldContext(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions", strings.NewReader(body)), adminPrincipal())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var response application.Definition
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Key != "po_number" || response.ID != "def-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCreateFieldDefinitionValidation422(t *testing.T) {
	service := application.Service{Store: &stubFieldStore{definitions: map[string]application.Definition{}}}
	handler := createFieldDefinition(service)
	body := `{"entity_type":"incident","key":"PO_BAD","label":"Bad","field_type":"TEXT"}`
	req := withFieldContext(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions", strings.NewReader(body)), adminPrincipal())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid definition, got %d", rec.Code)
	}
}
```

Note: `identitydomain.WithPrincipal` — verify the exact helper name in `internal/identity/domain/principal_context.go` (or wherever `PrincipalFromContext` lives). If it is `WithPrincipal(ctx, p)`, use it; otherwise adapt.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/platform/httpserver/ -run TestCreateFieldDefinition`
Expected: FAIL (undefined functions).

- [ ] **Step 3: Implement the handlers**

`internal/platform/httpserver/customfields.go`:

```go
package httpserver

import (
	"errors"
	"net/http"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/go-chi/chi/v5"
)

func mountCustomFieldRoutes(router chi.Router, fields application.Service) {
	router.Get("/admin/field-definitions", listFieldDefinitions(fields))
	router.Post("/admin/field-definitions", createFieldDefinition(fields))
	router.Get("/admin/field-definitions/{fieldID}", getFieldDefinition(fields))
	router.Patch("/admin/field-definitions/{fieldID}", updateFieldDefinition(fields))
	router.Post("/admin/field-definitions/{fieldID}/retire", retireFieldDefinition(fields))
	router.Get("/admin/field-definitions/{fieldID}/history", fieldDefinitionHistory(fields))
}

func listFieldDefinitions(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		entityType := r.URL.Query().Get("entity_type")
		if entityType == "" {
			entityType = "incident"
		}
		items, err := service.ListByEntity(r.Context(), principal, entityType)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		command, ok := decodeCommand[application.CreateDefinition](w, r)
		if !ok {
			return
		}
		result, err := service.Create(r.Context(), principal, command)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	}
}

func getFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		result, err := service.Get(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func updateFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		command, ok := decodeCommand[application.UpdateDefinition](w, r)
		if !ok {
			return
		}
		result, err := service.Update(r.Context(), principal, id, command)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func retireFieldDefinition(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		result, err := service.Retire(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func fieldDefinitionHistory(service application.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "fieldID")
		if !validUUIDParam(w, id, "field definition ID") {
			return
		}
		items, err := service.History(r.Context(), principal, id)
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func writeFieldError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "Forbidden", "permission denied")
	case errors.Is(err, application.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "Not Found", "field definition was not found")
	case errors.Is(err, application.ErrValidation):
		writeProblem(w, http.StatusUnprocessableEntity, "Unprocessable Entity", err.Error())
	case errors.Is(err, application.ErrConflict):
		writeProblem(w, http.StatusConflict, "Conflict", err.Error())
	case errors.Is(err, application.ErrInvalid):
		writeProblem(w, http.StatusBadRequest, "Invalid Request", err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "request could not be completed")
	}
}
```

- [ ] **Step 4: Wire routes in router.go**

In `internal/platform/httpserver/router.go`:
1. Add import `customfieldapp "github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"`.
2. Add to `Dependencies` struct: `CustomFields customfieldapp.Service`.
3. In `New`, after `mountEvaluationRoutes(api, dependencies.Evaluations)`, add:
   `mountCustomFieldRoutes(api, dependencies.CustomFields)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/platform/httpserver/ -run TestCreateFieldDefinition -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/platform/httpserver/customfields.go internal/platform/httpserver/customfields_test.go internal/platform/httpserver/router.go
git commit -m "feat: add admin field definition API routes"
```

---

### Task 7: Incident integration — values, history, and filter

**Files:**
- Modify: `internal/incident/domain/incident.go` (add `CustomValues` field)
- Modify: `internal/incident/application/service.go` (commands, `Fields` interface, `UpdateCustomValues`)
- Modify: `internal/incident/adapter/postgres/store.go` (create/select/scan/map/update-custom-values/history/filter)
- Modify: `internal/incident/application/service_test.go` (existing tests still pass; add new ones)
- Create: `internal/incident/adapter/postgres/custom_values_integration_test.go` (optional; rely on existing suite + new unit tests)

**Interfaces:**
- Consumes: `customfield/application` (Task 3).
- Produces:
  - `incidentapp.CreateIncident.CustomValues map[string]any json:"custom_values,omitempty"`
  - `incidentapp.Incident.CustomValues map[string]any json:"custom_values,omitempty"`
  - `incidentapp.Filter.CustomFields map[string]string` (definition-id-keyed, at store boundary)
  - `incidentapp.Service.Fields Fields` where
    ```go
    type Fields interface {
        Definitions(ctx context.Context, organizationID, entityType string) ([]customfieldapp.Definition, error)
        ResolveAndValidate(ctx context.Context, organizationID, entityType string, values map[string]any) (map[string]any, error)
        ResolveKeys(ctx context.Context, organizationID, entityType string, keys []string) (map[string]customfieldapp.Definition, error)
    }
    ```
  - `type UpdateCustomValues struct { ExpectedVersion int64; CustomValues map[string]any }`
  - `Service.UpdateCustomValues(ctx, principal, key, incidentID string, command UpdateCustomValues) (Incident, bool, error)`
  - Store interface gains `UpdateCustomValues(ctx, principal, key, incidentID string, command UpdateCustomValues) (Incident, bool, error)`.

- [ ] **Step 1: Extend the domain type**

In `internal/incident/domain/incident.go`, add to the `Incident` struct (after `ResolutionSummary`):

```go
	CustomValues      map[string]any
```

- [ ] **Step 2: Extend the application service**

In `internal/incident/application/service.go`:

1. Add import `customfieldapp "github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"`.
2. Add `CustomValues map[string]any` to `CreateIncident` and to the `Incident` app struct (with `json:"custom_values,omitempty"`).
3. Add `CustomFields map[string]string` to `Filter`.
4. Add the `Fields` interface and `Service.Fields` field:

```go
type Fields interface {
	Definitions(ctx context.Context, organizationID, entityType string) ([]customfieldapp.Definition, error)
	ResolveAndValidate(ctx context.Context, organizationID, entityType string, values map[string]any) (map[string]any, error)
	ResolveKeys(ctx context.Context, organizationID, entityType string, keys []string) (map[string]customfieldapp.Definition, error)
}
```

5. In `Service` struct: add `Fields Fields`.

6. In `Create`, after the basic `ErrInvalid` check, add:

```go
	if len(command.CustomValues) > 0 {
		if s.Fields == nil {
			return Incident{}, false, errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		normalized, err := s.Fields.ResolveAndValidate(ctx, principal.OrganizationID, "incident", command.CustomValues)
		if err != nil {
			return Incident{}, false, errors.Join(ErrInvalid, err)
		}
		command.CustomValues = normalized
	}
```

7. In `List`, after cursor decoding, translate custom field keys to definition ids:

```go
	if len(filter.CustomFields) > 0 && s.Fields != nil {
		keys := make([]string, 0, len(filter.CustomFields))
		for key := range filter.CustomFields {
			keys = append(keys, key)
		}
		resolved, err := s.Fields.ResolveKeys(ctx, principal.OrganizationID, "incident", keys)
		if err != nil {
			return nil, "", errors.Join(ErrInvalid, err)
		}
		byID := make(map[string]string, len(filter.CustomFields))
		for key, value := range filter.CustomFields {
			byID[resolved[key].ID] = value
		}
		filter.CustomFields = byID
	}
```

8. Add the update command and service method:

```go
type UpdateCustomValues struct {
	ExpectedVersion int64          `json:"expected_version"`
	CustomValues    map[string]any `json:"custom_values"`
}

func (s Service) UpdateCustomValues(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command UpdateCustomValues,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	if len(command.CustomValues) > 0 {
		if s.Fields == nil {
			return Incident{}, false, errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		normalized, err := s.Fields.ResolveAndValidate(ctx, principal.OrganizationID, "incident", command.CustomValues)
		if err != nil {
			return Incident{}, false, errors.Join(ErrInvalid, err)
		}
		command.CustomValues = normalized
	}
	return s.Store.UpdateCustomValues(ctx, principal, key, incidentID, command)
}
```

- [ ] **Step 3: Extend the store**

In `internal/incident/adapter/postgres/store.go`:

1. `Create` INSERT: add `custom_values` to columns and `$22::jsonb` to values; marshal the map:

```go
	customValues, err := json.Marshal(command.CustomValues)
	if err != nil {
		return outcome{}, err
	}
```

and extend both the column list and the arg list (`customValues` last, before `now`), shifting `now` to `$23`. Update the INSERT:

```go
			INSERT INTO incidents (
				id, organization_id, site_id, asset_id, number, summary, details, priority,
				status, source_of_truth, external_system, external_id, external_version,
				occurred_at, detected_at, version, created_by, assignee_id, reporter_id, team_id,
				custom_values, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, nullif($7, ''), $8,
				$9, $10, nullif($11, ''), nullif($12, ''), nullif($13, ''),
				$14, $15, $16, $17::uuid, nullif($18, '')::uuid, $19::uuid, nullif($20, '')::uuid,
				$21::jsonb, $22, $22
			)
```

with args ending `..., incident.TeamID, customValues, now)`.

2. After the insert, write history rows (inside the same tx):

```go
	for definitionID, value := range command.CustomValues {
		encoded, err := json.Marshal(value)
		if err != nil {
			return outcome{}, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO custom_value_history (
				incident_id, field_definition_id, principal_id, value_before, value_after, changed_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, NULL, $4::jsonb, $5)
		`, incident.ID, definitionID, principal.ID, encoded, now); err != nil {
			return outcome{}, err
		}
	}
```

3. `incidentSelect`: append `i.custom_values` after `coalesce(team.name, '')`:

```sql
		coalesce(i.team_id::text, ''), coalesce(team.name, ''),
		i.custom_values
```

4. `scanIncident` and `loadIncidentForUpdate`: scan the extra column into a local `[]byte` and unmarshal:

```go
	var rawCustom []byte
	err := row.Scan(..., &value.TeamID, &value.TeamName, &rawCustom)
	if err != nil {
		return incidentapp.Incident{}, err
	}
	value.CustomValues = map[string]any{}
	if len(rawCustom) > 0 {
		if err := json.Unmarshal(rawCustom, &value.CustomValues); err != nil {
			return incidentapp.Incident{}, err
		}
	}
```

(apply the same shape inside `loadIncidentForUpdate`, then pass `value.CustomValues` into the `incidentdomain.Incident{...}` literal as `CustomValues: value.CustomValues`).

5. `mapIncident`: copy `CustomValues: value.CustomValues`.

6. `List`: add filter conditions after the status filter:

```go
	if len(filter.CustomFields) > 0 {
		defIDs := make([]string, 0, len(filter.CustomFields))
		for defID := range filter.CustomFields {
			defIDs = append(defIDs, defID)
		}
		sort.Strings(defIDs)
		for _, defID := range defIDs {
			filterValue := filter.CustomFields[defID]
			keyArg := len(args) + 1
			valueArg := len(args) + 2
			args = append(args, defID, filterValue)
			query += fmt.Sprintf(" AND i.custom_values ? $%d::text AND i.custom_values->>$%d::text = $%d", keyArg, keyArg, valueArg)
		}
	}
```

Add `"sort"` to the store's imports.

7. Add the `UpdateCustomValues` store method (mirror the `Resolve`/`Close` shape: idempotency, tx, `loadIncidentForUpdate`, version check, diff + update `custom_values`, history rows, event, audit):

```go
func (s Store) UpdateCustomValues(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.UpdateCustomValues,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.custom_values.v1:" + incidentID
	type outcome struct {
		Value  incidentapp.Incident
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay incidentapp.Incident
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, err
			}
			return outcome{Value: replay, Replay: true}, nil
		}
		incident, assetTag, err := loadIncidentForUpdate(ctx, tx, principal, incidentID)
		if err != nil {
			return outcome{}, err
		}
		if incident.Version != command.ExpectedVersion {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		before := mapIncident(incident, assetTag)

		previous := incident.CustomValues
		if previous == nil {
			previous = map[string]any{}
		}
		merged := make(map[string]any, len(previous)+len(command.CustomValues))
		for defID, value := range previous {
			merged[defID] = value
		}
		for defID, value := range command.CustomValues {
			merged[defID] = value
		}
		encoded, err := json.Marshal(merged)
		if err != nil {
			return outcome{}, err
		}
		newVersion := incident.Version + 1
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET custom_values = $2::jsonb, version = $3, updated_at = $4
			WHERE id = $1::uuid AND version = $5
		`, incident.ID, encoded, newVersion, now, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		for defID, value := range command.CustomValues {
			encodedValue, err := json.Marshal(value)
			if err != nil {
				return outcome{}, err
			}
			var beforeValue any
			if raw, ok := previous[defID]; ok {
				beforeValue = raw
			}
			beforeEncoded, err := json.Marshal(beforeValue)
			if err != nil {
				return outcome{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO custom_value_history (
					incident_id, field_definition_id, principal_id, value_before, value_after, changed_at
				) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::jsonb, $5::jsonb, $6)
			`, incident.ID, defID, principal.ID, beforeEncoded, encodedValue, now); err != nil {
				return outcome{}, err
			}
		}
		incident.CustomValues = merged
		incident.Version = newVersion
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentCustomValuesUpdated", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.custom_values.updated", EntityKind: "incident",
			EntityID: incident.ID, Before: before, After: response, OccurredAt: now,
		}); err != nil {
			return outcome{}, err
		}
		body, _ := json.Marshal(response)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return outcome{}, err
		}
		return outcome{Value: response}, nil
	})
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	return result.Value, result.Replay, nil
}
```

- [ ] **Step 4: Add application tests**

In `internal/incident/application/service_test.go`, add a fake `Fields` and tests:

```go
type fakeFields struct{}

func (fakeFields) Definitions(context.Context, string, string) ([]customfieldapp.Definition, error) {
	return []customfieldapp.Definition{{ID: "def-1", Key: "po_number", FieldType: "TEXT", Config: domain.Config{Required: true}}}, nil
}
func (fakeFields) ResolveAndValidate(_ context.Context, _, _ string, values map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for key, value := range values {
		if key != "po_number" {
			return nil, errors.New("unknown custom field")
		}
		out["def-1"] = value
	}
	return out, nil
}
func (fakeFields) ResolveKeys(_ context.Context, _, _ string, keys []string) (map[string]customfieldapp.Definition, error) {
	out := map[string]customfieldapp.Definition{}
	for _, key := range keys {
		out[key] = customfieldapp.Definition{ID: "def-1", Key: key, FieldType: "TEXT"}
	}
	return out, nil
}

func TestCreateValidatesCustomValues(t *testing.T) {
	service := Service{
		Store: &stubStore{},
		Fields: fakeFields{},
	}
	command := CreateIncident{
		SiteID: "s1", AssetID: "a1", Summary: "Pump vibration", Priority: "HIGH",
		DetectedAt: time.Now(), CustomValues: map[string]any{"po_number": "PO-42"},
	}
	principal := identitydomain.Principal{
		OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionIncidentCreate: {}},
	}
	incident, _, err := service.Create(context.Background(), principal, "test-key", command)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	value, ok := incident.CustomValues["def-1"]
	if !ok || value != "PO-42" {
		t.Fatalf("expected normalized custom value keyed by definition id, got %#v", incident.CustomValues)
	}
}
```

`stubStore` must exist in the test package (check `service_test.go` — if the existing test file already defines one with the old `Store` interface, extend it with `UpdateCustomValues` returning `ErrForbidden`-style placeholder; the new interface method must exist for the fake to satisfy `Store`).

- [ ] **Step 5: Run backend tests**

Run: `go test ./internal/incident/...`
Expected: PASS (update any existing fake store that must now implement `UpdateCustomValues`).

- [ ] **Step 6: Commit**

```bash
git add internal/incident/
git commit -m "feat: persist and validate incident custom field values"
```

---

### Task 8: Wire services in cmd/api and incident routes

**Files:**
- Modify: `cmd/api/main.go` (~line 199 area)
- Modify: `internal/platform/httpserver/incidents.go` (list/get handlers gain `custom_fields`)
- Modify: `internal/platform/httpserver/router.go` (mountIncidentRoutes signature)

**Interfaces:**
- Consumes: Tasks 3, 6, 7.
- Produces: wired `customfieldapp.Service`; incident list/detail responses include `custom_fields: []customfieldapp.Definition`.

- [ ] **Step 1: Wire in cmd/api/main.go**

Near the existing service construction, add:

```go
	customFieldStore := customfieldpostgres.Store{Pool: pool}
	customFieldService := customfieldapp.Service{Store: customFieldStore}
```

Add imports for `customfieldapp` and `customfieldpostgres`. Then:

1. Change the incident service to inject fields:

```go
		Incidents: incidentapp.Service{
			Store:  incidentStore,
			Fields: customFieldService,
		},
```

2. Add to `httpserver.Dependencies` (already added in Task 6): `CustomFields: customFieldService,`.

- [ ] **Step 2: Extend incident handlers with custom_fields**

In `internal/platform/httpserver/incidents.go`:

1. `mountIncidentRoutes` gains a `customFields customfieldapp.Service` parameter:

```go
func mountIncidentRoutes(
	router chi.Router,
	incidents incidentapp.Service,
	executions executionapp.Service,
	attachments attachmentapp.Service,
	customFields customfieldapp.Service,
) {
```

2. `listIncidents`:

```go
		definitions, _ := customFields.Definitions(r.Context(), principal.OrganizationID, "incident")
		writeJSON(w, http.StatusOK, map[string]any{
			"items":        items,
			"custom_fields": definitions,
			"next_cursor":  nullableString(next),
			"has_more":     next != "",
		})
```

3. `getIncident`: after computing `result`, fetch definitions and embed:

```go
		definitions, err := customFields.Definitions(r.Context(), principal.OrganizationID, "incident")
		if err != nil {
			writeFieldError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			incidentapp.Incident
			Attachments []attachmentapp.Attachment          `json:"attachments"`
			CustomFields []customfieldapp.Definition         `json:"custom_fields"`
		}{result, items, definitions})
```

4. Update `router.go` call site to pass `dependencies.CustomFields`.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: compiles.

- [ ] **Step 4: Run existing tests**

Run: `go test ./internal/platform/httpserver/ ./cmd/...`
Expected: PASS (fix `mountIncidentRoutes` callers in tests — search for `mountIncidentRoutes(` in `*_test.go` and add the extra argument; use `customfieldapp.Service{Store: &stubFieldStore{definitions: map[string]customfieldapp.Definition{}}}` or nil service as the test requires).

- [ ] **Step 5: Commit**

```bash
git add cmd/api/main.go internal/platform/httpserver/incidents.go internal/platform/httpserver/router.go
git commit -m "feat: expose custom field definitions on incident list and detail"
```

---

### Task 9: OpenAPI documentation

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `web/public/openapi.yaml` (synced copy)

**Interfaces:**
- Produces: documented `FieldDefinition`, `CreateFieldDefinition`, `UpdateFieldDefinition`, `HistoryEntry` schemas and the admin routes; incident schemas gain `custom_values` and `custom_fields`.

- [ ] **Step 1: Add schemas**

Add to `components.schemas` (match existing style — check how `Incident` is documented first):

```yaml
    FieldDefinition:
      type: object
      properties:
        id: { type: string, format: uuid }
        entity_type: { type: string }
        key: { type: string, pattern: '^[a-z][a-z0-9_]{1,63}$' }
        label: { type: string }
        description: { type: string }
        field_type:
          type: string
          enum: [TEXT, NUMBER, DATE, SELECT, MULTI_SELECT]
        config:
          type: object
          properties:
            required: { type: boolean }
            max_length: { type: integer }
            regex: { type: string }
            min: { type: number }
            max: { type: number }
            options:
              type: array
              items:
                type: object
                properties:
                  label: { type: string }
                  value: { type: string }
        status: { type: string, enum: [ACTIVE, RETIRED] }
        sort_order: { type: integer }
        version: { type: integer }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
        retired_at: { type: string, format: date-time, nullable: true }
    HistoryEntry:
      type: object
      properties:
        incident_id: { type: string, format: uuid }
        principal_id: { type: string, format: uuid }
        value_before: {}
        value_after: {}
        changed_at: { type: string, format: date-time }
```

- [ ] **Step 2: Document the routes**

Add under `/api/v1` paths (mirror existing path style):

```yaml
    /admin/field-definitions:
      get:
        parameters:
          - name: entity_type
            in: query
            schema: { type: string, default: incident }
        responses:
          '200':
            content:
              application/json:
                schema:
                  type: object
                  properties:
                    items:
                      type: array
                      items: { $ref: '#/components/schemas/FieldDefinition' }
      post:
        requestBody:
          required: true
          content:
            application/json:
              schema:
                type: object
                required: [entity_type, key, label, field_type]
                properties:
                  entity_type: { type: string }
                  key: { type: string }
                  label: { type: string }
                  description: { type: string }
                  field_type: { type: string, enum: [TEXT, NUMBER, DATE, SELECT, MULTI_SELECT] }
                  config: { $ref: '#/components/schemas/FieldDefinition/properties/config' }
                  sort_order: { type: integer }
        responses:
          '201':
            content:
              application/json:
                schema: { $ref: '#/components/schemas/FieldDefinition' }
    /admin/field-definitions/{fieldID}:
      get:
        responses:
          '200':
            content:
              application/json:
                schema: { $ref: '#/components/schemas/FieldDefinition' }
      patch:
        requestBody:
          required: true
          content:
            application/json:
              schema:
                type: object
                required: [label, field_type, config, sort_order, expected_version]
                properties:
                  label: { type: string }
                  description: { type: string }
                  field_type: { type: string }
                  config: { $ref: '#/components/schemas/FieldDefinition/properties/config' }
                  sort_order: { type: integer }
                  expected_version: { type: integer }
        responses:
          '200':
            content:
              application/json:
                schema: { $ref: '#/components/schemas/FieldDefinition' }
    /admin/field-definitions/{fieldID}/retire:
      post:
        responses:
          '200':
            content:
              application/json:
                schema: { $ref: '#/components/schemas/FieldDefinition' }
    /admin/field-definitions/{fieldID}/history:
      get:
        responses:
          '200':
            content:
              application/json:
                schema:
                  type: object
                  properties:
                    items:
                      type: array
                      items: { $ref: '#/components/schemas/HistoryEntry' }
```

- [ ] **Step 3: Update Incident schema**

Add `custom_values: { type: object, additionalProperties: true }` and, in list/detail response wrappers, `custom_fields: { type: array, items: { $ref: '#/components/schemas/FieldDefinition' } }`.

- [ ] **Step 4: Sync the web copy**

Run: `cp api/openapi.yaml web/public/openapi.yaml`
Verify no accidental drift: `git diff --stat api/openapi.yaml web/public/openapi.yaml`.

- [ ] **Step 5: Validate**

Run: `go run ./cmd/api` smoke (or `python -c "import yaml,sys; yaml.safe_load(open('api/openapi.yaml'))"` if python available) to confirm the YAML parses.
Expected: no parse errors.

- [ ] **Step 6: Commit**

```bash
git add api/openapi.yaml web/public/openapi.yaml
git commit -m "docs: document custom field definitions API"
```

---

### Task 10: Web types and API client

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`
- Create: `web/src/api.test.ts` additions (or extend existing)

**Interfaces:**
- Consumes: Task 9 schema names.
- Produces (used by Tasks 11–12):
  - `web/src/types.ts`: `CustomFieldType`, `CustomFieldStatus`, `CustomFieldOption`, `CustomFieldConfig`, `CustomFieldDefinition`; `Incident.custom_values?: Record<string, unknown>`; `IncidentListResponse = ListPage<Incident> & { custom_fields: CustomFieldDefinition[] }`.
  - `web/src/api.ts`: `api.listFieldDefinitions(entityType)`, `api.createFieldDefinition(value)`, `api.updateFieldDefinition(id, value)`, `api.retireFieldDefinition(id)`, `api.fieldDefinitionHistory(id)`, `api.patchFieldDefinition` (internal), `api.createIncident` accepts `custom_values`, `api.incidents` accepts custom field filters via `ListOptions.custom_fields`.

- [ ] **Step 1: Write failing tests**

Add to `web/src/api.test.ts`:

```ts
test("createIncident forwards custom_values", async () => {
  mockFetchOnce({ id: "inc-1", custom_values: { "def-1": "PO-42" } });
  await api.createIncident({
    site_id: "s1", asset_id: "a1", summary: "Pump", priority: "HIGH",
    custom_values: { po_number: "PO-42" }
  });
  const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
  const body = JSON.parse(String(init.body));
  expect(body.custom_values).toEqual({ po_number: "PO-42" });
});

test("incident list query includes custom field filters", async () => {
  mockFetchOnce({ items: [], next_cursor: null, has_more: false, custom_fields: [] });
  await api.incidents({ custom_fields: { po_number: "PO-42" } });
  const [url] = fetchMock.mock.calls[0] as [string];
  expect(url).toContain("custom_field.po_number=PO-42");
});
```

(Match the file's existing `mockFetchOnce`/`fetchMock` helpers — read `web/src/api.test.ts` first and reuse its harness.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npm test -- api.test`
Expected: FAIL (types/functions missing).

- [ ] **Step 3: Implement types**

In `web/src/types.ts`, add after the `Team`/`Person` block:

```ts
export type CustomFieldType = "TEXT" | "NUMBER" | "DATE" | "SELECT" | "MULTI_SELECT";
export type CustomFieldStatus = "ACTIVE" | "RETIRED";

export type CustomFieldOption = { label: string; value: string };

export type CustomFieldConfig = {
  required?: boolean;
  max_length?: number;
  regex?: string;
  min?: number;
  max?: number;
  options?: CustomFieldOption[];
};

export type CustomFieldDefinition = {
  id: string;
  entity_type: string;
  key: string;
  label: string;
  description?: string;
  field_type: CustomFieldType;
  config: CustomFieldConfig;
  status: CustomFieldStatus;
  sort_order: number;
  version: number;
  created_at: string;
  updated_at: string;
  retired_at?: string;
};
```

Add to `Incident`:

```ts
  custom_values?: Record<string, unknown>;
```

Add:

```ts
export type IncidentListResponse = ListPage<Incident> & {
  custom_fields: CustomFieldDefinition[];
};
```

- [ ] **Step 4: Implement the API client**

In `web/src/api.ts`:

1. Extend `ListOptions`:

```ts
export interface ListOptions {
  site_id?: string;
  state?: string[];
  cursor?: string;
  page_size?: number;
  custom_fields?: Record<string, string>;
}
```

2. In `listQuery`, after the `page_size` line:

```ts
  for (const [key, value] of Object.entries(options?.custom_fields ?? {})) {
    params.set(`custom_field.${key}`, value);
  }
```

3. Change `incidents` to use the response type:

```ts
  incidents: (options?: ListOptions) =>
    request<IncidentListResponse>(`/incidents${listQuery(options)}`),
```

4. Add a `patch` helper next to `command`:

```ts
function patch<T>(path: string, value: unknown): Promise<T> {
  return request<T>(path, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(value)
  });
}
```

5. Add to the `api` object:

```ts
  listFieldDefinitions: (entityType = "incident") =>
    request<{ items: CustomFieldDefinition[] }>(
      `/admin/field-definitions?entity_type=${encodeURIComponent(entityType)}`
    ),
  createFieldDefinition: (value: {
    entity_type: string;
    key: string;
    label: string;
    description?: string;
    field_type: CustomFieldType;
    config: CustomFieldConfig;
    sort_order: number;
  }) => command<CustomFieldDefinition>("/admin/field-definitions", value),
  updateFieldDefinition: (id: string, value: {
    label: string;
    description?: string;
    field_type: CustomFieldType;
    config: CustomFieldConfig;
    sort_order: number;
    expected_version: number;
  }) => patch<CustomFieldDefinition>(`/admin/field-definitions/${id}`, value),
  retireFieldDefinition: (id: string) =>
    command<CustomFieldDefinition>(`/admin/field-definitions/${id}/retire`, {}),
  fieldDefinitionHistory: (id: string) =>
    request<{ items: HistoryEntry[] }>(`/admin/field-definitions/${id}/history`)
```

6. Extend `createIncident`'s value type and payload:

```ts
  createIncident: (value: {
    site_id: string;
    asset_id: string;
    summary: string;
    details?: string;
    priority: string;
    status?: string;
    assignee_id?: string;
    reporter_id?: string;
    team_id?: string;
    occurred_at?: string;
    custom_values?: Record<string, unknown>;
  }) =>
    command<Incident>("/incidents", {
      ...value,
      source_of_truth: "OWNED_BY_SKAWLD",
      detected_at: new Date().toISOString()
    }),
```

Add `HistoryEntry` type in `web/src/types.ts`:

```ts
export type HistoryEntry = {
  incident_id: string;
  principal_id: string;
  value_before?: unknown;
  value_after: unknown;
  changed_at: string;
};
```

- [ ] **Step 5: Run tests, lint, build**

Run: `cd web && npm test -- api.test && npm run lint && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.test.ts
git commit -m "feat: add web types and client for custom fields"
```

---

### Task 11: Dynamic custom field controls + create form integration

**Files:**
- Create: `web/src/console/components/CustomFieldControl.tsx`
- Create: `web/src/console/components/CustomFieldControl.test.tsx`
- Modify: `web/src/console/components/CreateIncidentForm.tsx`
- Modify: `web/src/console/components/CreateIncidentForm.test.tsx` (if it exists — check; otherwise add assertions to the page-level test)

**Interfaces:**
- Consumes: `CustomFieldDefinition` (Task 10), i18n `messages.ts`.
- Produces:
  - `CustomFieldControl(props: { field: CustomFieldDefinition; value: unknown; onChange: (value: unknown) => void })` — renders the right control per `field_type` and shows a required marker; value types: TEXT/DATE/SELECT → `string`, NUMBER → `number | ""`, MULTI_SELECT → `string[]`.
  - `CreateIncidentValue.custom_values?: Record<string, unknown>`.

- [ ] **Step 1: Write failing tests**

`web/src/console/components/CustomFieldControl.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CustomFieldControl } from "./CustomFieldControl";

function textField(): CustomFieldDefinition {
  return {
    id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number",
    field_type: "TEXT", config: { required: true }, status: "ACTIVE",
    sort_order: 1, version: 1, created_at: "", updated_at: ""
  };
}

test("renders a text input for TEXT fields", () => {
  render(<CustomFieldControl field={textField()} value="" onChange={() => {}} />);
  expect(screen.getByLabelText(/PO Number/)).toBeInTheDocument();
});

test("renders select with options for SELECT fields", () => {
  const field = { ...textField(), field_type: "SELECT" as const, config: { options: [{ label: "Zone A", value: "a" }, { label: "Zone B", value: "b" }] } };
  render(<CustomFieldControl field={field} value="" onChange={() => {}} />);
  expect(screen.getByRole("combobox")).toBeInTheDocument();
});

test("calls onChange with typed value", async () => {
  const user = userEvent.setup();
  const onChange = vi.fn();
  render(<CustomFieldControl field={textField()} value="" onChange={onChange} />);
  await user.type(screen.getByLabelText(/PO Number/), "PO-42");
  expect(onChange).toHaveBeenCalledWith("PO-42");
});
```

(Match the test file conventions in this repo — check an existing component test, e.g. `web/src/console/ui/avatar.test.tsx`, for the vitest/testing-library setup.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npm test -- CustomFieldControl`
Expected: FAIL (component missing).

- [ ] **Step 3: Implement the component**

`web/src/console/components/CustomFieldControl.tsx`:

```tsx
import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";
import type { CustomFieldDefinition } from "../../types";

export function CustomFieldControl(props: {
  field: CustomFieldDefinition;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const { t } = useI18n();
  const { field, value } = props;
  const label = field.label;
  const required = field.config.required === true;

  switch (field.field_type) {
    case "TEXT":
      return (
        <FormField label={label} required={required} hint={field.description}>
          <input
            type="text"
            aria-label={label}
            value={typeof value === "string" ? value : ""}
            maxLength={field.config.max_length}
            onChange={(event) => props.onChange(event.target.value)}
          />
        </FormField>
      );
    case "NUMBER":
      return (
        <FormField label={label} required={required} hint={field.description}>
          <input
            type="number"
            aria-label={label}
            value={typeof value === "number" ? value : ""}
            min={field.config.min}
            max={field.config.max}
            onChange={(event) =>
              props.onChange(event.target.value === "" ? undefined : Number(event.target.value))
            }
          />
        </FormField>
      );
    case "DATE":
      return (
        <FormField label={label} required={required} hint={field.description}>
          <input
            type="date"
            aria-label={label}
            value={typeof value === "string" ? value : ""}
            onChange={(event) => props.onChange(event.target.value)}
          />
        </FormField>
      );
    case "SELECT":
      return (
        <FormField label={label} required={required} hint={field.description}>
          <select
            aria-label={label}
            value={typeof value === "string" ? value : ""}
            onChange={(event) => props.onChange(event.target.value)}
          >
            <option value="">{t("form.selectPlaceholder")}</option>
            {field.config.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </FormField>
      );
    case "MULTI_SELECT":
      return (
        <FormField label={label} required={required} hint={field.description}>
          <div role="group" aria-label={label}>
            {field.config.options?.map((option) => {
              const selected = Array.isArray(value) && value.includes(option.value);
              return (
                <label key={option.value} className="option-row">
                  <input
                    type="checkbox"
                    checked={selected}
                    onChange={() => {
                      const current = Array.isArray(value) ? value : [];
                      const next = selected
                        ? current.filter((item) => item !== option.value)
                        : [...current, option.value];
                      props.onChange(next);
                    }}
                  />
                  {option.label}
                </label>
              );
            })}
          </div>
        </FormField>
      );
  }
}
```

Add i18n keys `form.selectPlaceholder` to both `en` and `vi` blocks in `web/src/i18n/messages.ts` ("Select an option" / "Chọn một lựa chọn").

- [ ] **Step 4: Integrate into CreateIncidentForm**

In `web/src/console/components/CreateIncidentForm.tsx`:

1. Extend props:

```tsx
export function CreateIncidentForm(props: {
  assets: Asset[];
  people: Person[];
  teams: Team[];
  principal: Principal | undefined;
  customFields: CustomFieldDefinition[];
  pending: boolean;
  onCancel: () => void;
  onCreate: (value: CreateIncidentValue) => void;
}) {
```

2. Extend `CreateIncidentValue`:

```ts
export interface CreateIncidentValue {
  site_id: string;
  asset_id: string;
  summary: string;
  details?: string;
  priority: string;
  status: string;
  assignee_id?: string;
  reporter_id: string;
  team_id?: string;
  occurred_at?: string;
  files: File[];
  custom_values?: Record<string, unknown>;
}
```

3. Add state:

```ts
  const [customValues, setCustomValues] = useState<Record<string, unknown>>({});
```

4. Validation: required custom fields must have non-empty values. After the existing `errors` object, add:

```ts
  const customFieldErrors = props.customFields.reduce<Record<string, string | undefined>>(
    (acc, field) => {
      const value = customValues[field.key];
      const empty = value === undefined || value === null || value === "" ||
        (Array.isArray(value) && value.length === 0);
      if (field.config.required && empty) acc[field.key] = t("form.required");
      return acc;
    },
    {},
  );
  const customValid = Object.values(customFieldErrors).every((error) => error === undefined);
```

and include `customValid` in the `valid` expression.

5. In the submit handler, include `custom_values` in the value passed to `onCreate` (build the object; find the existing submit function and add `custom_values: customValues`).

6. Render the custom section after the existing fields (before the submit row), only when `props.customFields.length > 0`:

```tsx
        {props.customFields.length > 0 && (
          <fieldset className="form-section">
            <legend>{t("incident.customFields")}</legend>
            {props.customFields.map((field) => (
              <CustomFieldControl
                key={field.id}
                field={field}
                value={customValues[field.key]}
                onChange={(value) =>
                  setCustomValues((prev) => ({ ...prev, [field.key]: value }))
                }
              />
            ))}
          </fieldset>
        )}
```

Add i18n key `incident.customFields` ("Custom fields" / "Trường tùy chỉnh").

7. Import `CustomFieldDefinition` type and `CustomFieldControl`.

- [ ] **Step 5: Update the caller**

Find where `CreateIncidentForm` is rendered (likely `IncidentsPage.tsx` or a dialog component). Pass `customFields={customFields}` where `customFields` comes from a `useQuery(() => api.listFieldDefinitions("incident"))`. If the dialog component owns the fetch, add it there. Ensure `onCreate` forwards `custom_values` into `api.createIncident`.

- [ ] **Step 6: Run web tests + lint + build**

Run: `cd web && npm test && npm run lint && npm run build`
Expected: PASS (update `CreateIncidentForm` tests for the new prop — add `customFields={[]}` to existing render calls).

- [ ] **Step 7: Commit**

```bash
git add web/src/console/components/CustomFieldControl.tsx web/src/console/components/CustomFieldControl.test.tsx web/src/console/components/CreateIncidentForm.tsx web/src/i18n/messages.ts
git commit -m "feat: render dynamic custom field controls in the incident form"
```

---

### Task 12: Admin custom fields page

**Files:**
- Create: `web/src/console/pages/CustomFieldsPage.tsx`
- Create: `web/src/console/pages/CustomFieldsPage.test.tsx`
- Modify: `web/src/App.tsx` (route)
- Modify: `web/src/console/layout/Sidebar.tsx` (nav item, `field:manage` gated)
- Modify: `web/src/i18n/messages.ts` (nav + page keys)

**Interfaces:**
- Consumes: `api.listFieldDefinitions` etc. (Task 10), `AuthorizedRoute` (existing), `useQuery`/`useMutation` (existing).
- Produces: route `/admin/custom-fields`; sidebar entry visible only with `field:manage`.

- [ ] **Step 1: Write failing tests**

`web/src/console/pages/CustomFieldsPage.test.tsx` — cover: table renders definitions; create dialog posts and refreshes; locked inputs disabled after data exists (use a definition with `version > 1`… in practice "locked" is driven by backend 409; assert the error surfaces via toast/error text); retire flow shows confirmation and calls the API.

Minimal starting test:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CustomFieldsPage } from "./CustomFieldsPage";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    listFieldDefinitions: vi.fn(),
    createFieldDefinition: vi.fn(),
    retireFieldDefinition: vi.fn(),
    fieldDefinitionHistory: vi.fn()
  }
}));

test("renders definitions in a table", async () => {
  vi.mocked(api.listFieldDefinitions).mockResolvedValue({
    items: [
      { id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number", field_type: "TEXT", config: { required: true }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  render(<CustomFieldsPage />);
  await waitFor(() => expect(screen.getByText("PO Number")).toBeInTheDocument());
  expect(screen.getByText("TEXT")).toBeInTheDocument();
});

test("create dialog posts a new field", async () => {
  vi.mocked(api.listFieldDefinitions).mockResolvedValue({ items: [] });
  vi.mocked(api.createFieldDefinition).mockResolvedValue({
    id: "def-2", entity_type: "incident", key: "zone", label: "Zone", field_type: "SELECT", config: { options: [{ label: "A", value: "a" }] }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: ""
  });
  const user = userEvent.setup();
  render(<CustomFieldsPage />);
  await user.click(await screen.findByRole("button", { name: /new field/i }));
  await user.type(await screen.findByLabelText(/label/i), "Zone");
  await user.click(screen.getByRole("button", { name: /create/i }));
  await waitFor(() => expect(api.createFieldDefinition).toHaveBeenCalled());
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npm test -- CustomFieldsPage`
Expected: FAIL (page missing).

- [ ] **Step 3: Implement the page**

`web/src/console/pages/CustomFieldsPage.tsx` — follow the existing page conventions (check `AssetsPage.tsx` or `AccountPage.tsx` for layout, `useQuery`, error/empty states). Core structure:

```tsx
import { useState } from "react";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { useQuery } from "../useQuery";
import { useMutation } from "../useMutation";
import { PermissionGate } from "../ui/PermissionGate";
import { ErrorState } from "../ui/ErrorState";
import { EmptyState } from "../ui/EmptyState";
import { DataTable } from "../ui/DataTable";
import type { CustomFieldDefinition, CustomFieldType } from "../../types";
import { CreateFieldDialog } from "../components/CreateFieldDialog";

export function CustomFieldsPage() {
  const { t } = useI18n();
  const fields = useQuery(() => api.listFieldDefinitions("incident"), []);
  const [creating, setCreating] = useState(false);
  const retire = useMutation((id: string) => api.retireFieldDefinition(id));

  if (fields.loading) return <div className="page-loading" aria-busy="true" />;
  if (fields.error) return <ErrorState message={fields.error} onRetry={() => void fields.refetch()} />;

  return (
    <PermissionGate permission="field:manage">
      <div className="page">
        <header className="page-header">
          <h1>{t("admin.customFields.title")}</h1>
          <p>{t("admin.customFields.subtitle")}</p>
          <button className="btn-primary" onClick={() => setCreating(true)}>
            {t("admin.customFields.newField")}
          </button>
        </header>
        {fields.data && fields.data.items.length === 0 ? (
          <EmptyState title={t("admin.customFields.empty")} />
        ) : (
          <DataTable columns={[...]}>
            {fields.data?.items.map((field) => (
              <FieldRow
                key={field.id}
                field={field}
                onRetire={async () => {
                  await retire.run(field.id);
                  void fields.refetch();
                }}
              />
            ))}
          </DataTable>
        )}
        {creating && (
          <CreateFieldDialog
            onClose={() => setCreating(false)}
            onCreated={() => {
              setCreating(false);
              void fields.refetch();
            }}
          />
        )}
      </div>
    </PermissionGate>
  );
}
```

Create `web/src/console/components/CreateFieldDialog.tsx` (form: label, key auto-suggested, type select, per-type config — options editor for SELECT/MULTI_SELECT, required checkbox, sort order; live preview using `CustomFieldControl`; on submit calls `api.createFieldDefinition` then `onCreated`; shows validation errors inline; disables the type/config inputs when the field already holds data — pass `locked={used}` if you track it, otherwise rely on backend 409 surfaced in the error area).

Also create `web/src/console/components/FieldRow.tsx` or inline row markup showing: label, key, type badge, required, sort order, status; Edit (opens dialog, locked inputs disabled), Retire (confirmation dialog: "N incidents hold values" — derive count from a `history`-style API only if added; otherwise generic confirm text), History (drawer fetching `api.fieldDefinitionHistory`).

Keep it consistent with the existing dialog patterns (`web/src/console/feedback/Dialog.tsx`, `web/src/console/ui/DataTable.tsx`).

- [ ] **Step 4: Wire the route and nav**

In `web/src/App.tsx`, inside `<ConsoleLayout>`:

```tsx
                <Route
                  path="/admin/custom-fields"
                  element={
                    <AuthorizedRoute permission="field:manage">
                      <CustomFieldsPage />
                    </AuthorizedRoute>
                  }
                />
```

In `web/src/console/layout/Sidebar.tsx`, add to `CROSS_LINKS`:

```tsx
  { to: "/admin/custom-fields", key: "nav.customFields", icon: SlidersHorizontal, permission: "field:manage" },
```

(Import the icon from `@phosphor-icons/react` — verify it exists in the dependency set; otherwise reuse an existing icon like `GearSix`.)

- [ ] **Step 5: Add i18n keys**

In `web/src/i18n/messages.ts` (both `en` and `vi` blocks): `nav.customFields`, `admin.customFields.title`, `admin.customFields.subtitle`, `admin.customFields.newField`, `admin.customFields.empty`, `admin.customFields.edit`, `admin.customFields.retire`, `admin.customFields.history`, `admin.customFields.lockedHint`, `admin.customFields.retireConfirm`, `admin.customFields.type`, `admin.customFields.key`, `admin.customFields.label`, `admin.customFields.required`, `admin.customFields.options`, `admin.customFields.addOption`, `admin.customFields.sortOrder`, `admin.customFields.preview`.

- [ ] **Step 6: Run web tests + lint + build**

Run: `cd web && npm test && npm run lint && npm run build`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src/App.tsx web/src/console/layout/Sidebar.tsx web/src/console/pages/CustomFieldsPage.tsx web/src/console/pages/CustomFieldsPage.test.tsx web/src/console/components/CreateFieldDialog.tsx web/src/i18n/messages.ts
git commit -m "feat: add admin custom fields page and navigation"
```

---

### Task 13: Detail page, queue display, and filter chips

**Files:**
- Modify: `web/src/console/pages/IncidentDetailPage.tsx`
- Modify: `web/src/console/pages/IncidentsPage.tsx`
- Modify: `web/src/console/pages/IncidentDetailPage.test.tsx` / `IncidentsPage` tests
- Modify: `web/src/i18n/messages.ts`

**Interfaces:**
- Consumes: `IncidentListResponse` + `custom_values` (Task 10), `CustomFieldControl` conventions (Task 11).
- Produces: detail page "Custom fields" card; queue row disclosure showing custom values; saved-view filter chips for active custom fields.

- [ ] **Step 1: Detail page — custom fields card**

In `web/src/console/pages/IncidentDetailPage.tsx`:
1. The detail fetch must load definitions. If the detail endpoint now returns `custom_fields`, extend the fetch type (or add a second `useQuery` for `api.listFieldDefinitions`). Prefer the endpoint's `custom_fields` array.
2. Render a card after the existing metadata rail:

```tsx
{customFields.length > 0 && (
  <section className="detail-card" aria-label={t("incident.customFields")}>
    <h2>{t("incident.customFields")}</h2>
    <dl className="detail-list">
      {customFields.map((field) => {
        const value = incident.custom_values?.[field.id];
        if (value === undefined || value === null || value === "") return null;
        return (
          <div key={field.id} className="detail-row">
            <dt>{field.label}</dt>
            <dd>{formatCustomValue(field, value)}</dd>
          </div>
        );
      })}
    </dl>
  </section>
)}
```

Add a small `formatCustomValue(field, value)` helper: for MULTI_SELECT map option values → labels; DATE show as-is; others stringify. Put it in `web/src/console/components/customFieldFormat.ts` with a unit test `customFieldFormat.test.ts`.

- [ ] **Step 2: Queue — disclosure row**

In `web/src/console/pages/IncidentsPage.tsx`:
1. The list fetch uses `api.incidents(options)`; the response now includes `custom_fields`. Store it alongside items (the `usePaginatedList` hook — check how it feeds `data.items`; keep a parallel state for `customFields` from the first page response).
2. Add a chevron per row that expands to show custom values:

```tsx
<tr className="incident-row" onClick={() => toggleExpanded(incident.id)}>
  ...
</tr>
{expanded === incident.id && (
  <tr className="incident-row-detail">
    <td colSpan={COLUMN_COUNT}>
      <dl className="custom-fields-inline">
        {activeFields(incident).map(([field, value]) => (
          <div key={field.id} className="inline-value">
            <dt>{field.label}</dt>
            <dd>{formatCustomValue(field, value)}</dd>
          </div>
        ))}
      </dl>
    </td>
  </tr>
)}
```

where `activeFields(incident)` filters `customFields` to those present in `incident.custom_values`.

- [ ] **Step 3: Filter chips**

In the saved-view/filter chip area of `IncidentsPage.tsx`, when `customFields` are loaded, render a filter row for ACTIVE SELECT/TEXT fields:

```tsx
{customFields.filter((f) => f.field_type === "SELECT" || f.field_type === "TEXT").map((field) => (
  <select
    key={field.id}
    aria-label={field.label}
    value={customFieldFilters[field.key] ?? ""}
    onChange={(event) => {
      const next = { ...customFieldFilters };
      if (event.target.value === "") delete next[field.key];
      else next[field.key] = event.target.value;
      setCustomFieldFilters(next);
    }}
  >
    <option value="">{field.label}</option>
    {field.field_type === "SELECT"
      ? field.config.options?.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))
      : null}
  </select>
))}
```

Pass `custom_fields: customFieldFilters` into the `api.incidents` options (already supported from Task 10).

- [ ] **Step 4: Update tests**

Extend the existing page tests: mock `custom_fields` in list responses; assert the disclosure row renders a custom value; assert selecting a filter chip appends `custom_field.<key>` to the request.

- [ ] **Step 5: Run web tests + lint + build**

Run: `cd web && npm test && npm run lint && npm run build`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/console/pages/IncidentDetailPage.tsx web/src/console/pages/IncidentsPage.tsx web/src/console/components/customFieldFormat.ts web/src/console/components/customFieldFormat.test.ts web/src/i18n/messages.ts
git commit -m "feat: surface custom field values in incident detail and queue"
```

---

### Task 14: Changelog and full verification

**Files:**
- Modify: `ChangeLogs.md`

- [ ] **Step 1: Add a changelog entry**

Add at the top of `ChangeLogs.md` under `## [Unreleased]`:

```markdown
### 2026-08-09 — Custom fields core (tenant-configurable incident fields)

- Tenant administrators can now define custom fields on incidents
  (TEXT/NUMBER/DATE/SELECT/MULTI_SELECT) through a new `/admin/custom-fields`
  page, gated by a new Administrator-only `field:manage` permission.
- Field definitions are org-scoped; keys are immutable; once an incident holds
  a value the field type and config lock and the field can only be retired,
  keeping all values and history.
- Incident create form renders the tenant's active fields dynamically; values
  are validated server-side (type, required, length, regex, range, options),
  stored in a JSONB `custom_values` column with a full audit history, and
  surfaced on the detail page and queue rows.
- Incident list supports `custom_field.<key>` filter parameters, joined with
  existing saved-view filters.
```

- [ ] **Step 2: Full backend verification**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: PASS. If `TEST_DATABASE_URL` is available, also run `go test -race -count=1 ./internal/customfield/... ./internal/incident/...`.

- [ ] **Step 3: Full web verification**

Run: `cd web && npm run lint && npm test && npm run build`
Expected: PASS.

- [ ] **Step 4: Smoke test the flow**

1. `make compose-up && make migrate-up` (or the local dev flow from `docs/runbooks/local-development.md`).
2. `make seed` (existing demo data — ensure it still seeds with the new column defaulting to `{}`).
3. `go run ./cmd/api` and `cd web && npm run dev`.
4. As an Administrator: create a TEXT field `po_number` required and a SELECT field `zone`.
5. Create an incident via the form — custom fields render, required enforced; the detail page and queue show values.
6. Try `PATCH` changing the field type after data exists → expect 409.
7. Retire the SELECT field → disappears from the form, values remain visible.

- [ ] **Step 5: Commit**

```bash
git add ChangeLogs.md
git commit -m "docs: record custom fields core in the changelog"
```

---

## Self-Review Notes

- **Spec coverage:** migration (T1), domain validation (T2), service + immutability (T3), persistence + history (T4), permission (T5), admin API (T6), incident values/history/filter (T7), wiring (T8), OpenAPI (T9), web types/client (T10), form renderer (T11), admin UI (T12), detail/queue/filters (T13), changelog + verification (T14). The spec's "retire keeps history" and "field:manage Administrator-only" are covered in T3/T5. NUMBER range filters are deliberately reduced to exact-match in stage 1 (the `->>` text comparison); range syntax is deferred.
- **Type consistency:** `customfieldapp.Definition`/`CreateDefinition`/`UpdateDefinition` names are stable across T3–T9. `incidentapp.Fields` interface names match between T7 and T8 (`Definitions`, `ResolveAndValidate`, `ResolveKeys`). Web `CustomFieldDefinition`/`IncidentListResponse`/`HistoryEntry` names match between T10–T13. `api.updateFieldDefinition` uses a `patch` helper added in T10 and consumed in T12.
