package application

import (
	"context"
	"encoding/json"
	"testing"

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
