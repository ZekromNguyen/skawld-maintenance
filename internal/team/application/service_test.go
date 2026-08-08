package application

import (
	"context"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type fakeStore struct{}

func (*fakeStore) ListTeams(context.Context, identitydomain.Principal) ([]Team, error) {
	return []Team{{ID: "t1", Name: "Facilities"}}, nil
}
func (*fakeStore) ListPeople(context.Context, identitydomain.Principal) ([]Person, error) {
	return []Person{{ID: "p1", DisplayName: "Ada"}}, nil
}

func TestListTeamsRequiresIncidentCreate(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org",
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	if _, err := service.ListTeams(context.Background(), principal); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	principal.Permissions[identitydomain.PermissionIncidentCreate] = struct{}{}
	teams, err := service.ListTeams(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].Name != "Facilities" {
		t.Fatalf("unexpected teams: %#v", teams)
	}
}

func TestListPeopleRequiresIncidentCreate(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org",
		Permissions: map[identitydomain.Permission]struct{}{},
	}
	if _, err := service.ListPeople(context.Background(), principal); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	principal.Permissions[identitydomain.PermissionIncidentCreate] = struct{}{}
	people, err := service.ListPeople(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 1 || people[0].DisplayName != "Ada" {
		t.Fatalf("unexpected people: %#v", people)
	}
}
