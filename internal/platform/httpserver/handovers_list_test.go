package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type listHandoverStore struct{}

func (*listHandoverStore) LoadWindowContext(context.Context, identitydomain.Principal, handoverapp.PrepareDraft) (handoverapp.WindowContext, error) {
	return handoverapp.WindowContext{}, nil
}
func (*listHandoverStore) SaveDraft(context.Context, identitydomain.Principal, string, handoverapp.Handover) (handoverapp.Handover, bool, error) {
	return handoverapp.Handover{}, false, nil
}
func (*listHandoverStore) Get(context.Context, identitydomain.Principal, string) (handoverapp.Handover, error) {
	return handoverapp.Handover{}, nil
}
func (*listHandoverStore) Edit(context.Context, identitydomain.Principal, string, string, handoverapp.Edit) (handoverapp.Handover, bool, error) {
	return handoverapp.Handover{}, false, nil
}
func (*listHandoverStore) Submit(context.Context, identitydomain.Principal, string, string, handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return handoverapp.Handover{}, false, nil
}
func (*listHandoverStore) Accept(context.Context, identitydomain.Principal, string, string, handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return handoverapp.Handover{}, false, nil
}
func (*listHandoverStore) Acknowledge(context.Context, identitydomain.Principal, string, string, handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return handoverapp.Handover{}, false, nil
}
func (*listHandoverStore) RecordCall(context.Context, identitydomain.Principal, handoverapp.WindowContext, skawld.Generation, string, string) error {
	return nil
}
func (*listHandoverStore) List(context.Context, identitydomain.Principal, handoverapp.HandoverFilter) ([]handoverapp.Handover, bool, error) {
	return []handoverapp.Handover{{
		ID: "00000000-0000-0000-0000-00000000000a", State: "DRAFT",
		ShiftStart: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

const listHandoverSiteID = "11111111-1111-4111-8111-111111111111"

func listHandoversHandler() http.Handler {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{listHandoverSiteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionHandoverWrite: {},
		},
	}
	return New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Handovers: handoverapp.Service{
			Store: &listHandoverStore{},
		},
	})
}

func TestListHandoversEnvelope(t *testing.T) {
	handler := listHandoversHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/handovers?site_id="+listHandoverSiteID+"&page_size=25", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Code, response.Body.String())
	}
	var body struct {
		Items      []handoverapp.Handover `json:"items"`
		NextCursor *string                `json:"next_cursor"`
		HasMore    bool                   `json:"has_more"`
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

func TestListHandoversValidatesPageSize(t *testing.T) {
	handler := listHandoversHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/handovers?page_size=0", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestListHandoversRejectsBadSite(t *testing.T) {
	handler := listHandoversHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/handovers?site_id=not-a-uuid", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
