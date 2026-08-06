package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type listAssetStore struct{}

func (*listAssetStore) Create(
	context.Context, identitydomain.Principal, string, assetapp.CreateAsset,
) (assetapp.Asset, bool, error) {
	return assetapp.Asset{}, false, nil
}
func (*listAssetStore) Get(context.Context, identitydomain.Principal, string) (assetapp.Asset, error) {
	return assetapp.Asset{}, nil
}
func (*listAssetStore) List(context.Context, identitydomain.Principal, assetapp.Filter) ([]assetapp.Asset, bool, error) {
	return []assetapp.Asset{{
		ID:        "00000000-0000-0000-0000-00000000000b",
		CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}
func (*listAssetStore) ApproveCriticality(
	context.Context, identitydomain.Principal, string, string, assetapp.ApproveCriticality,
) (assetapp.Criticality, bool, error) {
	return assetapp.Criticality{}, false, nil
}

type fakeAuthorities struct{}

func (fakeAuthorities) ForSubject(
	context.Context, string, string, string,
) ([]identitydomain.ApprovalAuthority, error) {
	return nil, nil
}

func TestListAssetsEnvelope(t *testing.T) {
	principal := identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAssetRead: {},
		},
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth:   fakeAuth{principal: principal},
		Assets: assetapp.Service{
			Store:       &listAssetStore{},
			Authorities: fakeAuthorities{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/assets?page_size=1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items      []assetapp.Asset `json:"items"`
		NextCursor *string          `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore || body.NextCursor == nil || *body.NextCursor == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}
