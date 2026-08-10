package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type listExecutionStore struct{}

func (*listExecutionStore) Create(
	context.Context, identitydomain.Principal, string, executionapp.CreateExecution,
) (executionapp.Execution, bool, error) {
	return executionapp.Execution{}, false, nil
}
func (*listExecutionStore) Get(context.Context, identitydomain.Principal, string) (executionapp.Execution, error) {
	return executionapp.Execution{}, nil
}
func (*listExecutionStore) List(context.Context, identitydomain.Principal, executionapp.Filter) ([]executionapp.Execution, bool, error) {
	return []executionapp.Execution{{
		ID:        "00000000-0000-0000-0000-00000000000d",
		UpdatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listExecutionStore) Start(
	context.Context, identitydomain.Principal, string, string, executionapp.StartExecution,
) (executionapp.Execution, bool, error) {
	return executionapp.Execution{}, false, nil
}
func (*listExecutionStore) VerifyPrerequisite(
	context.Context, identitydomain.Principal, string, string, executionapp.VerifyPrerequisite,
) (executionapp.Prerequisite, bool, error) {
	return executionapp.Prerequisite{}, false, nil
}
func (*listExecutionStore) CompleteStep(
	context.Context, identitydomain.Principal, string, string, string, executionapp.CompleteStep,
) (executionapp.Step, bool, error) {
	return executionapp.Step{}, false, nil
}
func (*listExecutionStore) RecordMeasurement(
	context.Context, identitydomain.Principal, string, string, executionapp.RecordMeasurement,
) (executionapp.Measurement, bool, error) {
	return executionapp.Measurement{}, false, nil
}
func (*listExecutionStore) RecordObservation(
	context.Context, identitydomain.Principal, string, string, executionapp.RecordObservation,
) (executionapp.Observation, bool, error) {
	return executionapp.Observation{}, false, nil
}
func (*listExecutionStore) RecordAction(
	context.Context, identitydomain.Principal, string, string, executionapp.RecordAction,
) (executionapp.Action, bool, error) {
	return executionapp.Action{}, false, nil
}
func (*listExecutionStore) RecordDecision(
	context.Context, identitydomain.Principal, string, string, executionapp.RecordDecision,
) (executionapp.Decision, bool, error) {
	return executionapp.Decision{}, false, nil
}
func (*listExecutionStore) Complete(
	context.Context, identitydomain.Principal, string, string, executionapp.CompleteExecution,
) (executionapp.Execution, bool, error) {
	return executionapp.Execution{}, false, nil
}

func TestListExecutionsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExecutionRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Executions: executionapp.Service{
			Store: &listExecutionStore{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/executions?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []executionapp.Execution `json:"items"`
		NextCursor *string                  `json:"next_cursor"`
		HasMore    bool                     `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}

func TestListExecutionsRejectsInvalidPageSize(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExecutionRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Executions: executionapp.Service{Store: &listExecutionStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/executions?page_size=0", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListExecutionsRejectsMalformedCursor(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionExecutionRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Executions: executionapp.Service{Store: &listExecutionStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/executions?cursor=bm90LWEtY3Vyc29y", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestListExecutionsForbiddenWithoutPermission(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentRead: {},
		},
	}
	handler := New(Dependencies{
		Logger:     slog.New(slog.DiscardHandler),
		Auth:       fakeAuth{principal: principal},
		Executions: executionapp.Service{Store: &listExecutionStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}
