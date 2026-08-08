package domain

import (
	"errors"
	"strings"
	"time"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

type Status string
type Priority string

const (
	StatusOpen       Status = "OPEN"
	StatusInProgress Status = "IN_PROGRESS"
	StatusResolved   Status = "RESOLVED"
	StatusClosed     Status = "CLOSED"
	StatusReopened   Status = "REOPENED"

	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

type Incident struct {
	ID                string
	OrganizationID    string
	SiteID            string
	AssetID           string
	Number            string
	Summary           string
	Details           string
	Priority          Priority
	Status            Status
	AssigneeID        string
	ReporterID        string
	TeamID            string
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
	switch value.Priority {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
	default:
		return Incident{}, errors.New("unsupported incident priority")
	}
	hasExternal := strings.TrimSpace(value.ExternalSystem) != "" &&
		strings.TrimSpace(value.ExternalID) != ""
	if err := value.SourceOfTruth.Validate(hasExternal); err != nil {
		return Incident{}, err
	}
	if value.DetectedAt.IsZero() {
		return Incident{}, errors.New("incident detection time is required")
	}
	if value.Status == "" {
		value.Status = StatusOpen
	}
	switch value.Status {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed, StatusReopened:
	default:
		return Incident{}, errors.New("unsupported incident status")
	}
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
	if i.Status != StatusOpen && i.Status != StatusReopened {
		return errors.New("only an open or reopened incident can move to in progress")
	}
	i.Status = StatusInProgress
	i.Version++
	return nil
}

func (i *Incident) Resolve(summary string, at time.Time) error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.Status == StatusResolved {
		return errors.New("incident is already resolved")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" || at.IsZero() {
		return errors.New("resolution summary and time are required")
	}
	resolvedAt := at.UTC()
	i.Status = StatusResolved
	i.ResolutionSummary = summary
	i.ResolvedAt = &resolvedAt
	i.Version++
	return nil
}

func (i *Incident) Close(at time.Time) error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.Status != StatusResolved {
		return errors.New("only a resolved incident can be closed")
	}
	if at.IsZero() {
		return errors.New("resolution time is required")
	}
	closedAt := at.UTC()
	if i.ResolvedAt == nil {
		i.ResolvedAt = &closedAt
	}
	i.Status = StatusClosed
	i.Version++
	return nil
}

func (i *Incident) Reopen() error {
	if err := i.requireNative(); err != nil {
		return err
	}
	if i.Status != StatusResolved && i.Status != StatusClosed {
		return errors.New("only a resolved or closed incident can be reopened")
	}
	i.Status = StatusReopened
	i.ResolutionSummary = ""
	i.ResolvedAt = nil
	i.Version++
	return nil
}

func (i Incident) requireNative() error {
	if i.SourceOfTruth != integrationdomain.OwnedBySkawld {
		return errors.New("external incident projection cannot use native lifecycle commands")
	}
	return nil
}
