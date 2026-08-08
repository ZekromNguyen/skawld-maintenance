package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"
)

type listTeamStore struct{}

func (*listTeamStore) ListTeams(context.Context, identitydomain.Principal) ([]teamapp.Team, error) {
	return []teamapp.Team{{ID: "t1", Name: "Facilities"}}, nil
}
func (*listTeamStore) ListPeople(context.Context, identitydomain.Principal) ([]teamapp.Person, error) {
	return []teamapp.Person{{ID: "p1", DisplayName: "Ada Lovelace"}}, nil
}

func principalWith(permissions ...identitydomain.Permission) identitydomain.Principal {
	return identitydomain.Principal{
		ID: "00000000-0000-0000-0000-000000000001", OrganizationID: "o1",
		SiteIDs: []string{"11111111-1111-4111-8111-111111111111"},
		Permissions: func() map[identitydomain.Permission]struct{} {
			set := map[identitydomain.Permission]struct{}{}
			for _, permission := range permissions {
				set[permission] = struct{}{}
			}
			return set
		}(),
	}
}

func TestListTeamsEnvelope(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth: fakeAuth{principal: principalWith(
			identitydomain.PermissionIncidentCreate,
		)},
		Teams: teamapp.Service{Store: &listTeamStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items []teamapp.Team `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Name != "Facilities" {
		t.Fatalf("unexpected teams: %#v", body.Items)
	}
}

func TestListPeopleEnvelope(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.DiscardHandler),
		Auth: fakeAuth{principal: principalWith(
			identitydomain.PermissionIncidentCreate,
		)},
		Teams: teamapp.Service{Store: &listTeamStore{}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/people", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Items []teamapp.Person `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].DisplayName != "Ada Lovelace" {
		t.Fatalf("unexpected people: %#v", body.Items)
	}
}
