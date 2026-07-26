package application

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

var (
	ErrForbidden   = errors.New("transcription operation forbidden")
	ErrInvalid     = errors.New("invalid transcription command")
	ErrNotFound    = errors.New("transcription not found")
	ErrConflict    = errors.New("transcription state conflict")
	ErrUnavailable = errors.New("transcription provider is not configured")
)

type Transcription struct {
	ID                 string     `json:"id"`
	OrganizationID     string     `json:"organization_id"`
	SiteID             string     `json:"site_id"`
	AttachmentID       string     `json:"attachment_id"`
	State              string     `json:"state"`
	Transcript         string     `json:"transcript,omitempty"`
	Language           string     `json:"language,omitempty"`
	Provider           string     `json:"provider"`
	Model              string     `json:"model"`
	ModelVersion       string     `json:"model_version"`
	Confidence         float64    `json:"confidence,omitempty"`
	VerificationStatus string     `json:"verification_status"`
	VerifiedBy         string     `json:"verified_by,omitempty"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	SourceSHA256       string     `json:"source_sha256"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Verify struct {
	ExpectedVersion int64  `json:"expected_version"`
	Transcript      string `json:"transcript"`
	Accept          bool   `json:"accept"`
}

type Store interface {
	Request(context.Context, identitydomain.Principal, string, string, skawld.ProviderMetadata) (Transcription, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Transcription, error)
	Verify(context.Context, identitydomain.Principal, string, string, Verify) (Transcription, bool, error)
}

type Service struct {
	Store    Store
	Provider skawld.TranscriptionProvider
}

func (s Service) Request(
	ctx context.Context,
	principal identitydomain.Principal,
	key, attachmentID string,
) (Transcription, bool, error) {
	if !principal.Has(identitydomain.PermissionAttachmentWrite) {
		return Transcription{}, false, ErrForbidden
	}
	if !validKey(key) || strings.TrimSpace(attachmentID) == "" {
		return Transcription{}, false, ErrInvalid
	}
	if s.Provider == nil || s.Provider.Model().Provider == "unavailable" {
		return Transcription{}, false, ErrUnavailable
	}
	return s.Store.Request(ctx, principal, key, attachmentID, s.Provider.Model())
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	transcriptionID string,
) (Transcription, error) {
	if !principal.Has(identitydomain.PermissionExecutionRead) {
		return Transcription{}, ErrForbidden
	}
	return s.Store.Get(ctx, principal, transcriptionID)
}

func (s Service) Verify(
	ctx context.Context,
	principal identitydomain.Principal,
	key, transcriptionID string,
	command Verify,
) (Transcription, bool, error) {
	if !principal.Has(identitydomain.PermissionExecutionWrite) {
		return Transcription{}, false, ErrForbidden
	}
	command.Transcript = strings.TrimSpace(command.Transcript)
	if !validKey(key) || command.ExpectedVersion <= 0 ||
		(command.Accept && command.Transcript == "") ||
		len(command.Transcript) > 50000 {
		return Transcription{}, false, ErrInvalid
	}
	return s.Store.Verify(ctx, principal, key, transcriptionID, command)
}

func validKey(value string) bool {
	length := len(strings.TrimSpace(value))
	return length >= 8 && length <= 200
}
