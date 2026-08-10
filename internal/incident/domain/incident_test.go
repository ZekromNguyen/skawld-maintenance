package domain

import (
	"testing"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

func TestResolveRequiresNativeIncidentAndSummary(t *testing.T) {
	t.Parallel()
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := incident.Resolve("", time.Now()); err == nil {
		t.Fatal("empty resolution must fail")
	}
	if err := incident.Resolve("Bearing replaced and verified", time.Now()); err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusResolved || incident.Version != 2 {
		t.Fatalf("unexpected resolved incident: %#v", incident)
	}
}

func TestNewValidatesPriorityAndDefaultsStatus(t *testing.T) {
	t.Parallel()
	_, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: "UNKNOWN",
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("unsupported priority must fail")
	}
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusOpen {
		t.Fatalf("status = %q, want OPEN", incident.Status)
	}
}

func TestResolveCloseReopenLifecycle(t *testing.T) {
	t.Parallel()
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := incident.Resolve("Bearing replaced", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := incident.Close(time.Now()); err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusClosed {
		t.Fatalf("status = %q, want CLOSED", incident.Status)
	}
	if err := incident.Reopen(); err != nil {
		t.Fatal(err)
	}
	if incident.Status != StatusReopened {
		t.Fatalf("status = %q, want REOPENED", incident.Status)
	}
	if incident.ResolvedAt != nil || incident.ResolutionSummary != "" {
		t.Fatal("reopen must clear resolution")
	}
	if err := incident.Close(time.Now()); err == nil {
		t.Fatal("closing a reopened incident directly must fail")
	}
}

func TestStartAllowsReopened(t *testing.T) {
	t.Parallel()
	incident, err := New(Incident{
		ID: "incident", OrganizationID: "org", SiteID: "site", AssetID: "asset",
		Number: "INC-1", Summary: "High vibration", Priority: PriorityHigh,
		SourceOfTruth: integrationdomain.OwnedBySkawld, DetectedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := incident.Resolve("Bearing replaced", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := incident.Reopen(); err != nil {
		t.Fatal(err)
	}
	if err := incident.Start(); err != nil {
		t.Fatalf("reopened incident must start: %v", err)
	}
	if incident.Status != StatusInProgress {
		t.Fatalf("status = %q, want IN_PROGRESS", incident.Status)
	}
}
