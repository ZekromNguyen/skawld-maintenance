package application

import (
	"context"
	"errors"
	"testing"
	"time"

	customfieldapp "github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/customfield/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type stubStore struct {
	items      []Incident
	lastFilter Filter
}

func (s *stubStore) Create(_ context.Context, _ identitydomain.Principal, _ string, command CreateIncident) (Incident, bool, error) {
	return Incident{
		ID: "inc-1", OrganizationID: "o1", SiteID: command.SiteID,
		AssetID: command.AssetID, Number: "INC-1", Summary: command.Summary,
		Priority: command.Priority, Status: "OPEN", DetectedAt: command.DetectedAt,
		CustomValues: command.CustomValues, Version: 1,
	}, false, nil
}

func (s *stubStore) Get(_ context.Context, _ identitydomain.Principal, id string) (Incident, error) {
	return Incident{ID: id}, nil
}

func (s *stubStore) List(_ context.Context, _ identitydomain.Principal, filter Filter) ([]Incident, bool, error) {
	s.lastFilter = filter
	return s.items, false, nil
}

func (s *stubStore) Resolve(_ context.Context, _ identitydomain.Principal, _, _ string, _ ResolveIncident) (Incident, bool, error) {
	return Incident{}, false, nil
}

func (s *stubStore) Close(_ context.Context, _ identitydomain.Principal, _, _ string, _ CloseIncident) (Incident, bool, error) {
	return Incident{}, false, nil
}

func (s *stubStore) Reopen(_ context.Context, _ identitydomain.Principal, _, _ string, _ ReopenIncident) (Incident, bool, error) {
	return Incident{}, false, nil
}

func (s *stubStore) UpdateCustomValues(_ context.Context, _ identitydomain.Principal, _, incidentID string, command UpdateCustomValues) (Incident, bool, error) {
	return Incident{ID: incidentID, CustomValues: command.CustomValues, Version: command.ExpectedVersion + 1}, false, nil
}

type fakeFields struct {
	types map[string]string // field key -> field_type
}

func (f fakeFields) Definitions(_ context.Context, _, _ string) ([]customfieldapp.Definition, error) {
	return []customfieldapp.Definition{
		{ID: "def-1", Key: "po_number", FieldType: "TEXT", Config: domain.Config{Required: true}},
	}, nil
}

func (f fakeFields) ResolveAndValidate(_ context.Context, _, _ string, values map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for key, value := range values {
		if key != "po_number" {
			return nil, errors.New("unknown custom field")
		}
		out["def-1"] = value
	}
	return out, nil
}

func (f fakeFields) ResolveKeys(_ context.Context, _, _ string, keys []string) (map[string]customfieldapp.Definition, error) {
	out := map[string]customfieldapp.Definition{}
	for _, key := range keys {
		fieldType := f.types[key]
		if fieldType == "" {
			fieldType = "TEXT"
		}
		out[key] = customfieldapp.Definition{ID: "def-" + key, Key: key, FieldType: fieldType}
	}
	return out, nil
}

func incidentPrincipal() identitydomain.Principal {
	return identitydomain.Principal{
		OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionIncidentCreate: {},
			identitydomain.PermissionIncidentRead:   {},
		},
	}
}

func TestCreateValidatesCustomValues(t *testing.T) {
	service := Service{Store: &stubStore{}, Fields: fakeFields{}}
	command := CreateIncident{
		SiteID: "s1", AssetID: "a1", Summary: "Pump vibration", Priority: "HIGH",
		DetectedAt: time.Now(), CustomValues: map[string]any{"po_number": "PO-42"},
	}
	incident, _, err := service.Create(context.Background(), incidentPrincipal(), "test-key-123", command)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	value, ok := incident.CustomValues["def-1"]
	if !ok || value != "PO-42" {
		t.Fatalf("expected normalized custom value keyed by definition id, got %#v", incident.CustomValues)
	}
}

