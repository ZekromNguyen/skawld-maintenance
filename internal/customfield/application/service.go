package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrForbidden  = errors.New("field operation forbidden")
	ErrNotFound   = errors.New("field definition not found")
	ErrInvalid    = errors.New("invalid field definition")
	ErrConflict   = errors.New("field definition conflict")
	ErrValidation = errors.New("custom field value validation failed")
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
	ID          string        `json:"id"`
	EntityType  string        `json:"entity_type"`
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Description string        `json:"description,omitempty"`
	FieldType   string        `json:"field_type"`
	Config      domain.Config `json:"config"`
	Status      string        `json:"status"`
	SortOrder   int           `json:"sort_order"`
	Version     int64         `json:"version"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	RetiredAt   *time.Time    `json:"retired_at,omitempty"`
}

type HistoryEntry struct {
	IncidentID  string    `json:"incident_id"`
	PrincipalID string    `json:"principal_id"`
	ValueBefore any       `json:"value_before,omitempty"`
	ValueAfter  any       `json:"value_after"`
	ChangedAt   time.Time `json:"changed_at"`
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
	return s.list(ctx, principal.OrganizationID, entityType)
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
		CreatedAt: current.CreatedAt, RetiredAt: current.RetiredAt,
	})
	if err != nil {
		return Definition{}, errors.Join(ErrInvalid, err)
	}
	// domain.New normalizes Version to 1 (creation semantics); restore the
	// optimistic-lock bump for updates.
	value.Version = current.Version + 1
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

// Definitions lists the org's field definitions for an entity without a
// permission check; it is the internal helper used to render incident
// responses with their custom_fields section.
func (s Service) Definitions(ctx context.Context, organizationID, entityType string) ([]Definition, error) {
	return s.list(ctx, organizationID, entityType)
}

func (s Service) list(ctx context.Context, organizationID, entityType string) ([]Definition, error) {
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
