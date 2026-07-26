package domain

import (
	"errors"
	"strings"
	"time"
)

type CriticalityRating string

const (
	CriticalityA CriticalityRating = "A"
	CriticalityB CriticalityRating = "B"
	CriticalityC CriticalityRating = "C"
)

type Criticality struct {
	ID                  string
	OrganizationID      string
	AssetID             string
	Rating              CriticalityRating
	SafetyImpact        int
	ProductionImpact    int
	EnvironmentalImpact int
	FinancialImpact     int
	Redundancy          string
	Rationale           string
	ApprovedBy          string
	ApprovedAt          time.Time
}

func NewCriticality(value Criticality) (Criticality, error) {
	if value.ID == "" || value.OrganizationID == "" || value.AssetID == "" || value.ApprovedBy == "" {
		return Criticality{}, errors.New("criticality identity, scope, asset, and approver are required")
	}
	if value.Rating != CriticalityA && value.Rating != CriticalityB && value.Rating != CriticalityC {
		return Criticality{}, errors.New("criticality rating must be A, B, or C")
	}
	for _, impact := range []int{
		value.SafetyImpact,
		value.ProductionImpact,
		value.EnvironmentalImpact,
		value.FinancialImpact,
	} {
		if impact < 0 || impact > 5 {
			return Criticality{}, errors.New("criticality impacts must be between 0 and 5")
		}
	}
	switch value.Redundancy {
	case "NONE", "PARTIAL", "FULL", "UNKNOWN":
	default:
		return Criticality{}, errors.New("unsupported redundancy value")
	}
	value.Rationale = strings.TrimSpace(value.Rationale)
	if value.Rationale == "" || value.ApprovedAt.IsZero() {
		return Criticality{}, errors.New("criticality rationale and approval time are required")
	}
	value.ApprovedAt = value.ApprovedAt.UTC()
	return value, nil
}
