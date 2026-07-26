package domain

import (
	"testing"
	"time"
)

func TestIntrusiveStepBlockedUntilIsolationVerified(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	execution, err := NewExecution(Execution{
		ID: "execution", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Purpose: "Diagnose high vibration",
		Steps:   PumpInspectionSteps([]string{"1", "2", "3", "4"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := execution.CompleteStep("4", 1, now); err == nil {
		t.Fatal("intrusive step must be blocked without isolation")
	}
	execution.Verifications = append(execution.Verifications, Verification{
		Type: "ENERGY_ISOLATION", Status: "VERIFIED", ExternalReference: "LOTO-42",
		VerifiedBy: "supervisor", VerifiedAt: now.Add(-time.Minute),
	})
	if err := execution.CompleteStep("4", 1, now); err != nil {
		t.Fatal(err)
	}
}
