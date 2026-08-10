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
	ErrForbidden = errors.New("report operation forbidden")
	ErrInvalid   = errors.New("invalid report command")
	ErrNotFound  = errors.New("report not found")
	ErrConflict  = errors.New("report state conflict")
)

const (
	reportSchema = "maintenance.report.v1"
	reportPrompt = "maintenance.report.evidence-first.v1"
)

type Content struct {
	Summary             string   `json:"summary"`
	Measurements        []string `json:"measurements"`
	Observations        []string `json:"observations"`
	Actions             []string `json:"actions"`
	Outcome             string   `json:"outcome"`
	EvidenceIDs         []string `json:"evidence_ids"`
	Unknowns            []string `json:"unknowns"`
	RequiresHumanReview bool     `json:"requires_human_review"`
}

type Report struct {
	ID             string                     `json:"id"`
	OrganizationID string                     `json:"organization_id"`
	SiteID         string                     `json:"site_id"`
	ExecutionID    string                     `json:"execution_id"`
	Revision       int                        `json:"revision"`
	Version        int64                      `json:"version"`
	State          string                     `json:"state"`
	Content        Content                    `json:"structured_content"`
	Evidence       []knowledgedomain.Evidence `json:"evidence"`
	Provider       string                     `json:"provider,omitempty"`
	Model          string                     `json:"model,omitempty"`
	ModelVersion   string                     `json:"model_version,omitempty"`
	PromptVersion  string                     `json:"prompt_version,omitempty"`
	InputSHA256    string                     `json:"input_sha256,omitempty"`
	OutputSHA256   string                     `json:"output_sha256,omitempty"`
	GeneratedBy    string                     `json:"generated_by_kind"`
	SubmittedBy    string                     `json:"submitted_by,omitempty"`
	SubmittedAt    *time.Time                 `json:"submitted_at,omitempty"`
	ApprovedBy     string                     `json:"approved_by,omitempty"`
	ApprovedAt     *time.Time                 `json:"approved_at,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type ExecutionContext struct {
	OrganizationID string                     `json:"organization_id"`
	SiteID         string                     `json:"site_id"`
	ExecutionID    string                     `json:"execution_id"`
	AssetID        string                     `json:"asset_id"`
	Summary        string                     `json:"summary"`
	Snapshot       json.RawMessage            `json:"snapshot"`
	Facts          []knowledgedomain.Evidence `json:"facts"`
}

type Edit struct {
	ExpectedVersion int64   `json:"expected_version"`
	Content         Content `json:"structured_content"`
}

type Transition struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type ReportFilter struct {
	SiteID   string
	States   []string
	Cursor   string
	PageSize int
}

type ListResult struct {
	Items      []Report
	NextCursor string
	HasMore    bool
}

type Searcher interface {
	Search(context.Context, identitydomain.Principal, knowledgedomain.SearchQuery) (knowledgedomain.SearchResult, error)
}

type Store interface {
	LoadExecutionContext(context.Context, identitydomain.Principal, string) (ExecutionContext, error)
	SaveDraft(context.Context, identitydomain.Principal, string, Report, json.RawMessage) (Report, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Report, error)
	List(context.Context, identitydomain.Principal, ReportFilter) ([]Report, bool, error)
	Edit(context.Context, identitydomain.Principal, string, string, Edit) (Report, bool, error)
	Submit(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error)
	Approve(context.Context, identitydomain.Principal, string, string, Transition) (Report, bool, error)
	RecordCall(context.Context, identitydomain.Principal, ExecutionContext, skawld.Generation, string, string) error
}

type AuthorityReader interface {
	ForSubject(context.Context, string, string, string) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Store       Store
	Search      Searcher
	Router      skawld.Router
	Authorities AuthorityReader
	Now         func() time.Time
}

func (s Service) Draft(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
) (Report, bool, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) {
		return Report{}, false, ErrForbidden
	}
	if len(strings.TrimSpace(key)) < 8 || len(key) > 200 {
		return Report{}, false, ErrInvalid
	}
	execution, err := s.Store.LoadExecutionContext(ctx, principal, executionID)
	if err != nil {
		return Report{}, false, err
	}
	retrieval, err := s.Search.Search(ctx, principal, knowledgedomain.SearchQuery{
		SiteID: execution.SiteID, AssetID: execution.AssetID,
		Query: execution.Summary, Limit: 6,
	})
	if err != nil {
		return Report{}, false, err
	}
	evidence := append([]knowledgedomain.Evidence{}, execution.Facts...)
	evidence = append(evidence, retrieval.Items...)
	packet := toPacket(evidence)
	generation, err := s.Router.Generate(ctx, skawld.GenerateRequest{
		Capability:    skawld.CapabilityReportDraft,
		SchemaVersion: reportSchema, PromptVersion: reportPrompt,
		Context: execution.Snapshot, Evidence: packet,
	})
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, execution, generation, "PROVIDER_ERROR", "PROVIDER_ERROR")
		return Report{}, false, err
	}
	content, err := decodeContent(generation.Output, packet)
	if err != nil {
		_ = s.Store.RecordCall(ctx, principal, execution, generation, "INVALID_OUTPUT", "INVALID_REPORT")
		return Report{}, false, err
	}
	if err := s.Store.RecordCall(ctx, principal, execution, generation, "SUCCEEDED", ""); err != nil {
		return Report{}, false, err
	}
	return s.Store.SaveDraft(ctx, principal, key, Report{
		OrganizationID: execution.OrganizationID, SiteID: execution.SiteID,
		ExecutionID: execution.ExecutionID, State: "DRAFT", Content: content,
		Evidence: evidence, Provider: generation.Metadata.Provider,
		Model: generation.Metadata.Model, ModelVersion: generation.Metadata.ModelVersion,
		PromptVersion: generation.Prompt, InputSHA256: generation.InputHash,
		OutputSHA256: generation.OutputHash, GeneratedBy: "AI_DRAFT",
	}, execution.Snapshot)
}

func (s Service) Edit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command Edit,
) (Report, bool, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) {
		return Report{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 ||
		validateHumanContent(command.Content) != nil {
		return Report{}, false, ErrInvalid
	}
	report, err := s.Store.Get(ctx, principal, reportID)
	if err != nil {
		return Report{}, false, err
	}
	if err := validateEvidenceIDs(
		command.Content.EvidenceIDs, toPacket(report.Evidence),
	); err != nil {
		return Report{}, false, ErrInvalid
	}
	return s.Store.Edit(ctx, principal, key, reportID, command)
}

func (s Service) Submit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command Transition,
) (Report, bool, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) {
		return Report{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Report{}, false, ErrInvalid
	}
	return s.Store.Submit(ctx, principal, key, reportID, command)
}

func (s Service) Approve(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command Transition,
) (Report, bool, error) {
	if !principal.Has(identitydomain.PermissionReportApprove) {
		return Report{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Report{}, false, ErrInvalid
	}
	report, err := s.Store.Get(ctx, principal, reportID)
	if err != nil {
		return Report{}, false, err
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, principal.OrganizationID, report.SiteID,
	)
	if err != nil {
		return Report{}, false, err
	}
	if !identitydomain.CanApprove(
		principal, identitydomain.PermissionReportApprove, authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID, SiteID: report.SiteID,
			ScopeKind: "REPORT", ScopeID: reportID,
			Risk: identitydomain.RiskOperationalLow, At: s.now(),
		},
	) {
		return Report{}, false, ErrForbidden
	}
	return s.Store.Approve(ctx, principal, key, reportID, command)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	reportID string,
) (Report, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) &&
		!principal.Has(identitydomain.PermissionReportApprove) {
		return Report{}, ErrForbidden
	}
	return s.Store.Get(ctx, principal, reportID)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter ReportFilter,
) (ListResult, error) {
	if !principal.Has(identitydomain.PermissionReportWrite) &&
		!principal.Has(identitydomain.PermissionReportApprove) {
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
		case "DRAFT", "SUBMITTED", "APPROVED", "REJECTED":
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
		result.NextCursor = keyset.Encode(last.CreatedAt, last.ID)
	}
	return result, nil
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
	if !content.RequiresHumanReview || strings.TrimSpace(content.Summary) == "" {
		return Content{}, skawld.ErrInvalidOutput
	}
	if err := validateEvidenceIDs(content.EvidenceIDs, evidence); err != nil {
		return Content{}, err
	}
	return content, nil
}

func validateHumanContent(content Content) error {
	if strings.TrimSpace(content.Summary) == "" || len(content.Summary) > 5000 ||
		len(content.Measurements) > 100 || len(content.Observations) > 100 ||
		len(content.Actions) > 100 || len(content.Unknowns) > 50 {
		return ErrInvalid
	}
	return nil
}

func validateEvidenceIDs(ids []string, evidence []skawld.Evidence) error {
	eligible := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		eligible[item.ID] = struct{}{}
	}
	for _, evidenceID := range ids {
		if _, ok := eligible[evidenceID]; !ok {
			return errors.New("report cited evidence outside the eligible packet")
		}
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

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validKey(value string) bool {
	length := len(strings.TrimSpace(value))
	return length >= 8 && length <= 200
}
