package application

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"
)

var (
	ErrForbidden       = errors.New("execution operation forbidden")
	ErrNotFound        = errors.New("execution not found")
	ErrInvalid         = errors.New("invalid execution command")
	ErrVersionConflict = errors.New("execution version conflict")
	ErrBlocked         = errors.New("execution step is blocked")
)

type CreateExecution struct {
	IncidentID string `json:"incident_id"`
	AssignedTo string `json:"assigned_to,omitempty"`
	Purpose    string `json:"purpose"`
}

type SyncMetadata struct {
	ClientEventID     string     `json:"client_event_id,omitempty"`
	DeviceID          string     `json:"device_id,omitempty"`
	CreatedAtDevice   *time.Time `json:"created_at_device,omitempty"`
	PayloadVersion    int        `json:"payload_version,omitempty"`
	BaseServerVersion int64      `json:"base_server_version,omitempty"`
}

type StartExecution struct {
	ExpectedVersion int64 `json:"expected_version"`
	SyncMetadata
}

type VerifyPrerequisite struct {
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	ExternalReference string     `json:"external_reference"`
	VerifiedAt        time.Time  `json:"verified_at"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
}

type CompleteStep struct {
	ExpectedExecutionVersion int64 `json:"expected_execution_version"`
	ExpectedStepVersion      int64 `json:"expected_step_version"`
	SyncMetadata
}

type RecordMeasurement struct {
	ClientEventID       string     `json:"client_event_id"`
	DeviceID            string     `json:"device_id,omitempty"`
	CreatedAtDevice     *time.Time `json:"created_at_device,omitempty"`
	ComponentID         string     `json:"component_id,omitempty"`
	MeasurementType     string     `json:"measurement_type"`
	Value               string     `json:"value"`
	Unit                string     `json:"unit"`
	Source              string     `json:"source"`
	DataQuality         string     `json:"data_quality"`
	InstrumentReference string     `json:"instrument_reference,omitempty"`
	VerificationStatus  string     `json:"verification_status"`
	ObservedAt          time.Time  `json:"observed_at"`
}

type RecordObservation struct {
	ClientEventID      string     `json:"client_event_id"`
	DeviceID           string     `json:"device_id,omitempty"`
	CreatedAtDevice    *time.Time `json:"created_at_device,omitempty"`
	ComponentID        string     `json:"component_id,omitempty"`
	Property           string     `json:"property,omitempty"`
	Status             string     `json:"status,omitempty"`
	Narrative          string     `json:"narrative"`
	Source             string     `json:"source"`
	VerificationStatus string     `json:"verification_status"`
	ObservedAt         time.Time  `json:"observed_at"`
}

type RecordAction struct {
	StepID      string    `json:"step_id,omitempty"`
	ComponentID string    `json:"component_id,omitempty"`
	ActionType  string    `json:"action_type"`
	Narrative   string    `json:"narrative"`
	Outcome     string    `json:"outcome,omitempty"`
	PerformedAt time.Time `json:"performed_at"`
}

type RecordDecision struct {
	ClientEventID   string     `json:"client_event_id"`
	DeviceID        string     `json:"device_id,omitempty"`
	CreatedAtDevice *time.Time `json:"created_at_device,omitempty"`
	StepID          string     `json:"step_id,omitempty"`
	ComponentID     string     `json:"component_id,omitempty"`
	Decision        string     `json:"decision"`
	Rationale       string     `json:"rationale"`
	Alternatives    []string   `json:"alternatives,omitempty"`
	DecidedAt       time.Time  `json:"decided_at"`
}

type CompleteExecution struct {
	ExpectedVersion int64  `json:"expected_version"`
	OutcomeSummary  string `json:"outcome_summary"`
}

type Filter struct {
	SiteID     string
	State      string
	AssignedTo string
	PageSize   int
	Cursor     string
}

type Step struct {
	ID                   string     `json:"id"`
	Key                  string     `json:"key"`
	Sequence             int        `json:"sequence"`
	Title                string     `json:"title"`
	State                string     `json:"state"`
	RiskLevel            string     `json:"risk_level"`
	RequiredPrerequisite string     `json:"required_prerequisite,omitempty"`
	BlockedReason        string     `json:"blocked_reason,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	Version              int64      `json:"version"`
}

