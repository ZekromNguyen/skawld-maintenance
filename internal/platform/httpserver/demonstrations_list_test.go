package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type listDemonstrationGateway struct{}

func (*listDemonstrationGateway) Start(
	context.Context, identitydomain.Principal, demonstrationapp.Start,
) (demonstrationapp.Demonstration, error) {
	return demonstrationapp.Demonstration{}, nil
}
func (*listDemonstrationGateway) Get(
	context.Context, identitydomain.Principal, string,
) (demonstrationapp.Demonstration, error) {
	return demonstrationapp.Demonstration{}, nil
}
func (*listDemonstrationGateway) List(
	context.Context, identitydomain.Principal, string, demonstrationapp.ListFilter,
) ([]demonstrationapp.Demonstration, bool, error) {
	return []demonstrationapp.Demonstration{{
		ID:        "00000000-0000-0000-0000-00000000000f",
		StartedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listDemonstrationGateway) Complete(
	context.Context, identitydomain.Principal, string, demonstrationapp.Complete,
) (demonstrationapp.Demonstration, error) {
	return demonstrationapp.Demonstration{}, nil
}
func (*listDemonstrationGateway) RecordEvidenceView(
	context.Context, identitydomain.Principal, string, demonstrationapp.RecordEvidenceView,
) (demonstrationapp.Event, error) {
	return demonstrationapp.Event{}, nil
}
func (*listDemonstrationGateway) Review(
	context.Context, identitydomain.Principal, string, demonstrationapp.Review,
) (demonstrationapp.ReviewRecord, error) {
	return demonstrationapp.ReviewRecord{}, nil
}
func (*listDemonstrationGateway) RedactEvent(
	context.Context, identitydomain.Principal, string, string, demonstrationapp.RedactEvent,
) (demonstrationapp.Redaction, error) {
	return demonstrationapp.Redaction{}, nil
}

func TestListDemonstrationsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Demonstrations: demonstrationapp.Service{
			Gateway: &listDemonstrationGateway{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/demonstrations?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []demonstrationapp.Demonstration `json:"items"`
		NextCursor *string                          `json:"next_cursor"`
		HasMore    bool                             `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}

func TestListDemonstrationsRejectsInvalidPageSize(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:         slog.New(slog.DiscardHandler),
		Auth:           fakeAuth{principal: principal},
		Demonstrations: demonstrationapp.Service{Gateway: &listDemonstrationGateway{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/demonstrations?page_size=0", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListDemonstrationsRejectsMalformedCursor(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionDemonstrationRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:         slog.New(slog.DiscardHandler),
		Auth:           fakeAuth{principal: principal},
		Demonstrations: demonstrationapp.Service{Gateway: &listDemonstrationGateway{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/demonstrations?cursor=bm90LWEtY3Vyc29y", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListDemonstrationsForbiddenWithoutPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:         slog.New(slog.DiscardHandler),
		Auth:           fakeAuth{principal: principal},
		Demonstrations: demonstrationapp.Service{Gateway: &listDemonstrationGateway{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/demonstrations", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}
