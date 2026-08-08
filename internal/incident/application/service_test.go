package application

import (
	"context"
	"errors"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

// recordingStore captures the Filter passed to List so tests can assert
// server-side filter forwarding without a database.
type recordingStore struct {
	got Filter
}

func (r *recordingStore) Create(context.Context, identitydomain.Principal, string, CreateIncident) (Incident, bool, error) {
	return Incident{}, false, errors.New("unexpected Create")
}

func (r *recordingStore) Get(context.Context, identitydomain.Principal, string) (Incident, error) {
	return Incident{}, errors.New("unexpected Get")
}

func (r *recordingStore) List(_ context.Context, _ identitydomain.Principal, filter Filter) ([]Incident, bool, error) {
	r.got = filter
	return nil, false, nil
}

func (r *recordingStore) Resolve(context.Context, identitydomain.Principal, string, string, ResolveIncident) (Incident, bool, error) {
	return Incident{}, false, errors.New("unexpected Resolve")
}

func TestListForwardsSeverityFilter(t *testing.T) {
	t.Parallel()
	recorder := &recordingStore{}
	service := Service{Store: recorder}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionIncidentRead: {}},
	}
	_, _, err := service.List(context.Background(), principal, Filter{Severity: "HIGH"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recorder.got.Severity != "HIGH" {
		t.Fatalf("severity = %q, want HIGH", recorder.got.Severity)
	}
}
