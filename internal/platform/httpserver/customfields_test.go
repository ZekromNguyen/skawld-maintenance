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
		out = append(out, domain.Definition{
			ID: d.ID, OrganizationID: "o1", EntityType: d.EntityType,
			Key: d.Key, Label: d.Label, FieldType: domain.FieldType(d.FieldType),
			Status: domain.Status(d.Status), Version: d.Version,
		})
	}
	return out, nil
}

func (s *stubFieldStore) Get(_ context.Context, _, id string) (domain.Definition, error) {
	d, ok := s.definitions[id]
	if !ok {
		return domain.Definition{}, application.ErrNotFound
	}
	return domain.Definition{
		ID: d.ID, OrganizationID: "o1", EntityType: d.EntityType,
		Key: d.Key, Label: d.Label, FieldType: domain.FieldType(d.FieldType),
		Status: domain.Status(d.Status), Version: d.Version,
	}, nil
}

func (s *stubFieldStore) Create(_ context.Context, _ string, d domain.Definition, now time.Time) (domain.Definition, error) {
	d.ID = "def-1"
	d.CreatedAt = now
	d.UpdatedAt = now
	s.definitions[d.ID] = application.Definition{
		ID: d.ID, EntityType: d.EntityType, Key: d.Key, Label: d.Label,
		FieldType: string(d.FieldType), Status: string(d.Status),
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	return d, nil
}

func (s *stubFieldStore) Update(_ context.Context, _, id string, d domain.Definition, _ time.Time) (domain.Definition, error) {
	d.ID = id
	s.definitions[id] = application.Definition{
		ID: id, EntityType: d.EntityType, Key: d.Key, Label: d.Label,
		FieldType: string(d.FieldType), Status: string(d.Status), Version: d.Version,
	}
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

func (s *stubFieldStore) HasValues(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func withFieldPrincipal(r *http.Request, principal identitydomain.Principal) *http.Request {
	return r.WithContext(identitydomain.WithPrincipal(r.Context(), principal))
}

// withFieldParam injects a chi route context carrying the named URL param,
// mirroring how chi.URLParam resolves it when the handler is invoked directly.
func withFieldParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func fieldAdmin() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
			identitydomain.PermissionFieldManage:  {},
		},
	}
}

func fieldReader() identitydomain.Principal {
	return identitydomain.Principal{
		ID: "p1", OrganizationID: "o1",
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionIncidentRead: {}},
	}
}

func fieldService() application.Service {
	return application.Service{
		Store: &stubFieldStore{definitions: map[string]application.Definition{}},
		Now:   func() time.Time { return time.Unix(0, 0).UTC() },
	}
}

func TestCreateFieldDefinitionHandler(t *testing.T) {
	handler := createFieldDefinition(fieldService())
	body := `{"entity_type":"incident","key":"po_number","label":"PO Number","field_type":"TEXT","config":{"required":true},"sort_order":1}`
	req := withFieldPrincipal(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions", strings.NewReader(body)), fieldAdmin())
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

func TestCreateFieldDefinitionValidation400(t *testing.T) {
	handler := createFieldDefinition(fieldService())
	body := `{"entity_type":"incident","key":"PO_BAD","label":"Bad","field_type":"TEXT"}`
	req := withFieldPrincipal(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions", strings.NewReader(body)), fieldAdmin())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid definition, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateFieldDefinitionRequiresManagePermission(t *testing.T) {
	handler := createFieldDefinition(fieldService())
	body := `{"entity_type":"incident","key":"po","label":"PO","field_type":"TEXT"}`
	req := withFieldPrincipal(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions", strings.NewReader(body)), fieldReader())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListFieldDefinitionsHandler(t *testing.T) {
	service := fieldService()
	store := service.Store.(*stubFieldStore)
	store.definitions["11111111-1111-4111-8111-111111111111"] = application.Definition{
		ID: "def-1", EntityType: "incident", Key: "po_number", Label: "PO Number",
		FieldType: "TEXT", Status: "ACTIVE", Version: 1,
	}
	handler := listFieldDefinitions(service)
	req := withFieldPrincipal(httptest.NewRequest(http.MethodGet, "/api/v1/admin/field-definitions?entity_type=incident", nil), fieldAdmin())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []application.Definition `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Key != "po_number" {
		t.Fatalf("unexpected list response: %+v", response.Items)
	}
}

func TestRetireFieldDefinitionHandlerConflict(t *testing.T) {
	service := fieldService()
	store := service.Store.(*stubFieldStore)
	now := time.Unix(0, 0).UTC()
	store.definitions["11111111-1111-4111-8111-111111111111"] = application.Definition{
		ID: "11111111-1111-4111-8111-111111111111", EntityType: "incident", Key: "po", Label: "PO",
		FieldType: "TEXT", Status: "RETIRED", Version: 2, RetiredAt: &now,
	}
	handler := retireFieldDefinition(service)
	req := withFieldParam(
		withFieldPrincipal(httptest.NewRequest(http.MethodPost, "/api/v1/admin/field-definitions/11111111-1111-4111-8111-111111111111/retire", nil), fieldAdmin()),
		"fieldID", "11111111-1111-4111-8111-111111111111",
	)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for double retire, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWriteFieldErrorValidation422(t *testing.T) {
	rec := httptest.NewRecorder()
	writeFieldError(rec, application.ErrValidation)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for ErrValidation, got %d", rec.Code)
	}
}

func TestWriteFieldErrorForbidden403(t *testing.T) {
	rec := httptest.NewRecorder()
	writeFieldError(rec, application.ErrForbidden)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ErrForbidden, got %d", rec.Code)
	}
}
