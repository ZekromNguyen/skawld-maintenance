package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("workflow operation forbidden")
	ErrInvalid   = errors.New("invalid workflow command")
	ErrNotFound  = errors.New("workflow version not found")
	ErrConflict  = errors.New("workflow state conflict")
	ErrPolicy    = errors.New("workflow publication policy failed")
)

type Compile struct {
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	DemonstrationIDs []string `json:"demonstration_ids"`
}

type Applicability struct {
	ID                 string `json:"id,omitempty"`
	SiteID             string `json:"site_id,omitempty"`
	AssetID            string `json:"asset_id,omitempty"`
	AssetClass         string `json:"asset_class,omitempty"`
	Manufacturer       string `json:"manufacturer,omitempty"`
	Model              string `json:"model,omitempty"`
	ProcessService     string `json:"process_service,omitempty"`
	OperatingCondition string `json:"operating_condition,omitempty"`
	ValidationStatus   string `json:"validation_status"`
	ApprovedBy         string `json:"approved_by,omitempty"`
	ApprovedAt         string `json:"approved_at,omitempty"`
}

type Review struct {
	Decision             string          `json:"decision"`
	Reason               string          `json:"reason"`
	Applicability        []Applicability `json:"applicability"`
	Prerequisites        []string        `json:"prerequisites,omitempty"`
	RequiredCompetencies []string        `json:"required_competencies,omitempty"`
	EffectiveAt          time.Time       `json:"effective_at"`
	ReviewAt             time.Time       `json:"review_at"`
}

type Publish struct {
	Reason string `json:"reason"`
}

type ExpandApplicability struct {
	Reason        string        `json:"reason"`
	Applicability Applicability `json:"applicability"`
}

type Retire struct {
	Reason string `json:"reason"`
}

type WorkflowStep struct {
	ID        string         `json:"id"`
	Name      string         `json:"name,omitempty"`
	Kind      string         `json:"kind"`
	ToolName  string         `json:"tool_name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Evidence  []Evidence     `json:"evidence"`
}

type Evidence struct {
	DemonstrationID string   `json:"demonstration_id"`
	EventIDs        []string `json:"event_ids"`
}

type ReviewRecord struct {
	ID              string    `json:"id"`
	CandidateDigest string    `json:"candidate_digest"`
	Decision        string    `json:"decision"`
	Reason          string    `json:"reason"`
	ReviewedBy      string    `json:"reviewed_by"`
	ReviewedAt      time.Time `json:"reviewed_at"`
}

type Evaluation struct {
	ID          string         `json:"id"`
	SuiteName   string         `json:"suite_name"`
	GatesPassed bool           `json:"gates_passed"`
	Metrics     map[string]any `json:"metrics"`
	CompletedAt time.Time      `json:"completed_at"`
}

type ImprovementCandidate struct {
	ID                string         `json:"id"`
	DemonstrationID   string         `json:"demonstration_id"`
	CorrectionEventID string         `json:"correction_event_id"`
	CorrectedEventID  string         `json:"corrected_event_id"`
	CorrectedAction   string         `json:"corrected_action"`
	Reason            string         `json:"reason,omitempty"`
	Context           map[string]any `json:"context"`
	Outcome           map[string]any `json:"outcome"`
	Status            string         `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
}

