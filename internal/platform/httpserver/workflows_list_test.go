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
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
)

type listWorkflowGateway struct{}

func (*listWorkflowGateway) Compile(
	context.Context, identitydomain.Principal, workflowapp.Compile,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) Get(
	context.Context, identitydomain.Principal, string, int,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) List(
	context.Context, identitydomain.Principal, workflowapp.ListFilter,
) ([]workflowapp.Version, bool, error) {
	return []workflowapp.Version{{
		WorkflowID: "00000000-0000-0000-0000-000000000010",
		Version:    1,
		CreatedAt:  time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listWorkflowGateway) Review(
	context.Context, identitydomain.Principal, string, int, workflowapp.Review,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) Publish(
	context.Context, identitydomain.Principal, string, int, workflowapp.Publish,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) ExpandApplicability(
	context.Context, identitydomain.Principal, string, int, workflowapp.ExpandApplicability,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) Retire(
	context.Context, identitydomain.Principal, string, int, workflowapp.Retire,
) (workflowapp.Version, error) {
	return workflowapp.Version{}, nil
}
func (*listWorkflowGateway) Applicable(
	context.Context, identitydomain.Principal, string,
) ([]workflowapp.Version, error) {
	return nil, nil
}

func TestListWorkflowsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionWorkflowRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Workflows: workflowapp.Service{
			Gateway: &listWorkflowGateway{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []workflowapp.Version `json:"items"`
		NextCursor *string               `json:"next_cursor"`
		HasMore    bool                  `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}
