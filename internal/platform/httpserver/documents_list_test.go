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
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
)

type listDocumentStore struct{}

func (*listDocumentStore) CreateDocument(
	context.Context, identitydomain.Principal, string, knowledgeapp.CreateDocument,
) (knowledgedomain.Document, bool, error) {
	return knowledgedomain.Document{}, false, nil
}
func (*listDocumentStore) CreateRevision(
	context.Context, identitydomain.Principal, string, string, knowledgeapp.CreateRevision,
) (knowledgedomain.Revision, bool, error) {
	return knowledgedomain.Revision{}, false, nil
}
func (*listDocumentStore) GetDocument(context.Context, identitydomain.Principal, string) (knowledgedomain.Document, error) {
	return knowledgedomain.Document{}, nil
}
func (*listDocumentStore) ListDocuments(context.Context, identitydomain.Principal, knowledgeapp.Filter) ([]knowledgedomain.Document, bool, error) {
	return []knowledgedomain.Document{{
		ID:        "00000000-0000-0000-0000-00000000000e",
		UpdatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listDocumentStore) ApproveRevision(
	context.Context, identitydomain.Principal, string, string, knowledgeapp.ApproveRevision,
) (knowledgedomain.Revision, bool, error) {
	return knowledgedomain.Revision{}, false, nil
}
func (*listDocumentStore) RetireRevision(
	context.Context, identitydomain.Principal, string, string, knowledgeapp.RetireRevision,
) (knowledgedomain.Revision, bool, error) {
	return knowledgedomain.Revision{}, false, nil
}
func (*listDocumentStore) RequestIngestion(
	context.Context, identitydomain.Principal, string, string, knowledgeapp.RequestIngestion,
) (knowledgedomain.Revision, bool, error) {
	return knowledgedomain.Revision{}, false, nil
}
func (*listDocumentStore) Search(
	context.Context, identitydomain.Principal, knowledgedomain.SearchQuery,
) (knowledgedomain.SearchResult, error) {
	return knowledgedomain.SearchResult{}, nil
}
func (*listDocumentStore) RevisionScope(context.Context, identitydomain.Principal, string) (string, string, error) {
	return "", "", nil
}

func TestListDocumentsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionKnowledgeRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Knowledge: knowledgeapp.Service{
			Store: &listDocumentStore{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/documents?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []knowledgedomain.Document `json:"items"`
		NextCursor *string                    `json:"next_cursor"`
		HasMore    bool                       `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}
