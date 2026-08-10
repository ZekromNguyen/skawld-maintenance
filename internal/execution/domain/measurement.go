package domain

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

type MeasurementType string
type Unit string

const (
	MeasurementVibrationVelocity MeasurementType = "VIBRATION_VELOCITY"
	MeasurementTemperature       MeasurementType = "TEMPERATURE"

	UnitMillimetersPerSecond Unit = "MM_PER_S"
	UnitInchesPerSecond      Unit = "IN_PER_S"
	UnitCelsius              Unit = "DEG_C"
	UnitFahrenheit           Unit = "DEG_F"
)

var decimalPattern = regexp.MustCompile(`\A-?(0|[1-9][0-9]*)(\.[0-9]{1,6})?\z`)

type Measurement struct {
	ID                  string
	OrganizationID      string
	SiteID              string
	ExecutionID         string
	AssetID             string
	ComponentID         string
	Type                MeasurementType
	Value               string
	Unit                Unit
	OriginalValue       string
	OriginalUnit        Unit
	Source              string
	DataQuality         string
	InstrumentReference string
	VerificationStatus  string
	ObservedAt          time.Time
	RecordedBy          string
	ClientEventID       string
	DeviceID            string
	CreatedAtDevice     *time.Time
	ReceivedAtServer    time.Time
}

func NewMeasurement(value Measurement) (Measurement, error) {
	if value.ID == "" || value.OrganizationID == "" || value.SiteID == "" ||
		value.ExecutionID == "" || value.AssetID == "" || value.RecordedBy == "" {
		return Measurement{}, errors.New("measurement identity, scope, execution, asset, and recorder are required")
	}
	original := strings.TrimSpace(value.OriginalValue)
	if !decimalPattern.MatchString(original) {
		return Measurement{}, errors.New("measurement value must be a finite decimal with at most 6 decimal places")
	}
	rational, ok := new(big.Rat).SetString(original)
	if !ok {
		return Measurement{}, errors.New("invalid decimal measurement")
	}
	canonical, canonicalUnit, err := convertToCanonical(value.Type, value.OriginalUnit, rational)
	if err != nil {
		return Measurement{}, err
	}
	if err := validateRange(value.Type, canonical); err != nil {
		return Measurement{}, err
	}
	switch value.Source {
	case "MANUAL", "INSTRUMENT", "EXTERNAL_SYSTEM":
	default:
		return Measurement{}, errors.New("unsupported measurement source")
	}
	switch value.DataQuality {
	case "GOOD", "QUESTIONABLE", "BAD", "UNKNOWN":
	default:
		return Measurement{}, errors.New("unsupported data quality")
	}
	switch value.VerificationStatus {
	case "UNVERIFIED", "VERIFIED", "REJECTED":
	default:
		return Measurement{}, errors.New("unsupported verification status")
	}
	if value.ObservedAt.IsZero() || value.ReceivedAtServer.IsZero() {
		return Measurement{}, errors.New("measurement observation and server receive times are required")
	}
	value.Value = formatRat(canonical, 6)
	value.Unit = canonicalUnit
	value.OriginalValue = original
	value.ObservedAt = value.ObservedAt.UTC()
	value.ReceivedAtServer = value.ReceivedAtServer.UTC()
	if value.CreatedAtDevice != nil {
		deviceTime := value.CreatedAtDevice.UTC()
		value.CreatedAtDevice = &deviceTime
	}
	return value, nil
}

func Convert(value string, measurementType MeasurementType, from, to Unit) (string, error) {
	if !decimalPattern.MatchString(value) {
		return "", errors.New("invalid decimal")
	}
	rational, _ := new(big.Rat).SetString(value)
	canonical, _, err := convertToCanonical(measurementType, from, rational)
	if err != nil {
		return "", err
	}
	var converted *big.Rat
	switch {
	case measurementType == MeasurementVibrationVelocity && to == UnitMillimetersPerSecond:
		converted = canonical
	case measurementType == MeasurementVibrationVelocity && to == UnitInchesPerSecond:
		converted = new(big.Rat).Quo(canonical, big.NewRat(127, 5))
	case measurementType == MeasurementTemperature && to == UnitCelsius:
		converted = canonical
	case measurementType == MeasurementTemperature && to == UnitFahrenheit:
		converted = new(big.Rat).Add(
			new(big.Rat).Mul(canonical, big.NewRat(9, 5)),
			big.NewRat(32, 1),
		)
	default:
		return "", fmt.Errorf("unit %s is incompatible with measurement type %s", to, measurementType)
	}
	return formatRat(converted, 6), nil
}

func convertToCanonical(
	measurementType MeasurementType,
	unit Unit,
	value *big.Rat,
) (*big.Rat, Unit, error) {
	switch measurementType {
	case MeasurementVibrationVelocity:
		switch unit {
		case UnitMillimetersPerSecond:
			return new(big.Rat).Set(value), UnitMillimetersPerSecond, nil
		case UnitInchesPerSecond:
			return new(big.Rat).Mul(value, big.NewRat(127, 5)), UnitMillimetersPerSecond, nil
		}
	case MeasurementTemperature:
		switch unit {
		case UnitCelsius:
			return new(big.Rat).Set(value), UnitCelsius, nil
		case UnitFahrenheit:
			return new(big.Rat).Mul(
				new(big.Rat).Sub(value, big.NewRat(32, 1)),
				big.NewRat(5, 9),
			), UnitCelsius, nil
		}
	}
	return nil, "", fmt.Errorf("unit %s is incompatible with measurement type %s", unit, measurementType)
}

func validateRange(measurementType MeasurementType, value *big.Rat) error {
	switch measurementType {
	case MeasurementVibrationVelocity:
		if value.Sign() < 0 || value.Cmp(big.NewRat(1000, 1)) > 0 {
			return errors.New("vibration velocity is outside supported maintenance range")
		}
	case MeasurementTemperature:
		if value.Cmp(big.NewRat(-27315, 100)) < 0 || value.Cmp(big.NewRat(2000, 1)) > 0 {
			return errors.New("temperature is outside supported physical range")
		}
	default:
		return errors.New("unsupported measurement type")
	}
	return nil
}

func formatRat(value *big.Rat, scale int) string {
	text := value.FloatString(scale)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	if text == "-0" {
		return "0"
	}
	return text
}
