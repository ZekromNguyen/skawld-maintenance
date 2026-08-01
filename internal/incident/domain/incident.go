package domain

import (
	"errors"
	"strings"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

type State string
type Severity string

const (
	StateOpen       State = "OPEN"
	StateInProgress State = "IN_PROGRESS"
	StateResolved   State = "RESOLVED"

	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type Incident struct {
	ID                string
	OrganizationID    string
	SiteID            string
	AssetID           string
	Number            string
	Summary           string
	Severity          Severity
	State             State
	SourceOfTruth     integrationdomain.SourceOfTruth
	ExternalSystem    string
	ExternalID        string
	ExternalVersion   string
	OccurredAt        *time.Time
	DetectedAt        time.Time
	ResolvedAt        *time.Time
	ResolutionSummary string
	Version           int64
}

func New(value Incident) (Incident, error) {
	value.Summary = strings.TrimSpace(value.Summary)
	if value.ID == "" || value.OrganizationID == "" || value.SiteID == "" ||
		value.AssetID == "" || value.Number == "" || value.Summary == "" {
		return Incident{}, errors.New("incident identity, scope, asset, number, and summary are required")
	}
	switch value.Severity {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
	default:
		return Incident{}, errors.New("unsupported incident severity")
	}
	hasExternal := strings.TrimSpace(value.ExternalSystem) != "" &&
		strings.TrimSpace(value.ExternalID) != ""
	if err := value.SourceOfTruth.Validate(hasExternal); err != nil {
		return Incident{}, err
	}
	if value.DetectedAt.IsZero() {
		return Incident{}, errors.New("incident detection time is required")
	}
	value.State = StateOpen
	value.Version = 1
	value.DetectedAt = value.DetectedAt.UTC()
	if value.OccurredAt != nil {
		occurred := value.OccurredAt.UTC()
		value.OccurredAt = &occurred
	}
	return value, nil
}

func (i *Incident) Start() error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.State != StateOpen {
		return errors.New("only an open incident can move to in progress")
	}
	i.State = StateInProgress
	i.Version++
	return nil
}

func (i *Incident) Resolve(summary string, at time.Time) error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.State == StateResolved {
		return errors.New("incident is already resolved")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" || at.IsZero() {
		return errors.New("resolution summary and time are required")
	}
	resolvedAt := at.UTC()
	i.State = StateResolved
	i.ResolutionSummary = summary
	i.ResolvedAt = &resolvedAt
	i.Version++
	return nil
}

func (i Incident) requireNative() error {
	if i.SourceOfTruth != integrationdomain.OwnedBySkawld {
		return errors.New("external incident projection cannot use native lifecycle commands")
	}
	return nil
}
