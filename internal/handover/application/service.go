package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

var (
	ErrForbidden = errors.New("handover operation forbidden")
	ErrInvalid   = errors.New("invalid handover command")
	ErrNotFound  = errors.New("handover not found")
	ErrConflict  = errors.New("handover state conflict")
)

const (
	handoverSchema = "maintenance.shift_handover.v1"
	handoverPrompt = "maintenance.shift_handover.evidence-first.v1"
)

type ListItem struct {
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type Content struct {
	Summary             string     `json:"summary"`
	OpenIncidents       []ListItem `json:"open_incidents"`
	ActiveExecutions    []ListItem `json:"active_executions"`
	SafetyConcerns      []ListItem `json:"safety_concerns"`
	FollowUp            []ListItem `json:"follow_up"`
	EvidenceIDs         []string   `json:"evidence_ids"`
	Unknowns            []ListItem `json:"unknowns"`
	RequiresHumanReview bool       `json:"requires_human_review"`
}

type Handover struct {
	ID             string                     `json:"id"`
	OrganizationID string                     `json:"organization_id"`
	SiteID         string                     `json:"site_id"`
	ShiftStart     time.Time                  `json:"shift_start"`
	ShiftEnd       time.Time                  `json:"shift_end"`
	State          string                     `json:"state"`
	Content        Content                    `json:"structured_content"`
	Evidence       []knowledgedomain.Evidence `json:"evidence"`
	Provider       string                     `json:"provider"`
	Model          string                     `json:"model"`
	ModelVersion   string                     `json:"model_version"`
	PromptVersion  string                     `json:"prompt_version"`
	InputSHA256    string                     `json:"input_sha256"`
	OutputSHA256   string                     `json:"output_sha256"`
	Version        int64                      `json:"version"`
	CreatedBy      string                     `json:"created_by"`
	SubmittedBy    string                     `json:"submitted_by,omitempty"`
	SubmittedAt    *time.Time                 `json:"submitted_at,omitempty"`
	AcceptedBy     string                     `json:"accepted_by,omitempty"`
	AcceptedAt     *time.Time                 `json:"accepted_at,omitempty"`
	AcknowledgedBy string                     `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time                 `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type PrepareDraft struct {
	SiteID     string    `json:"site_id"`
	ShiftStart time.Time `json:"shift_start"`
	ShiftEnd   time.Time `json:"shift_end"`
}

type WindowContext struct {
	OrganizationID string                     `json:"organization_id"`
	SiteID         string                     `json:"site_id"`
	Snapshot       json.RawMessage            `json:"snapshot"`
	Evidence       []knowledgedomain.Evidence `json:"evidence"`
}

type Edit struct {
	ExpectedVersion int64   `json:"expected_version"`
	Content         Content `json:"structured_content"`
}

type Transition struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type HandoverFilter struct {
	SiteID   string
	States   []string
	Cursor   string
	PageSize int
}

type ListResult struct {
	Items      []Handover
	NextCursor string
	HasMore    bool
}

type Store interface {
	LoadWindowContext(context.Context, identitydomain.Principal, PrepareDraft) (WindowContext, error)
	SaveDraft(context.Context, identitydomain.Principal, string, Handover) (Handover, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Handover, error)
	List(context.Context, identitydomain.Principal, HandoverFilter) ([]Handover, bool, error)
	Edit(context.Context, identitydomain.Principal, string, string, Edit) (Handover, bool, error)
	Submit(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error)
	Accept(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error)
	Acknowledge(context.Context, identitydomain.Principal, string, string, Transition) (Handover, bool, error)
	RecordCall(context.Context, identitydomain.Principal, WindowContext, skawld.Generation, string, string) error
}

type AuthorityReader interface {
	ForSubject(context.Context, string, string, string) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Store       Store
	Router      skawld.Router
	Authorities AuthorityReader
	Now         func() time.Time
}

func (s Service) Prepare(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command PrepareDraft,
) (Handover, bool, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) ||
		!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
		return Handover{}, false, ErrForbidden
	}
	if !validKey(key) || command.ShiftStart.IsZero() || command.ShiftEnd.IsZero() ||
		!command.ShiftEnd.After(command.ShiftStart) ||
		command.ShiftEnd.Sub(command.ShiftStart) > 48*time.Hour {
		return Handover{}, false, ErrInvalid
	}
	window, err := s.Store.LoadWindowContext(ctx, principal, command)
	if err != nil {
		return Handover{}, false, err
	}
	packet := toPacket(window.Evidence)
	generation, err := s.Router.Generate(ctx, skawld.GenerateRequest{
		Capability: skawld.CapabilityShiftHandover, SchemaVersion: handoverSchema,
		PromptVersion: handoverPrompt, Context: window.Snapshot, Evidence: packet,
	})
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, window, generation, "PROVIDER_ERROR", "PROVIDER_ERROR")
		return Handover{}, false, err
	}
	content, err := decodeContent(generation.Output, packet)
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, window, generation, "INVALID_OUTPUT", "INVALID_HANDOVER")
		return Handover{}, false, err
	}
	if err := s.Store.RecordCall(ctx, principal, window, generation, "SUCCEEDED", ""); err != nil {
		return Handover{}, false, err
	}
	return s.Store.SaveDraft(ctx, principal, key, Handover{
		OrganizationID: window.OrganizationID, SiteID: window.SiteID,
		ShiftStart: command.ShiftStart.UTC(), ShiftEnd: command.ShiftEnd.UTC(),
		State: "DRAFT", Content: content, Evidence: window.Evidence,
		Provider: generation.Metadata.Provider, Model: generation.Metadata.Model,
		ModelVersion: generation.Metadata.ModelVersion, PromptVersion: generation.Prompt,
		InputSHA256: generation.InputHash, OutputSHA256: generation.OutputHash,
	})
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	handoverID string,
) (Handover, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) &&
		!principal.Has(identitydomain.PermissionHandoverAccept) {
		return Handover{}, ErrForbidden
	}
	return s.Store.Get(ctx, principal, handoverID)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter HandoverFilter,
) (ListResult, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) &&
		!principal.Has(identitydomain.PermissionHandoverAccept) {
		return ListResult{}, ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return ListResult{}, ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	for _, state := range filter.States {
		switch state {
		case "DRAFT", "SUBMITTED", "ACCEPTED", "ACKNOWLEDGED":
		default:
			return ListResult{}, ErrInvalid
		}
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, principal, filter)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Items: items, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		result.NextCursor = keyset.Encode(last.ShiftStart, last.ID)
	}
	return result, nil
}

