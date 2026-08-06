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
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
)

type listIncidentStore struct{}

func (*listIncidentStore) Create(
	context.Context, identitydomain.Principal, string, incidentapp.CreateIncident,
) (incidentapp.Incident, bool, error) {
	return incidentapp.Incident{}, false, nil
}
func (*listIncidentStore) Get(context.Context, identitydomain.Principal, string) (incidentapp.Incident, error) {
	return incidentapp.Incident{}, nil
}
func (*listIncidentStore) List(context.Context, identitydomain.Principal, incidentapp.Filter) ([]incidentapp.Incident, bool, error) {
	return []incidentapp.Incident{{
		ID:         "00000000-0000-0000-0000-00000000000c",
		DetectedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listIncidentStore) Resolve(
	context.Context, identitydomain.Principal, string, string, incidentapp.ResolveIncident,
) (incidentapp.Incident, bool, error) {
	return incidentapp.Incident{}, false, nil
}

func TestListIncidentsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Incidents: incidentapp.Service{
			Store: &listIncidentStore{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/incidents?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []incidentapp.Incident `json:"items"`
		NextCursor *string                `json:"next_cursor"`
		HasMore    bool                   `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}

func TestListIncidentsRejectsInvalidPageSize(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:    slog.New(slog.DiscardHandler),
		Auth:      fakeAuth{principal: principal},
		Incidents: incidentapp.Service{Store: &listIncidentStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/incidents?page_size=0", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListIncidentsRejectsMalformedCursor(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:    slog.New(slog.DiscardHandler),
		Auth:      fakeAuth{principal: principal},
		Incidents: incidentapp.Service{Store: &listIncidentStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/incidents?cursor=bm90LWEtY3Vyc29y", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListIncidentsForbiddenWithoutPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:    slog.New(slog.DiscardHandler),
		Auth:      fakeAuth{principal: principal},
		Incidents: incidentapp.Service{Store: &listIncidentStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}
