package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
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

func (f *fakeStore) History(_ context.Context, _, _ string) ([]HistoryEntry, error) {
	return f.history, nil
}

func (f *fakeStore) HasValues(_ context.Context, _, _ string) (bool, error) {
	return f.hasValues, nil
}

func (f *fakeStore) Usage(_ context.Context, _, _ string) (map[string]bool, error) {
	out := map[string]bool{}
	for id, d := range f.definitions {
		out[id] = d.Status == domain.StatusRetired
	}
	return out, nil
}

func admin() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
			identitydomain.PermissionFieldManage:  {},
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
	created, err := s.Create(context.Background(), admin(), CreateDefinition{
		EntityType: "incident", Key: "po_number", Label: "PO Number",
		FieldType: "TEXT", Config: domain.Config{Required: true},
	})
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
	if _, err := s.ResolveAndValidate(context.Background(), "o1", "incident", map[string]any{"po_number": ""}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid required value must be ErrValidation, got %v", err)
	}
}

func TestUpdateLocksConfigAfterUse(t *testing.T) {
	store := &fakeStore{
		definitions: map[string]domain.Definition{},
		byKey:       map[string]domain.Definition{},
		hasValues:   true,
	}
	d := domain.Definition{
		ID: "def-1", OrganizationID: "o1", EntityType: "incident", Key: "po",
		Label: "PO", FieldType: domain.FieldTypeText, Status: domain.StatusActive, Version: 1,
	}
	store.definitions["def-1"] = d
	s := newService(store)
	_, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{
		FieldType: "NUMBER", Config: domain.Config{}, SortOrder: 1, ExpectedVersion: 1,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("changing type on a used field must be ErrConflict, got %v", err)
	}
	updated, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{
		Label: "Purchase Order", FieldType: "TEXT", Config: domain.Config{}, SortOrder: 5, ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("label change must pass: %v", err)
	}
	if updated.Label != "Purchase Order" {
		t.Fatalf("label not updated: %+v", updated)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version bump to 2, got %d", updated.Version)
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

func TestResolveKeys(t *testing.T) {
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	store.definitions["def-1"] = domain.Definition{ID: "def-1", Key: "po_number", EntityType: "incident", Status: domain.StatusActive, Label: "PO"}
	s := newService(store)
	resolved, err := s.ResolveKeys(context.Background(), "o1", "incident", []string{"po_number"})
	if err != nil {
		t.Fatalf("resolve keys: %v", err)
	}
	if resolved["po_number"].ID != "def-1" {
		t.Fatalf("expected def-1 for po_number, got %+v", resolved)
	}
	if _, err := s.ResolveKeys(context.Background(), "o1", "incident", []string{"nope"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown key must be ErrValidation, got %v", err)
	}
}

func TestUpdateVersionMismatchConflicts(t *testing.T) {
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	store.definitions["def-1"] = domain.Definition{
		ID: "def-1", OrganizationID: "o1", EntityType: "incident", Key: "po",
		Label: "PO", FieldType: domain.FieldTypeText, Status: domain.StatusActive, Version: 3,
	}
	s := newService(store)
	_, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{
		Label: "New", FieldType: "TEXT", Config: domain.Config{}, SortOrder: 1, ExpectedVersion: 2,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected version mismatch must be ErrConflict, got %v", err)
	}
}

func TestUpdatePreservesRetiredAt(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	store.definitions["def-1"] = domain.Definition{
		ID: "def-1", OrganizationID: "o1", EntityType: "incident", Key: "po",
		Label: "PO", FieldType: domain.FieldTypeText, Status: domain.StatusRetired,
		Version: 2, RetiredAt: &now,
	}
	s := newService(store)
	updated, err := s.Update(context.Background(), admin(), "def-1", UpdateDefinition{
		Label: "PO (legacy)", FieldType: "TEXT", Config: domain.Config{}, SortOrder: 1, ExpectedVersion: 2,
	})
	if err != nil {
		t.Fatalf("label change on retired field must pass: %v", err)
	}
	if updated.RetiredAt == nil || !updated.RetiredAt.Equal(now) {
		t.Fatalf("update must preserve retired_at, got %+v", updated.RetiredAt)
	}
}

func TestResolveAndValidateRejectsRetiredField(t *testing.T) {
	store := &fakeStore{definitions: map[string]domain.Definition{}, byKey: map[string]domain.Definition{}}
	store.definitions["def-1"] = domain.Definition{
		ID: "def-1", Key: "po_number", EntityType: "incident", Label: "PO",
		FieldType: domain.FieldTypeText, Status: domain.StatusRetired,
	}
	s := newService(store)
	if _, err := s.ResolveAndValidate(context.Background(), "o1", "incident", map[string]any{"po_number": "x"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("value for retired field must be ErrValidation, got %v", err)
	}
}
