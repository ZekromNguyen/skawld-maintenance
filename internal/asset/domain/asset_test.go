package domain

import (
	"testing"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

func TestExternalAssetCannotUseNativeMutation(t *testing.T) {
	t.Parallel()
	asset, err := Create(NewAsset{
		ID:             "asset",
		OrganizationID: "org",
		SiteID:         "site",
		Tag:            "P-302",
		Name:           "Pump P-302",
		Class:          "CENTRIFUGAL_PUMP",
		SourceOfTruth:  integrationdomain.ExternalReference,
		ExternalReference: &ExternalReference{
			System: "SAP",
			ID:     "1000302",
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := asset.RequireNativeMutation(); err == nil {
		t.Fatal("external projection must reject native mutation")
	}
}

func TestContainmentCycleIsRejected(t *testing.T) {
	t.Parallel()
	existing := []Containment{
		{ParentID: "site", ChildID: "system"},
		{ParentID: "system", ChildID: "pump"},
	}
	if err := ValidateContainment(
		Containment{ParentID: "pump", ChildID: "site"},
		existing,
	); err == nil {
		t.Fatal("expected containment cycle")
	}
}
