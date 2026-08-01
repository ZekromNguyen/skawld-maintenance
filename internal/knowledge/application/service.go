package application

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
)

var (
	ErrForbidden    = errors.New("knowledge operation forbidden")
	ErrInvalid      = errors.New("invalid knowledge command")
	ErrNotFound     = errors.New("knowledge item not found")
	ErrConflict     = errors.New("knowledge state conflict")
	ErrNoAttachment = errors.New("available PDF attachment is required")
)

type CreateDocument struct {
	SiteID          string                       `json:"site_id"`
	DocumentType    knowledgedomain.DocumentType `json:"document_type"`
	Title           string                       `json:"title"`
	Authority       knowledgedomain.Authority    `json:"authority"`
	SourceReference string                       `json:"source_reference,omitempty"`
}

type CreateRevision struct {
	Revision      string                          `json:"revision"`
	EffectiveAt   *time.Time                      `json:"effective_at,omitempty"`
	ExpiresAt     *time.Time                      `json:"expires_at,omitempty"`
	Language      string                          `json:"language"`
	Applicability []knowledgedomain.Applicability `json:"applicability"`
}

type ApproveRevision struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type RetireRevision struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason"`
}

type RequestIngestion struct {
	AttachmentID string `json:"attachment_id"`
}

type Filter struct {
	SiteID string
}

type Store interface {
	CreateDocument(context.Context, identitydomain.Principal, string, CreateDocument) (knowledgedomain.Document, bool, error)
	CreateRevision(context.Context, identitydomain.Principal, string, string, CreateRevision) (knowledgedomain.Revision, bool, error)
	GetDocument(context.Context, identitydomain.Principal, string) (knowledgedomain.Document, error)
	ListDocuments(context.Context, identitydomain.Principal, Filter) ([]knowledgedomain.Document, error)
	ApproveRevision(context.Context, identitydomain.Principal, string, string, ApproveRevision) (knowledgedomain.Revision, bool, error)
	RetireRevision(context.Context, identitydomain.Principal, string, string, RetireRevision) (knowledgedomain.Revision, bool, error)
	RequestIngestion(context.Context, identitydomain.Principal, string, string, RequestIngestion) (knowledgedomain.Revision, bool, error)
	Search(context.Context, identitydomain.Principal, knowledgedomain.SearchQuery) (knowledgedomain.SearchResult, error)
	RevisionScope(context.Context, identitydomain.Principal, string) (string, string, error)
}

type AuthorityReader interface {
	ForSubject(context.Context, string, string, string) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Store       Store
	Authorities AuthorityReader
	Now         func() time.Time
}

func (s Service) CreateDocument(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command CreateDocument,
) (knowledgedomain.Document, bool, error) {
	value := knowledgedomain.Document{
		SiteID: command.SiteID, Type: command.DocumentType,
		Title: strings.TrimSpace(command.Title), Authority: command.Authority,
		SourceReference: strings.TrimSpace(command.SourceReference),
	}
	if !principal.Has(identitydomain.PermissionKnowledgeWrite) ||
		!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
		return knowledgedomain.Document{}, false, ErrForbidden
	}
	if !validKey(key) || knowledgedomain.ValidateDocument(value) != nil {
		return knowledgedomain.Document{}, false, ErrInvalid
	}
	command.Title = value.Title
	command.SourceReference = value.SourceReference
	return s.Store.CreateDocument(ctx, principal, key, command)
}

func (s Service) CreateRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, documentID string,
	command CreateRevision,
) (knowledgedomain.Revision, bool, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeWrite) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	command.Revision = strings.TrimSpace(command.Revision)
	command.Language = strings.TrimSpace(command.Language)
	if command.Language == "" {
		command.Language = "en"
	}
	if !validKey(key) || knowledgedomain.ValidateRevision(knowledgedomain.Revision{
		Revision: command.Revision, EffectiveAt: command.EffectiveAt,
		ExpiresAt: command.ExpiresAt, Language: command.Language,
		Applicability: command.Applicability,
	}) != nil {
		return knowledgedomain.Revision{}, false, ErrInvalid
	}
	return s.Store.CreateRevision(ctx, principal, key, documentID, command)
}