func TestCreateRejectsUnknownCustomField(t *testing.T) {
	service := Service{Store: &stubStore{}, Fields: fakeFields{}}
	command := CreateIncident{
		SiteID: "s1", AssetID: "a1", Summary: "Pump vibration", Priority: "HIGH",
		DetectedAt: time.Now(), CustomValues: map[string]any{"nope": "x"},
	}
	if _, _, err := service.Create(context.Background(), incidentPrincipal(), "test-key-123", command); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown custom field must be ErrInvalid, got %v", err)
	}
}

func TestListResolvesCustomFieldFilterKeys(t *testing.T) {
	service := Service{Store: &stubStore{}, Fields: fakeFields{}}
	filter := Filter{
		SiteID:       "s1",
		CustomFields: map[string]string{"po_number": "PO-42"},
	}
	if _, _, err := service.List(context.Background(), incidentPrincipal(), filter); err != nil {
		t.Fatalf("list with custom field filter: %v", err)
	}
}

func TestListParsesNumberRangeFilter(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "cold", DetectedAt: time.Unix(1, 0).UTC()},
		{ID: "i2", Number: "INC-2", Summary: "warm", DetectedAt: time.Unix(2, 0).UTC()},
		{ID: "i3", Number: "INC-3", Summary: "hot", DetectedAt: time.Unix(3, 0).UTC()},
	}}
	s := Service{
		Store:  store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "20:30"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ranges := store.lastFilter.CustomFieldRanges
	if len(ranges) != 1 {
		t.Fatalf("CustomFieldRanges = %#v, want 1 entry", ranges)
	}
	r := ranges["def-temperature"]
	if r.Min == nil || *r.Min != 20 || r.Max == nil || *r.Max != 30 {
		t.Fatalf("range = min %v max %v, want 20..30", r.Min, r.Max)
	}
}

func TestListParsesOpenEndedNumberRanges(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "s", DetectedAt: time.Unix(1, 0).UTC()},
	}}
	s := Service{
		Store:  store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": ":30"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	r := store.lastFilter.CustomFieldRanges["def-temperature"]
	if r.Min != nil || r.Max == nil || *r.Max != 30 {
		t.Fatalf("range = min %v max %v, want open min, max 30", r.Min, r.Max)
	}
}

func TestListKeepsExactMatchForNonRangeValues(t *testing.T) {
	store := &stubStore{items: []Incident{
		{ID: "i1", Number: "INC-1", Summary: "s", DetectedAt: time.Unix(1, 0).UTC()},
	}}
	s := Service{
		Store:  store,
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER", "note": "TEXT"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "42", "note": "a:b"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(store.lastFilter.CustomFieldRanges) != 0 {
		t.Fatalf("CustomFieldRanges = %#v, want none", store.lastFilter.CustomFieldRanges)
	}
	exact := store.lastFilter.CustomFields
	if exact["def-temperature"] != "42" || exact["def-note"] != "a:b" {
		t.Fatalf("exact filters = %#v", exact)
	}
}

func TestListRejectsMalformedNumberRange(t *testing.T) {
	s := Service{
		Store:  &stubStore{},
		Fields: fakeFields{types: map[string]string{"temperature": "NUMBER"}},
	}
	_, _, err := s.List(context.Background(), incidentPrincipal(), Filter{
		CustomFields: map[string]string{"temperature": "abc:def"},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestUpdateCustomValuesValidatesAndMerges(t *testing.T) {
	service := Service{Store: &stubStore{}, Fields: fakeFields{}}
	command := UpdateCustomValues{
		ExpectedVersion: 1,
		CustomValues:    map[string]any{"po_number": "PO-99"},
	}
	incident, _, err := service.UpdateCustomValues(context.Background(), incidentPrincipal(), "test-key-123", "inc-1", command)
	if err != nil {
		t.Fatalf("update custom values: %v", err)
	}
	value, ok := incident.CustomValues["def-1"]
	if !ok || value != "PO-99" {
		t.Fatalf("expected normalized value keyed by definition id, got %#v", incident.CustomValues)
	}
}
