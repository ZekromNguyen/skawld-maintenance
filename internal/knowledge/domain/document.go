package domain

import (
	"errors"
	"strings"
	"time"
)

type DocumentType string

const (
	DocumentOEMManual           DocumentType = "OEM_MANUAL"
	DocumentSOP                 DocumentType = "SOP"
	DocumentWorkInstruction     DocumentType = "WORK_INSTRUCTION"
	DocumentDatasheet           DocumentType = "DATASHEET"
	DocumentInspectionProcedure DocumentType = "INSPECTION_PROCEDURE"
	DocumentSafetyProcedure     DocumentType = "SAFETY_PROCEDURE"
	DocumentTechnicalBulletin   DocumentType = "TECHNICAL_BULLETIN"
)

type Authority string

const (
	AuthorityOEM             Authority = "OEM"
	AuthorityCorporate       Authority = "CORPORATE"
	AuthoritySiteApproved    Authority = "SITE_APPROVED"
	AuthorityRegulatory      Authority = "REGULATORY"
	AuthorityExpertReference Authority = "EXPERT_REFERENCE"
)

type ApprovalStatus string

const (
	ApprovalDraft          ApprovalStatus = "DRAFT"
	ApprovalApproved       ApprovalStatus = "APPROVED"
	ApprovalReviewRequired ApprovalStatus = "REVIEW_REQUIRED"
	ApprovalSuperseded     ApprovalStatus = "SUPERSEDED"
	ApprovalRetired        ApprovalStatus = "RETIRED"
)

type IngestionState string

const (
	IngestionAwaitingUpload IngestionState = "AWAITING_UPLOAD"
	IngestionQueued         IngestionState = "QUEUED"
	IngestionProcessing     IngestionState = "PROCESSING"
	IngestionReady          IngestionState = "READY"
	IngestionFailed         IngestionState = "FAILED"
)

var ErrInvalidDocument = errors.New("invalid document knowledge")

type Applicability struct {
	ID             string `json:"id,omitempty"`
	SiteID         string `json:"site_id,omitempty"`
	AssetID        string `json:"asset_id,omitempty"`
	AssetClass     string `json:"asset_class,omitempty"`
	Manufacturer   string `json:"manufacturer,omitempty"`
	Model          string `json:"model,omitempty"`
	ProcessService string `json:"process_service,omitempty"`
}

func (a Applicability) Valid() bool {
	return a.SiteID != "" || a.AssetID != "" || strings.TrimSpace(a.AssetClass) != "" ||
		strings.TrimSpace(a.Manufacturer) != "" || strings.TrimSpace(a.Model) != "" ||
		strings.TrimSpace(a.ProcessService) != ""
}

type Revision struct {
	ID             string          `json:"id"`
	DocumentID     string          `json:"document_id"`
	Revision       string          `json:"revision"`
	ApprovalStatus ApprovalStatus  `json:"approval_status"`
	EffectiveAt    *time.Time      `json:"effective_at,omitempty"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	SupersededByID string          `json:"superseded_by_id,omitempty"`
	AttachmentID   string          `json:"attachment_id,omitempty"`
	IngestionState IngestionState  `json:"ingestion_state"`
	IngestionError string          `json:"ingestion_error,omitempty"`
	ContentSHA256  string          `json:"content_sha256,omitempty"`
	Language       string          `json:"language"`
	ApprovedBy     string          `json:"approved_by,omitempty"`
	ApprovedAt     *time.Time      `json:"approved_at,omitempty"`
	Version        int64           `json:"version"`
	Applicability  []Applicability `json:"applicability"`
}

func (r Revision) Eligible(at time.Time) bool {
	if r.ApprovalStatus != ApprovalApproved || r.IngestionState != IngestionReady ||
		r.SupersededByID != "" {
		return false
	}
	if r.EffectiveAt != nil && at.Before(r.EffectiveAt.UTC()) {
		return false
	}
	return r.ExpiresAt == nil || at.Before(r.ExpiresAt.UTC())
}

type Document struct {
	ID              string       `json:"id"`
	OrganizationID  string       `json:"organization_id"`
	SiteID          string       `json:"site_id,omitempty"`
	Type            DocumentType `json:"document_type"`
	Title           string       `json:"title"`
	Authority       Authority    `json:"authority"`
	SourceReference string       `json:"source_reference,omitempty"`
	Revisions       []Revision   `json:"revisions"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

func ValidateDocument(document Document) error {
	if strings.TrimSpace(document.Title) == "" {
		return ErrInvalidDocument
	}
	switch document.Type {
	case DocumentOEMManual, DocumentSOP, DocumentWorkInstruction, DocumentDatasheet,
		DocumentInspectionProcedure, DocumentSafetyProcedure, DocumentTechnicalBulletin:
	default:
		return ErrInvalidDocument
	}
	switch document.Authority {
	case AuthorityOEM, AuthorityCorporate, AuthoritySiteApproved,
		AuthorityRegulatory, AuthorityExpertReference:
	default:
		return ErrInvalidDocument
	}
	return nil
}

func ValidateRevision(revision Revision) error {
	if strings.TrimSpace(revision.Revision) == "" ||
		strings.TrimSpace(revision.Language) == "" ||
		len(revision.Applicability) == 0 {
		return ErrInvalidDocument
	}
	if revision.EffectiveAt != nil && revision.ExpiresAt != nil &&
		!revision.ExpiresAt.After(*revision.EffectiveAt) {
		return ErrInvalidDocument
	}
	for _, applicability := range revision.Applicability {
		if !applicability.Valid() {
			return ErrInvalidDocument
		}
	}
	return nil
}
