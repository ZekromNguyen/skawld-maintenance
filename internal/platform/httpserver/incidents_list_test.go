package httpserver

import (
	"context"
	"encoding/json"
	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
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

func (*listIncidentStore) Close(
	context.Context, identitydomain.Principal, string, string, incidentapp.CloseIncident,
) (incidentapp.Incident, bool, error) {
	return incidentapp.Incident{}, false, nil
}
func (*listIncidentStore) Reopen(
	context.Context, identitydomain.Principal, string, string, incidentapp.ReopenIncident,
) (incidentapp.Incident, bool, error) {
	return incidentapp.Incident{}, false, nil
}

type listAttachmentStore struct{}

func (*listAttachmentStore) Create(
	context.Context, identitydomain.Principal, string, attachmentapp.CreateManifest,
) (attachmentapp.Attachment, bool, error) {
	return attachmentapp.Attachment{}, false, nil
}
func (*listAttachmentStore) Complete(
	context.Context, identitydomain.Principal, string, string, attachmentapp.CompleteUpload,
) (attachmentapp.Attachment, bool, error) {
	return attachmentapp.Attachment{}, false, nil
}
func (*listAttachmentStore) ListByEntity(
	context.Context, identitydomain.Principal, string, string,
) ([]attachmentapp.Attachment, error) {
	return []attachmentapp.Attachment{{
		ID:               "00000000-0000-0000-0000-00000000000a",
		OriginalFilename: "clip.mp4",
		DeclaredMIME:     "video/mp4",
		State:            "AVAILABLE",
	}}, nil
}

func TestGetIncidentIncludesAttachments(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead:    {},
			identitydomain.PermissionAttachmentWrite: {},
		},
	}
	handler := New(Dependencies{
		Logger:      slog.New(slog.DiscardHandler),
		Auth:        fakeAuth{principal: principal},
		Incidents:   incidentapp.Service{Store: &listIncidentStore{}},
		Attachments: attachmentapp.Service{Store: &listAttachmentStore{}},
	})
	request := httptest.NewRequest(http.MethodGet,
		"/api/v1/incidents/00000000-0000-0000-0000-00000000000c", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		ID          string                     `json:"id"`
		Attachments []attachmentapp.Attachment `json:"attachments"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Attachments) != 1 || body.Attachments[0].OriginalFilename != "clip.mp4" {
		t.Fatalf("unexpected attachments: %#v", body.Attachments)
	}
}
