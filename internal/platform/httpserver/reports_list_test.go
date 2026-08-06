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
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type listReportStore struct{}

func (*listReportStore) LoadExecutionContext(context.Context, identitydomain.Principal, string) (reportapp.ExecutionContext, error) {
	return reportapp.ExecutionContext{}, nil
}
func (*listReportStore) SaveDraft(context.Context, identitydomain.Principal, string, reportapp.Report, json.RawMessage) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Get(context.Context, identitydomain.Principal, string) (reportapp.Report, error) {
	return reportapp.Report{}, nil
}
func (*listReportStore) Edit(context.Context, identitydomain.Principal, string, string, reportapp.Edit) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Submit(context.Context, identitydomain.Principal, string, string, reportapp.Transition) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) Approve(context.Context, identitydomain.Principal, string, string, reportapp.Transition) (reportapp.Report, bool, error) {
	return reportapp.Report{}, false, nil
}
func (*listReportStore) RecordCall(context.Context, identitydomain.Principal, reportapp.ExecutionContext, skawld.Generation, string, string) error {
	return nil
}
func (*listReportStore) List(context.Context, identitydomain.Principal, reportapp.ReportFilter) ([]reportapp.Report, bool, error) {
	return []reportapp.Report{{
		ID: "00000000-0000-0000-0000-00000000000a", State: "DRAFT",
		CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

const listReportSiteID = "11111111-1111-4111-8111-111111111111"

func listReportsHandler() http.Handler {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{listReportSiteID},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionReportWrite: {},
		},
	}
	return New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Reports: reportapp.Service{
			Store: &listReportStore{},
		},
	})
}

func TestListReportsEnvelope(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?site_id="+listReportSiteID+"&page_size=25", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", response.Code, response.Body.String())
	}
	var body struct {
		Items      []reportapp.Report `json:"items"`
		NextCursor *string            `json:"next_cursor"`
		HasMore    bool               `json:"has_more"`
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

func TestListReportsValidatesPageSize(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?page_size=500", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestListReportsRejectsBadSite(t *testing.T) {
	handler := listReportsHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/reports?site_id=not-a-uuid", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
