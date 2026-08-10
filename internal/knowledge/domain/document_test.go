package domain

import (
	"testing"
	"time"
)

func TestRevisionEligibilityFailsClosed(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	approved := Revision{
		ApprovalStatus: ApprovalApproved,
		IngestionState: IngestionReady,
	}
	if !approved.Eligible(now) {
		t.Fatal("current approved and ingested revision should be eligible")
	}
	cases := map[string]Revision{
		"draft":      {ApprovalStatus: ApprovalDraft, IngestionState: IngestionReady},
		"processing": {ApprovalStatus: ApprovalApproved, IngestionState: IngestionProcessing},
		"superseded": {
			ApprovalStatus: ApprovalApproved, IngestionState: IngestionReady,
			SupersededByID: "new",
		},
		"future": {
			ApprovalStatus: ApprovalApproved, IngestionState: IngestionReady,
			EffectiveAt: pointer(now.Add(time.Hour)),
		},
		"expired": {
			ApprovalStatus: ApprovalApproved, IngestionState: IngestionReady,
			ExpiresAt: pointer(now),
		},
	}
	for name, revision := range cases {
		if revision.Eligible(now) {
			t.Errorf("%s revision must be ineligible", name)
		}
	}
}

func pointer(value time.Time) *time.Time {
	return &value
}