func (s Service) ApproveRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command ApproveRevision,
) (knowledgedomain.Revision, bool, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeApprove) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	if !validKey(key) {
		return knowledgedomain.Revision{}, false, ErrInvalid
	}
	siteID, _, err := s.Store.RevisionScope(ctx, principal, revisionID)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, principal.OrganizationID, siteID,
	)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	if !identitydomain.CanApprove(
		principal, identitydomain.PermissionKnowledgeApprove, authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID,
			SiteID:         siteID,
			ScopeKind:      "DOCUMENT", ScopeID: revisionID,
			Risk: identitydomain.RiskAdvisory, At: s.now(),
		},
	) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	if command.ExpectedVersion <= 0 {
		return knowledgedomain.Revision{}, false, ErrInvalid
	}
	return s.Store.ApproveRevision(ctx, principal, key, revisionID, command)
}

func (s Service) RetireRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command RetireRevision,
) (knowledgedomain.Revision, bool, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeApprove) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 ||
		strings.TrimSpace(command.Reason) == "" {
		return knowledgedomain.Revision{}, false, ErrInvalid
	}
	siteID, _, err := s.Store.RevisionScope(ctx, principal, revisionID)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	authorities, err := s.Authorities.ForSubject(
		ctx, principal.ID, principal.OrganizationID, siteID,
	)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	if !identitydomain.CanApprove(
		principal, identitydomain.PermissionKnowledgeApprove, authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID, SiteID: siteID,
			ScopeKind: "DOCUMENT", ScopeID: revisionID,
			Risk: identitydomain.RiskAdvisory, At: s.now(),
		},
	) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	command.Reason = strings.TrimSpace(command.Reason)
	return s.Store.RetireRevision(ctx, principal, key, revisionID, command)
}

func (s Service) RequestIngestion(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command RequestIngestion,
) (knowledgedomain.Revision, bool, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeWrite) {
		return knowledgedomain.Revision{}, false, ErrForbidden
	}
	if !validKey(key) || strings.TrimSpace(command.AttachmentID) == "" {
		return knowledgedomain.Revision{}, false, ErrInvalid
	}
	return s.Store.RequestIngestion(ctx, principal, key, revisionID, command)
}

func (s Service) GetDocument(
	ctx context.Context,
	principal identitydomain.Principal,
	documentID string,
) (knowledgedomain.Document, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeRead) {
		return knowledgedomain.Document{}, ErrForbidden
	}
	return s.Store.GetDocument(ctx, principal, documentID)
}

func (s Service) ListDocuments(
	ctx context.Context,
	principal identitydomain.Principal,
	filter Filter,
) ([]knowledgedomain.Document, error) {
	if !principal.Has(identitydomain.PermissionKnowledgeRead) {
		return nil, ErrForbidden
	}
	if filter.SiteID != "" &&
		!principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return nil, ErrForbidden
	}
	return s.Store.ListDocuments(ctx, principal, filter)
}

func (s Service) Search(
	ctx context.Context,
	principal identitydomain.Principal,
	query knowledgedomain.SearchQuery,
) (knowledgedomain.SearchResult, error) {
	query.Query = strings.TrimSpace(query.Query)
	if !principal.Has(identitydomain.PermissionKnowledgeRead) ||
		!principal.CanAccessSite(principal.OrganizationID, query.SiteID) {
		return knowledgedomain.SearchResult{}, ErrForbidden
	}
	if query.Query == "" || len(query.Query) > 500 {
		return knowledgedomain.SearchResult{}, ErrInvalid
	}
	if query.Limit <= 0 || query.Limit > 20 {
		query.Limit = 8
	}
	return s.Store.Search(ctx, principal, query)
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
