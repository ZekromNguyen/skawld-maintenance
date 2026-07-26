package application

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

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
