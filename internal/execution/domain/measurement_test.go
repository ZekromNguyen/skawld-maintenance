package domain

import (
	"testing"
	"time"
)

func TestPumpMeasurementsPersistExactly(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, test := range []struct {
		kind      MeasurementType
		input     string
		unit      Unit
		wantValue string
		wantUnit  Unit
	}{
		{MeasurementVibrationVelocity, "8.1", UnitMillimetersPerSecond, "8.1", UnitMillimetersPerSecond},
		{MeasurementTemperature, "94", UnitCelsius, "94", UnitCelsius},
		{MeasurementTemperature, "201.2", UnitFahrenheit, "94", UnitCelsius},
	} {
		value, err := NewMeasurement(Measurement{
			ID: "m", OrganizationID: "org", SiteID: "site", ExecutionID: "exec",
			AssetID: "asset", Type: test.kind, OriginalValue: test.input,
			OriginalUnit: test.unit, Source: "INSTRUMENT", DataQuality: "GOOD",
			VerificationStatus: "UNVERIFIED", ObservedAt: now,
			RecordedBy: "person", ReceivedAtServer: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if value.Value != test.wantValue || value.Unit != test.wantUnit {
			t.Fatalf("got %s %s, want %s %s", value.Value, value.Unit, test.wantValue, test.wantUnit)
		}
	}
}

func TestIncompatibleUnitFails(t *testing.T) {
	t.Parallel()
	if _, err := Convert("8.1", MeasurementVibrationVelocity, UnitCelsius, UnitMillimetersPerSecond); err == nil {
		t.Fatal("temperature unit must not be accepted for vibration")
	}
}
