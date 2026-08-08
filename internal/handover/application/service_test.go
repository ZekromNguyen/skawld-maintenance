package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type fakeStore struct{}

func (*fakeStore) LoadWindowContext(context.Context, identitydomain.Principal, PrepareDraft) (WindowContext, error) {
	return WindowContext{}, nil
}
func (*fakeStore) SaveDraft(context.Context, identitydomain.Principal, string, Handover) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Get(context.Context, identitydomain.Principal, string) (Handover, error) {
	return Handover{}, nil
}
func (*fakeStore) Edit(context.Context, identitydomain.Principal, string, string, Edit) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Submit(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Accept(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) Acknowledge(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error) {
	return Handover{}, false, nil
}
func (*fakeStore) RecordCall(context.Context, identitydomain.Principal, WindowContext, skawld.Generation, string, string) error {
	return nil
}
func (*fakeStore) List(context.Context, identitydomain.Principal, HandoverFilter) ([]Handover, bool, error) {
	return []Handover{{
		ID: "00000000-0000-0000-0000-00000000000a", ShiftStart: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func TestDecodeContentRejectsUnknownEvidence(t *testing.T) {
	t.Parallel()
	_, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":[],
		"active_executions":[],
		"safety_concerns":[],
		"follow_up":[],
		"evidence_ids":["incident:invented"],
		"unknowns":[],
		"requires_human_review":true
	}`), []skawld.Evidence{{ID: "incident:eligible"}})
	if !errors.Is(err, skawld.ErrInvalidOutput) {
		t.Fatalf("error = %v, want invalid output", err)
	}
}

func TestListHandoversRequiresPermission(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	_, err := service.List(context.Background(), identitydomain.Principal{ID: "p1"}, HandoverFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListHandoversRejectsSiteOutsideScope(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	_, err := service.List(context.Background(), principal, HandoverFilter{SiteID: "s2"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListHandoversRejectsUnknownState(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	_, err := service.List(context.Background(), principal, HandoverFilter{States: []string{"NOPE"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestListHandoversComputesCursor(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionHandoverWrite: {}},
	}
	result, err := service.List(context.Background(), principal, HandoverFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if result.NextCursor == "" {
		t.Fatal("NextCursor must be set when HasMore")
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(result.Items))
	}
}

func TestDecodeContentAcceptsStructuredListItem(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":[{"title":"P-302 vibration","detail":"High vibes","severity":"HIGH"}],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "P-302 vibration" {
		t.Fatalf("open_incidents = %+v, want structured item", content.OpenIncidents)
	}
	if content.OpenIncidents[0].Severity != "HIGH" {
		t.Fatalf("severity = %q, want HIGH", content.OpenIncidents[0].Severity)
	}
}

func TestDecodeContentAcceptsLegacyStringList(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":["P-302 vibration"],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "P-302 vibration" {
		t.Fatalf("open_incidents = %+v, want legacy string normalized to title", content.OpenIncidents)
	}
}

func TestDecodeContentNormalizesJsonStringItems(t *testing.T) {
	t.Parallel()
	content, err := decodeContent(json.RawMessage(`{
		"summary":"Shift summary",
		"open_incidents":["{\"asset_tag\":\"P-302\",\"summary\":\"High vibes\"}"],
		"active_executions":[],"safety_concerns":[],"follow_up":[],
		"evidence_ids":[],"unknowns":[],
		"requires_human_review":true
	}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content.OpenIncidents) != 1 || content.OpenIncidents[0].Title != "High vibes" {
		t.Fatalf("open_incidents = %+v, want embedded summary promoted to title", content.OpenIncidents)
	}
}
