package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type editStore struct {
	edited bool
}

func (*editStore) LoadExecutionContext(context.Context, identitydomain.Principal, string) (ExecutionContext, error) {
	return ExecutionContext{}, nil
}
func (*editStore) SaveDraft(context.Context, identitydomain.Principal, string, Report, json.RawMessage) (Report, bool, error) {
	return Report{}, false, nil
}
func (*editStore) Get(context.Context, identitydomain.Principal, string) (Report, error) {
	return Report{Evidence: []knowledgedomain.Evidence{{
		ID: "measurement:allowed", Kind: "MEASUREMENT", SourceID: "allowed",
	}}}, nil
}
func (s *editStore) Edit(context.Context, identitydomain.Principal, string, string, Edit) (Report, bool, error) {
	s.edited = true
	return Report{}, false, nil
}
func (*editStore) Submit(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*editStore) Approve(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*editStore) RecordCall(context.Context, identitydomain.Principal, ExecutionContext, skawld.Generation, string, string) error {
	return nil
}
func (*editStore) List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error) {
	return nil, false, nil
}

type listReportStore struct{}

func (*listReportStore) LoadExecutionContext(context.Context, identitydomain.Principal, string) (ExecutionContext, error) {
	return ExecutionContext{}, nil
}
func (*listReportStore) SaveDraft(context.Context, identitydomain.Principal, string, Report, json.RawMessage) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Get(context.Context, identitydomain.Principal, string) (Report, error) {
	return Report{}, nil
}
func (*listReportStore) Edit(context.Context, identitydomain.Principal, string, string, Edit) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Submit(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) Approve(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error) {
	return Report{}, false, nil
}
func (*listReportStore) RecordCall(context.Context, identitydomain.Principal, ExecutionContext, skawld.Generation, string, string) error {
	return nil
}
func (*listReportStore) List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error) {
	return []Report{{
		ID: "00000000-0000-0000-0000-00000000000a", CreatedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}}, true, nil
}

func TestListReportsRequiresPermission(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	_, err := service.List(context.Background(), identitydomain.Principal{ID: "p1"}, ReportFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListReportsRejectsSiteOutsideScope(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	_, err := service.List(context.Background(), principal, ReportFilter{SiteID: "s2"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListReportsRejectsUnknownState(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	_, err := service.List(context.Background(), principal, ReportFilter{States: []string{"NOPE"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestListReportsComputesCursor(t *testing.T) {
	service := Service{Store: &listReportStore{}}
	principal := identitydomain.Principal{
		ID: "p1", OrganizationID: "o1", SiteIDs: []string{"s1"},
		Permissions: map[identitydomain.Permission]struct{}{identitydomain.PermissionReportWrite: {}},
	}
	result, err := service.List(context.Background(), principal, ReportFilter{})
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


func TestHumanEditCannotIntroduceEvidence(t *testing.T) {
	t.Parallel()
	store := &editStore{}
	service := Service{Store: store}
	_, _, err := service.Edit(
		context.Background(),
		identitydomain.Principal{
			Permissions: map[identitydomain.Permission]struct{}{
				identitydomain.PermissionReportWrite: {},
			},
		},
		"retry-key-123", "report", Edit{
			ExpectedVersion: 1,
			Content: Content{
				Summary: "Edited summary", EvidenceIDs: []string{"measurement:invented"},
				RequiresHumanReview: true,
			},
		},
	)
	if err != ErrInvalid {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
	if store.edited {
		t.Fatal("invalid edit must not reach persistence")
	}
}