type Version struct {
	WorkflowID           string                 `json:"workflow_id"`
	WorkflowKey          string                 `json:"workflow_key"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	Version              int                    `json:"version"`
	Status               string                 `json:"status"`
	SiteID               string                 `json:"site_id"`
	AssetClass           string                 `json:"asset_class"`
	CandidateDigest      string                 `json:"candidate_digest"`
	ToolCatalogDigest    string                 `json:"tool_catalog_digest"`
	SourceDemonstrations []string               `json:"source_demonstration_ids"`
	Steps                []WorkflowStep         `json:"steps"`
	Analysis             map[string]any         `json:"analysis"`
	BehavioralChanges    map[string]any         `json:"behavioral_changes"`
	Learning             map[string]any         `json:"learning"`
	Applicability        []Applicability        `json:"applicability"`
	Prerequisites        []string               `json:"prerequisites"`
	RequiredCompetencies []string               `json:"required_competencies"`
	EffectiveAt          *time.Time             `json:"effective_at,omitempty"`
	ReviewAt             *time.Time             `json:"review_at,omitempty"`
	Reviews              []ReviewRecord         `json:"reviews"`
	Evaluations          []Evaluation           `json:"evaluations"`
	Improvements         []ImprovementCandidate `json:"improvement_candidates"`
	CreatedAt            time.Time              `json:"created_at"`
	PublishedAt          *time.Time             `json:"published_at,omitempty"`
	PublishedBy          string                 `json:"published_by,omitempty"`
}

type ListFilter struct {
	PageSize int
	Cursor   string
}

// cursorPayload is the three-part workflow version cursor: the sort key is
// (created_at, workflow_id, version) because created_at alone is not unique
// across versions of the same workflow.
type cursorPayload struct {
	T string `json:"t"`
	W string `json:"w"`
	V int    `json:"v"`
}

type Gateway interface {
	Compile(context.Context, identitydomain.Principal, Compile) (Version, error)
	Get(context.Context, identitydomain.Principal, string, int) (Version, error)
	List(context.Context, identitydomain.Principal, ListFilter) ([]Version, bool, error)
	Review(context.Context, identitydomain.Principal, string, int, Review) (Version, error)
	Publish(context.Context, identitydomain.Principal, string, int, Publish) (Version, error)
	ExpandApplicability(context.Context, identitydomain.Principal, string, int, ExpandApplicability) (Version, error)
	Retire(context.Context, identitydomain.Principal, string, int, Retire) (Version, error)
	Applicable(context.Context, identitydomain.Principal, string) ([]Version, error)
}

type AuthorityReader interface {
	ForSubject(context.Context, string, string, string) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Gateway     Gateway
	Authorities AuthorityReader
	Now         func() time.Time
}

func (s Service) Compile(
	ctx context.Context,
	principal identitydomain.Principal,
	command Compile,
) (Version, error) {
	if !principal.Has(identitydomain.PermissionWorkflowReview) {
		return Version{}, ErrForbidden
	}
	command.Name = strings.TrimSpace(command.Name)
	command.Description = strings.TrimSpace(command.Description)
	command.DemonstrationIDs = uniqueStrings(command.DemonstrationIDs)
	if command.Name == "" || len(command.Name) > 200 ||
		len(command.Description) > 2_000 || len(command.DemonstrationIDs) < 2 ||
		len(command.DemonstrationIDs) > 20 {
		return Version{}, ErrInvalid
	}
	return s.Gateway.Compile(ctx, principal, command)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
) (Version, error) {
	if !principal.Has(identitydomain.PermissionWorkflowRead) {
		return Version{}, ErrForbidden
	}
	if workflowID == "" || version < 1 {
		return Version{}, ErrInvalid
	}
	return s.Gateway.Get(ctx, principal, workflowID, version)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter ListFilter,
) ([]Version, string, error) {
	if !principal.Has(identitydomain.PermissionWorkflowRead) {
		return nil, "", ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Cursor != "" {
		createdAt, workflowID, version, err := decodeWorkflowCursor(filter.Cursor)
		if err != nil {
			return nil, "", ErrInvalid
		}
		filter.Cursor = createdAt.UTC().Format(time.RFC3339Nano) + "|" + workflowID + "|" + strconv.Itoa(version)
	}
	items, hasMore, err := s.Gateway.List(ctx, principal, filter)
	if err != nil {
		return nil, "", err
	}
	if !hasMore || len(items) == 0 {
		return items, "", nil
	}
	last := items[len(items)-1]
	return items, encodeWorkflowCursor(last.CreatedAt, last.WorkflowID, last.Version), nil
}

// encodeWorkflowCursor packs the version sort key (created_at, workflow_id,
// version) into an opaque cursor. A two-part cursor would silently skip rows
// when two versions share the same created_at and workflow_id.
func encodeWorkflowCursor(createdAt time.Time, workflowID string, version int) string {
	payload, _ := json.Marshal(cursorPayload{
		T: createdAt.UTC().Format(time.RFC3339Nano),
		W: workflowID,
		V: version,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeWorkflowCursor(raw string) (time.Time, string, int, error) {
	payloadBytes, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", 0, err
	}
	var payload cursorPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return time.Time{}, "", 0, err
	}
	if payload.T == "" || payload.W == "" || payload.V < 1 {
		return time.Time{}, "", 0, errors.New("workflow cursor is incomplete")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.T)
	if err != nil {
		return time.Time{}, "", 0, err
	}
	if _, err := uuid.Parse(payload.W); err != nil {
		return time.Time{}, "", 0, err
	}
	return createdAt, payload.W, payload.V, nil
}

func (s Service) Review(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command Review,
) (Version, error) {
	if !principal.Has(identitydomain.PermissionWorkflowReview) {
		return Version{}, ErrForbidden
	}
	command.Decision = strings.ToUpper(strings.TrimSpace(command.Decision))
	command.Reason = strings.TrimSpace(command.Reason)
	if workflowID == "" || version < 1 || command.Reason == "" ||
		len(command.Reason) > 2_000 {
		return Version{}, ErrInvalid
	}
	switch command.Decision {
	case "APPROVED":
		if len(command.Applicability) == 0 || command.EffectiveAt.IsZero() ||
			command.ReviewAt.IsZero() ||
			!command.ReviewAt.After(command.EffectiveAt) ||
			!command.ReviewAt.After(s.now()) {
			return Version{}, ErrInvalid
		}
	case "REJECTED", "REVIEW_REQUIRED":
	default:
		return Version{}, ErrInvalid
	}
	if !validApplicability(command.Applicability) ||
		!validTerms(command.Prerequisites, 20) ||
		!validTerms(command.RequiredCompetencies, 20) {
		return Version{}, ErrInvalid
	}
	current, err := s.Gateway.Get(ctx, principal, workflowID, version)
	if err != nil {
		return Version{}, err
	}
	if current.Status != "CANDIDATE" && current.Status != "REVIEW_REQUIRED" {
		return Version{}, ErrConflict
	}
	for _, applicability := range command.Applicability {
		if applicability.SiteID != "" && applicability.SiteID != current.SiteID ||
			applicability.AssetClass != "" &&
				applicability.AssetClass != current.AssetClass {
			return Version{}, ErrInvalid
		}
	}
	if allowed, err := s.canApprove(
		ctx, principal, current, identitydomain.PermissionWorkflowReview,
		identitydomain.RiskAdvisory,
	); err != nil || !allowed {
		if err != nil {
			return Version{}, err
		}
		return Version{}, ErrForbidden
	}
	return s.Gateway.Review(ctx, principal, workflowID, version, command)
}

func (s Service) Publish(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command Publish,
) (Version, error) {
	command.Reason = strings.TrimSpace(command.Reason)
	if !principal.Has(identitydomain.PermissionWorkflowPublish) {
		return Version{}, ErrForbidden
	}
	if workflowID == "" || version < 1 || command.Reason == "" ||
		len(command.Reason) > 2_000 {
		return Version{}, ErrInvalid
	}
	current, err := s.Gateway.Get(ctx, principal, workflowID, version)
	if err != nil {
		return Version{}, err
	}
	if current.Status != "APPROVED" {
		return Version{}, ErrConflict
	}
	if current.EffectiveAt == nil || current.ReviewAt == nil ||
		!current.ReviewAt.After(s.now()) || len(current.Applicability) == 0 {
		return Version{}, ErrPolicy
	}
	if allowed, err := s.canApprove(
		ctx, principal, current, identitydomain.PermissionWorkflowPublish,
		identitydomain.RiskOperationalLow,
	); err != nil || !allowed {
		if err != nil {
			return Version{}, err
		}
		return Version{}, ErrForbidden
	}
	return s.Gateway.Publish(ctx, principal, workflowID, version, command)
}

func (s Service) ExpandApplicability(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command ExpandApplicability,
) (Version, error) {
	command.Reason = strings.TrimSpace(command.Reason)
	if !principal.Has(identitydomain.PermissionWorkflowPublish) {
		return Version{}, ErrForbidden
	}
	if workflowID == "" || version < 1 || command.Reason == "" ||
		command.Applicability.SiteID == "" ||
		strings.TrimSpace(command.Applicability.AssetClass) == "" ||
		!validApplicability([]Applicability{command.Applicability}) {
		return Version{}, ErrInvalid
	}
	current, err := s.Gateway.Get(ctx, principal, workflowID, version)
	if err != nil {
		return Version{}, err
	}
	if current.Status != "PUBLISHED" {
		return Version{}, ErrConflict
	}
	authorityTarget := current
	authorityTarget.SiteID = command.Applicability.SiteID
	authorityTarget.AssetClass = command.Applicability.AssetClass
	if allowed, err := s.canApprove(
		ctx, principal, authorityTarget, identitydomain.PermissionWorkflowPublish,
		identitydomain.RiskOperationalLow,
	); err != nil || !allowed {
		if err != nil {
			return Version{}, err
		}
		return Version{}, ErrForbidden
	}
	return s.Gateway.ExpandApplicability(
		ctx, principal, workflowID, version, command,
	)
}

func (s Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s Service) Retire(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command Retire,
) (Version, error) {
	command.Reason = strings.TrimSpace(command.Reason)
	if !principal.Has(identitydomain.PermissionWorkflowPublish) {
		return Version{}, ErrForbidden
	}
	if workflowID == "" || version < 1 || command.Reason == "" {
		return Version{}, ErrInvalid
	}
	current, err := s.Gateway.Get(ctx, principal, workflowID, version)
	if err != nil {
		return Version{}, err
	}
	if current.Status != "PUBLISHED" {
		return Version{}, ErrConflict
	}
	if allowed, err := s.canApprove(
		ctx, principal, current, identitydomain.PermissionWorkflowPublish,
		identitydomain.RiskOperationalLow,
	); err != nil || !allowed {
		if err != nil {
			return Version{}, err
		}
		return Version{}, ErrForbidden
	}
	return s.Gateway.Retire(ctx, principal, workflowID, version, command)
}

func (s Service) Applicable(
	ctx context.Context,
	principal identitydomain.Principal,
	assetID string,
) ([]Version, error) {
	if !principal.Has(identitydomain.PermissionWorkflowRead) {
		return nil, ErrForbidden
	}
	if assetID == "" {
		return nil, ErrInvalid
	}
	return s.Gateway.Applicable(ctx, principal, assetID)
}

func (s Service) canApprove(
	ctx context.Context,
	principal identitydomain.Principal,
	version Version,
	permission identitydomain.Permission,
	risk identitydomain.RiskLevel,
) (bool, error) {
	if s.Authorities == nil {
		return false, nil
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, principal.OrganizationID, version.SiteID,
	)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	return identitydomain.CanApprove(
		principal, permission, authorities, identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID,
			SiteID:         version.SiteID,
			ScopeKind:      "workflow",
			ScopeID:        version.WorkflowID,
			Competency:     version.AssetClass,
			Risk:           risk,
			At:             now,
		},
	), nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func validTerms(values []string, maximum int) bool {
	if len(values) > maximum {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > 200 {
			return false
		}
	}
	return true
}

func validApplicability(values []Applicability) bool {
	for index := range values {
		value := &values[index]
		value.ValidationStatus = strings.ToUpper(
			strings.TrimSpace(value.ValidationStatus),
		)
		switch value.ValidationStatus {
		case "VALIDATED", "LIKELY_APPLICABLE", "NOT_VALIDATED", "NOT_APPLICABLE":
		default:
			return false
		}
		if value.SiteID == "" && value.AssetID == "" &&
			strings.TrimSpace(value.AssetClass) == "" &&
			strings.TrimSpace(value.Manufacturer) == "" &&
			strings.TrimSpace(value.Model) == "" &&
			strings.TrimSpace(value.ProcessService) == "" {
			return false
		}
	}
	return true
}