type Prerequisite struct {
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	ExternalReference string     `json:"external_reference"`
	VerifiedBy        string     `json:"verified_by"`
	VerifiedAt        time.Time  `json:"verified_at"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
}

type Measurement struct {
	ID                 string     `json:"id"`
	ComponentID        string     `json:"component_id,omitempty"`
	MeasurementType    string     `json:"measurement_type"`
	Value              string     `json:"value"`
	Unit               string     `json:"unit"`
	OriginalValue      string     `json:"original_value"`
	OriginalUnit       string     `json:"original_unit"`
	Source             string     `json:"source"`
	DataQuality        string     `json:"data_quality"`
	VerificationStatus string     `json:"verification_status"`
	ObservedAt         time.Time  `json:"observed_at"`
	ClientEventID      string     `json:"client_event_id,omitempty"`
	CreatedAtDevice    *time.Time `json:"created_at_device,omitempty"`
	ReceivedAtServer   time.Time  `json:"received_at_server"`
}

type Observation struct {
	ID                 string    `json:"id"`
	ComponentID        string    `json:"component_id,omitempty"`
	Property           string    `json:"property,omitempty"`
	Status             string    `json:"status,omitempty"`
	Narrative          string    `json:"narrative"`
	Source             string    `json:"source"`
	VerificationStatus string    `json:"verification_status"`
	ObservedAt         time.Time `json:"observed_at"`
	ClientEventID      string    `json:"client_event_id,omitempty"`
}

type Action struct {
	ID          string    `json:"id"`
	StepID      string    `json:"step_id,omitempty"`
	ComponentID string    `json:"component_id,omitempty"`
	ActionType  string    `json:"action_type"`
	Narrative   string    `json:"narrative"`
	Outcome     string    `json:"outcome,omitempty"`
	PerformedBy string    `json:"performed_by"`
	PerformedAt time.Time `json:"performed_at"`
}

type Decision struct {
	ID            string    `json:"id"`
	StepID        string    `json:"step_id,omitempty"`
	ComponentID   string    `json:"component_id,omitempty"`
	Decision      string    `json:"decision"`
	Rationale     string    `json:"rationale"`
	Alternatives  []string  `json:"alternatives"`
	DecidedBy     string    `json:"decided_by"`
	DecidedAt     time.Time `json:"decided_at"`
	ClientEventID string    `json:"client_event_id,omitempty"`
}

type Execution struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	SiteID         string         `json:"site_id"`
	IncidentID     string         `json:"incident_id,omitempty"`
	IncidentNumber string         `json:"incident_number,omitempty"`
	AssetID        string         `json:"asset_id"`
	AssetTag       string         `json:"asset_tag"`
	Purpose        string         `json:"purpose"`
	State          string         `json:"state"`
	AssignedTo     string         `json:"assigned_to,omitempty"`
	Version        int64          `json:"version"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	OutcomeSummary string         `json:"outcome_summary,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Steps          []Step         `json:"steps"`
	Prerequisites  []Prerequisite `json:"prerequisites"`
	Measurements   []Measurement  `json:"measurements"`
	Observations   []Observation  `json:"observations"`
	Actions        []Action       `json:"actions"`
	Decisions      []Decision     `json:"decisions"`
}

type Store interface {
	Create(context.Context, identitydomain.Principal, string, CreateExecution) (Execution, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Execution, error)
	List(context.Context, identitydomain.Principal, Filter) ([]Execution, bool, error)
	Start(context.Context, identitydomain.Principal, string, string, StartExecution) (Execution, bool, error)
	VerifyPrerequisite(context.Context, identitydomain.Principal, string, string, VerifyPrerequisite) (Prerequisite, bool, error)
	CompleteStep(context.Context, identitydomain.Principal, string, string, string, CompleteStep) (Step, bool, error)
	RecordMeasurement(context.Context, identitydomain.Principal, string, string, RecordMeasurement) (Measurement, bool, error)
	RecordObservation(context.Context, identitydomain.Principal, string, string, RecordObservation) (Observation, bool, error)
	RecordAction(context.Context, identitydomain.Principal, string, string, RecordAction) (Action, bool, error)
	RecordDecision(context.Context, identitydomain.Principal, string, string, RecordDecision) (Decision, bool, error)
	Complete(context.Context, identitydomain.Principal, string, string, CompleteExecution) (Execution, bool, error)
}

type Service struct {
	Store Store
}

func (s Service) Create(ctx context.Context, p identitydomain.Principal, key string, c CreateExecution) (Execution, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Execution{}, false, ErrForbidden
	}
	if !validKey(key) || c.IncidentID == "" || strings.TrimSpace(c.Purpose) == "" {
		return Execution{}, false, ErrInvalid
	}
	return s.Store.Create(ctx, p, key, c)
}

func (s Service) Get(ctx context.Context, p identitydomain.Principal, id string) (Execution, error) {
	if !p.Has(identitydomain.PermissionExecutionRead) {
		return Execution{}, ErrForbidden
	}
	return s.Store.Get(ctx, p, id)
}

func (s Service) List(
	ctx context.Context,
	p identitydomain.Principal,
	filter Filter,
) ([]Execution, string, error) {
	if !p.Has(identitydomain.PermissionExecutionRead) {
		return nil, "", ErrForbidden
	}
	if filter.SiteID != "" && !p.CanAccessSite(p.OrganizationID, filter.SiteID) {
		return nil, "", ErrForbidden
	}
	if !p.Has(identitydomain.PermissionExecutionReadAll) || filter.AssignedTo == "me" {
		filter.AssignedTo = p.ID
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return nil, "", ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, p, filter)
	if err != nil {
		return nil, "", err
	}
	if !hasMore || len(items) == 0 {
		return items, "", nil
	}
	last := items[len(items)-1]
	return items, keyset.Encode(last.UpdatedAt, last.ID), nil
}

func (s Service) Start(ctx context.Context, p identitydomain.Principal, key, id string, c StartExecution) (Execution, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Execution{}, false, ErrForbidden
	}
	if !validKey(key) || c.ExpectedVersion <= 0 || !validSyncMetadata(c.SyncMetadata) {
		return Execution{}, false, ErrInvalid
	}
	return s.Store.Start(ctx, p, key, id, c)
}

func (s Service) VerifyPrerequisite(ctx context.Context, p identitydomain.Principal, key, id string, c VerifyPrerequisite) (Prerequisite, bool, error) {
	if !p.Has(identitydomain.PermissionPrerequisiteVerify) {
		return Prerequisite{}, false, ErrForbidden
	}
	if !validKey(key) || c.Type == "" || c.Status == "" ||
		strings.TrimSpace(c.ExternalReference) == "" || c.VerifiedAt.IsZero() {
		return Prerequisite{}, false, ErrInvalid
	}
	return s.Store.VerifyPrerequisite(ctx, p, key, id, c)
}

func (s Service) CompleteStep(ctx context.Context, p identitydomain.Principal, key, id, stepID string, c CompleteStep) (Step, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Step{}, false, ErrForbidden
	}
	if !validKey(key) || c.ExpectedExecutionVersion <= 0 || c.ExpectedStepVersion <= 0 ||
		!validSyncMetadata(c.SyncMetadata) {
		return Step{}, false, ErrInvalid
	}
	return s.Store.CompleteStep(ctx, p, key, id, stepID, c)
}

func (s Service) RecordMeasurement(ctx context.Context, p identitydomain.Principal, key, id string, c RecordMeasurement) (Measurement, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Measurement{}, false, ErrForbidden
	}
	if !validKey(key) || c.ClientEventID == "" {
		return Measurement{}, false, ErrInvalid
	}
	return s.Store.RecordMeasurement(ctx, p, key, id, c)
}

func (s Service) RecordObservation(ctx context.Context, p identitydomain.Principal, key, id string, c RecordObservation) (Observation, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Observation{}, false, ErrForbidden
	}
	if !validKey(key) || c.ClientEventID == "" || strings.TrimSpace(c.Narrative) == "" {
		return Observation{}, false, ErrInvalid
	}
	return s.Store.RecordObservation(ctx, p, key, id, c)
}

func (s Service) RecordAction(ctx context.Context, p identitydomain.Principal, key, id string, c RecordAction) (Action, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Action{}, false, ErrForbidden
	}
	if !validKey(key) || c.ActionType == "" || strings.TrimSpace(c.Narrative) == "" || c.PerformedAt.IsZero() {
		return Action{}, false, ErrInvalid
	}
	return s.Store.RecordAction(ctx, p, key, id, c)
}

func (s Service) RecordDecision(
	ctx context.Context,
	p identitydomain.Principal,
	key, id string,
	c RecordDecision,
) (Decision, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Decision{}, false, ErrForbidden
	}
	if !validKey(key) || c.ClientEventID == "" ||
		strings.TrimSpace(c.Decision) == "" || strings.TrimSpace(c.Rationale) == "" ||
		c.DecidedAt.IsZero() {
		return Decision{}, false, ErrInvalid
	}
	return s.Store.RecordDecision(ctx, p, key, id, c)
}

func (s Service) Complete(ctx context.Context, p identitydomain.Principal, key, id string, c CompleteExecution) (Execution, bool, error) {
	if !p.Has(identitydomain.PermissionExecutionWrite) {
		return Execution{}, false, ErrForbidden
	}
	if !validKey(key) || c.ExpectedVersion <= 0 || strings.TrimSpace(c.OutcomeSummary) == "" {
		return Execution{}, false, ErrInvalid
	}
	return s.Store.Complete(ctx, p, key, id, c)
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}

func validSyncMetadata(value SyncMetadata) bool {
	empty := value.ClientEventID == "" && value.DeviceID == "" &&
		value.CreatedAtDevice == nil && value.PayloadVersion == 0 &&
		value.BaseServerVersion == 0
	if empty {
		return true
	}
	return value.ClientEventID != "" && value.DeviceID != "" &&
		value.CreatedAtDevice != nil && value.PayloadVersion == 1 &&
		value.BaseServerVersion > 0
}