func (s Service) Edit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID string,
	command Edit,
) (Handover, bool, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) {
		return Handover{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 ||
		validateContent(command.Content) != nil {
		return Handover{}, false, ErrInvalid
	}
	current, err := s.Store.Get(ctx, principal, handoverID)
	if err != nil {
		return Handover{}, false, err
	}
	if validateEvidenceIDs(command.Content.EvidenceIDs, toPacket(current.Evidence)) != nil {
		return Handover{}, false, ErrInvalid
	}
	return s.Store.Edit(ctx, principal, key, handoverID, command)
}

func (s Service) Submit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID string,
	command Transition,
) (Handover, bool, error) {
	if !principal.Has(identitydomain.PermissionHandoverWrite) {
		return Handover{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Handover{}, false, ErrInvalid
	}
	return s.Store.Submit(ctx, principal, key, handoverID, command)
}

func (s Service) Accept(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID string,
	command Transition,
) (Handover, bool, error) {
	if !principal.Has(identitydomain.PermissionHandoverAccept) {
		return Handover{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Handover{}, false, ErrInvalid
	}
	value, err := s.Store.Get(ctx, principal, handoverID)
	if err != nil {
		return Handover{}, false, err
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, principal.OrganizationID, value.SiteID,
	)
	if err != nil {
		return Handover{}, false, err
	}
	if !identitydomain.CanApprove(
		principal, identitydomain.PermissionHandoverAccept, authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID, SiteID: value.SiteID,
			ScopeKind: "HANDOVER", ScopeID: handoverID,
			Risk: identitydomain.RiskOperationalLow, At: s.now(),
		},
	) {
		return Handover{}, false, ErrForbidden
	}
	return s.Store.Accept(ctx, principal, key, handoverID, command)
}

func (s Service) Acknowledge(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID string,
	command Transition,
) (Handover, bool, error) {
	if !principal.Has(identitydomain.PermissionHandoverAccept) {
		return Handover{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Handover{}, false, ErrInvalid
	}
	return s.Store.Acknowledge(ctx, principal, key, handoverID, command)
}

// UnmarshalJSON accepts the legacy string form (normalized to Title) and the
// structured {title, detail, severity} form, including a JSON object that a
// provider serialized into a string.
func (item *ListItem) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		var embedded map[string]string
		if json.Unmarshal([]byte(s), &embedded) == nil && len(embedded) > 0 {
			item.Title = firstNonEmpty(embedded["summary"], embedded["title"], embedded["name"], s)
			item.Severity = embedded["severity"]
			return nil
		}
		item.Title = s
		return nil
	}
	type alias ListItem
	var plain alias
	if err := json.Unmarshal(raw, &plain); err != nil {
		return err
	}
	*item = ListItem(plain)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func decodeContent(raw json.RawMessage, evidence []skawld.Evidence) (Content, error) {
	var content Content
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return Content{}, errors.Join(skawld.ErrInvalidOutput, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Content{}, skawld.ErrInvalidOutput
	}
	if validateContent(content) != nil ||
		validateEvidenceIDs(content.EvidenceIDs, evidence) != nil {
		return Content{}, skawld.ErrInvalidOutput
	}
	return content, nil
}

func validateContent(content Content) error {
	if !content.RequiresHumanReview || strings.TrimSpace(content.Summary) == "" ||
		len(content.Summary) > 5000 || len(content.OpenIncidents) > 100 ||
		len(content.ActiveExecutions) > 100 || len(content.SafetyConcerns) > 100 ||
		len(content.FollowUp) > 100 || len(content.Unknowns) > 100 {
		return ErrInvalid
	}
	return nil
}

func validateEvidenceIDs(ids []string, evidence []skawld.Evidence) error {
	eligible := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		eligible[item.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(ids))
	for _, evidenceID := range ids {
		if _, ok := eligible[evidenceID]; !ok {
			return ErrInvalid
		}
		if _, duplicate := seen[evidenceID]; duplicate {
			return ErrInvalid
		}
		seen[evidenceID] = struct{}{}
	}
	return nil
}

func toPacket(evidence []knowledgedomain.Evidence) []skawld.Evidence {
	result := make([]skawld.Evidence, 0, len(evidence))
	for _, item := range evidence {
		result = append(result, skawld.Evidence{
			ID: item.ID, Kind: item.Kind, SourceID: item.SourceID,
			Revision: item.Revision, Locator: item.Locator,
			Authority: string(item.Authority), Content: item.Content,
		})
	}
	return result
}

func validKey(value string) bool {
	length := len(strings.TrimSpace(value))
	return length >= 8 && length <= 200
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
