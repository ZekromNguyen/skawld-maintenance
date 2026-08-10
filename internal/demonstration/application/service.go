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
	ErrForbidden      = errors.New("demonstration operation forbidden")
	ErrInvalid        = errors.New("invalid demonstration command")
	ErrNotFound       = errors.New("demonstration not found")
	ErrConflict       = errors.New("demonstration state conflict")
	ErrCapturePending = errors.New("semantic capture has pending or failed deliveries")
)

type SubjectKind string

const (
	SubjectExecution SubjectKind = "EXECUTION"
	SubjectHandover  SubjectKind = "HANDOVER"
)

type Start struct {
	SubjectKind SubjectKind `json:"subject_kind"`
	SubjectID   string      `json:"subject_id"`
}

type Complete struct {
	Outcome string `json:"outcome"`
}

type RecordEvidenceView struct {
	EvidenceID string `json:"evidence_id"`
	Intent     string `json:"intent,omitempty"`
}

type Review struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type RedactEvent struct {
	JSONPath string `json:"json_path"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

type Event struct {
	ID            string         `json:"id"`
	Ordinal       int            `json:"ordinal"`
	SchemaVersion string         `json:"schema_version"`
	Timestamp     time.Time      `json:"timestamp"`
	ActorID       string         `json:"actor_id"`
	Roles         []string       `json:"roles"`
	Source        string         `json:"source"`
	Trust         string         `json:"trust"`
	Sensitivity   string         `json:"sensitivity"`
	Application   string         `json:"application,omitempty"`
	Action        string         `json:"action"`
	Intent        string         `json:"intent,omitempty"`
	Entity        map[string]any `json:"entity,omitempty"`
	Input         map[string]any `json:"input,omitempty"`
	Output        map[string]any `json:"output,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
	Decision      map[string]any `json:"decision,omitempty"`
	Result        map[string]any `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
	CorrectionOf  string         `json:"correction_of,omitempty"`
	ApprovalID    string         `json:"approval_id,omitempty"`
	DomainEventID string         `json:"domain_event_id,omitempty"`
	Provenance    map[string]any `json:"provenance"`
	Redactions    []Redaction    `json:"redactions,omitempty"`
}

type Redaction struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	JSONPath    string    `json:"json_path"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`
	RequestedAt time.Time `json:"requested_at"`
}

type ReviewRecord struct {
	ID         string    `json:"id"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	ReviewedBy string    `json:"reviewed_by"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

type CaptureHealth struct {
	Pending    int       `json:"pending"`
	Processing int       `json:"processing"`
	Failed     int       `json:"failed"`
	Applied    int       `json:"applied"`
	LastError  string    `json:"last_error,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

type Demonstration struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	SiteID         string         `json:"site_id"`
	SubjectKind    SubjectKind    `json:"subject_kind"`
	SubjectID      string         `json:"subject_id"`
	WorkflowKey    string         `json:"workflow_key"`
	SchemaVersion  string         `json:"schema_version"`
	SessionID      string         `json:"session_id"`
	Status         string         `json:"status"`
	ReviewStatus   string         `json:"review_status"`
	InitialContext map[string]any `json:"initial_context"`
	FinalResult    map[string]any `json:"final_result,omitempty"`
	Events         []Event        `json:"events"`
	Reviews        []ReviewRecord `json:"reviews"`
	Capture        CaptureHealth  `json:"capture"`
	StartedAt      time.Time      `json:"started_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	CreatedBy      string         `json:"created_by"`
}

type ListFilter struct {
	PageSize int
	Cursor   string
}

type Gateway interface {
	Start(context.Context, identitydomain.Principal, Start) (Demonstration, error)
	Get(context.Context, identitydomain.Principal, string) (Demonstration, error)
	List(context.Context, identitydomain.Principal, string, ListFilter) ([]Demonstration, bool, error)
	Complete(context.Context, identitydomain.Principal, string, Complete) (Demonstration, error)
	RecordEvidenceView(context.Context, identitydomain.Principal, string, RecordEvidenceView) (Event, error)
	Review(context.Context, identitydomain.Principal, string, Review) (ReviewRecord, error)
	RedactEvent(context.Context, identitydomain.Principal, string, string, RedactEvent) (Redaction, error)
}

type AuthorityReader interface {
	ForSubject(context.Context, string, string, string) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Gateway     Gateway
	Authorities AuthorityReader
	Now         func() time.Time
}

func (s Service) Start(
	ctx context.Context,
	principal identitydomain.Principal,
	command Start,
) (Demonstration, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationCapture) {
		return Demonstration{}, ErrForbidden
	}
	if command.SubjectID == "" ||
		(command.SubjectKind != SubjectExecution && command.SubjectKind != SubjectHandover) {
		return Demonstration{}, ErrInvalid
	}
	return s.Gateway.Start(ctx, principal, command)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
) (Demonstration, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationRead) || id == "" {
		return Demonstration{}, ErrForbidden
	}
	return s.Gateway.Get(ctx, principal, id)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	siteID string,
	filter ListFilter,
) ([]Demonstration, string, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationRead) {
		return nil, "", ErrForbidden
	}
	if siteID != "" && !principal.CanAccessSite(principal.OrganizationID, siteID) {
		return nil, "", ErrForbidden
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
	items, hasMore, err := s.Gateway.List(ctx, principal, siteID, filter)
	if err != nil {
		return nil, "", err
	}
	if !hasMore || len(items) == 0 {
		return items, "", nil
	}
	last := items[len(items)-1]
	return items, keyset.Encode(last.StartedAt, last.ID), nil
}

func (s Service) Complete(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
	command Complete,
) (Demonstration, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationCapture) {
		return Demonstration{}, ErrForbidden
	}
	command.Outcome = strings.TrimSpace(command.Outcome)
	if id == "" || command.Outcome == "" || len(command.Outcome) > 2_000 {
		return Demonstration{}, ErrInvalid
	}
	return s.Gateway.Complete(ctx, principal, id, command)
}

func (s Service) RecordEvidenceView(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
	command RecordEvidenceView,
) (Event, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationCapture) ||
		!principal.Has(identitydomain.PermissionKnowledgeRead) {
		return Event{}, ErrForbidden
	}
	command.EvidenceID = strings.TrimSpace(command.EvidenceID)
	command.Intent = strings.TrimSpace(command.Intent)
	if id == "" || command.EvidenceID == "" || len(command.EvidenceID) > 256 ||
		len(command.Intent) > 500 {
		return Event{}, ErrInvalid
	}
	return s.Gateway.RecordEvidenceView(ctx, principal, id, command)
}

func (s Service) Review(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
	command Review,
) (ReviewRecord, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationReview) {
		return ReviewRecord{}, ErrForbidden
	}
	command.Decision = strings.ToUpper(strings.TrimSpace(command.Decision))
	command.Reason = strings.TrimSpace(command.Reason)
	switch command.Decision {
	case "APPROVED", "REJECTED", "REDACTION_REQUIRED":
	default:
		return ReviewRecord{}, ErrInvalid
	}
	if id == "" || command.Reason == "" || len(command.Reason) > 2_000 {
		return ReviewRecord{}, ErrInvalid
	}
	demo, err := s.Gateway.Get(ctx, principal, id)
	if err != nil {
		return ReviewRecord{}, err
	}
	if demo.Status != "completed" {
		return ReviewRecord{}, ErrConflict
	}
	allowed, err := s.canGovern(ctx, principal, demo)
	if err != nil {
		return ReviewRecord{}, err
	}
	if !allowed {
		return ReviewRecord{}, ErrForbidden
	}
	return s.Gateway.Review(ctx, principal, id, command)
}

func (s Service) RedactEvent(
	ctx context.Context,
	principal identitydomain.Principal,
	id, eventID string,
	command RedactEvent,
) (Redaction, error) {
	if !principal.Has(identitydomain.PermissionDemonstrationReview) {
		return Redaction{}, ErrForbidden
	}
	command.JSONPath = strings.TrimSpace(command.JSONPath)
	command.Action = strings.ToUpper(strings.TrimSpace(command.Action))
	command.Reason = strings.TrimSpace(command.Reason)
	if command.Action != "MASK" && command.Action != "DROP" {
		return Redaction{}, ErrInvalid
	}
	if id == "" || eventID == "" || !validRedactionPath(command.JSONPath) ||
		command.Reason == "" || len(command.Reason) > 2_000 {
		return Redaction{}, ErrInvalid
	}
	demo, err := s.Gateway.Get(ctx, principal, id)
	if err != nil {
		return Redaction{}, err
	}
	allowed, err := s.canGovern(ctx, principal, demo)
	if err != nil {
		return Redaction{}, err
	}
	if !allowed {
		return Redaction{}, ErrForbidden
	}
	return s.Gateway.RedactEvent(ctx, principal, id, eventID, command)
}

func (s Service) canGovern(
	ctx context.Context,
	principal identitydomain.Principal,
	demo Demonstration,
) (bool, error) {
	if s.Authorities == nil {
		return false, nil
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, demo.OrganizationID, demo.SiteID,
	)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	return identitydomain.CanApprove(
		principal,
		identitydomain.PermissionDemonstrationReview,
		authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: demo.OrganizationID,
			SiteID:         demo.SiteID,
			ScopeKind:      "demonstration",
			ScopeID:        demo.ID,
			Risk:           identitydomain.RiskAdvisory,
			At:             now,
		},
	), nil
}

func validRedactionPath(path string) bool {
	if path == "" || len(path) > 512 || strings.ContainsAny(path, "\r\n\x00") {
		return false
	}
	parts := strings.Split(path, ".")
	switch parts[0] {
	case "input", "output", "context", "decision", "result":
	default:
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return len(parts) >= 2
}
