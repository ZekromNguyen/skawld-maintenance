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
		Number: "INC-1", Summary: "High vibration", Severity: SeverityHigh,
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
	if incident.State != StateResolved || incident.Version != 2 {
		t.Fatalf("unexpected resolved incident: %#v", incident)
	}
}
