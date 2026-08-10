package application

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrForbidden = errors.New("attachment operation forbidden")
	ErrNotFound  = errors.New("attachment not found")
	ErrInvalid   = errors.New("invalid attachment command")
	ErrConflict  = errors.New("attachment state conflict")
)

var checksumPattern = regexp.MustCompile(`\A[a-f0-9]{64}\z`)

type CreateManifest struct {
	SiteID           string `json:"site_id"`
	EntityKind       string `json:"entity_kind"`
	EntityID         string `json:"entity_id"`
	ClientEventID    string `json:"client_event_id"`
	OriginalFilename string `json:"original_filename"`
	DeclaredMIME     string `json:"declared_mime"`
	SizeBytes        int64  `json:"size_bytes"`
	ChecksumSHA256   string `json:"checksum_sha256"`
}

type CompleteUpload struct{}

type Attachment struct {
	ID               string            `json:"id"`
	OrganizationID   string            `json:"organization_id"`
	SiteID           string            `json:"site_id"`
	EntityKind       string            `json:"entity_kind"`
	EntityID         string            `json:"entity_id"`
	ClientEventID    string            `json:"client_event_id"`
	OriginalFilename string            `json:"original_filename"`
	DeclaredMIME     string            `json:"declared_mime"`
	VerifiedMIME     string            `json:"verified_mime,omitempty"`
	SizeBytes        int64             `json:"size_bytes"`
	ChecksumSHA256   string            `json:"checksum_sha256"`
	State            string            `json:"state"`
	UploadURL        string            `json:"upload_url,omitempty"`
	UploadHeaders    map[string]string `json:"upload_headers,omitempty"`
	UploadExpiresAt  time.Time         `json:"upload_expires_at,omitempty"`
	DownloadURL      string            `json:"download_url,omitempty"`
}

type Store interface {
	Create(context.Context, identitydomain.Principal, string, CreateManifest) (Attachment, bool, error)
	Complete(context.Context, identitydomain.Principal, string, string, CompleteUpload) (Attachment, bool, error)
	ListByEntity(context.Context, identitydomain.Principal, string, string) ([]Attachment, error)
}

type Service struct {
	Store Store
}

func (s Service) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command CreateManifest,
) (Attachment, bool, error) {
	if !principal.Has(identitydomain.PermissionAttachmentWrite) ||
		!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
		return Attachment{}, false, ErrForbidden
	}
	command.OriginalFilename = filepath.Base(strings.TrimSpace(command.OriginalFilename))
	if !validKey(key) || command.EntityID == "" || command.ClientEventID == "" ||
		command.OriginalFilename == "" || command.OriginalFilename == "." ||
		command.SizeBytes <= 0 || command.SizeBytes > 50<<20 ||
		!checksumPattern.MatchString(command.ChecksumSHA256) {
		return Attachment{}, false, ErrInvalid
	}
	switch command.EntityKind {
	case "EXECUTION", "OBSERVATION", "INCIDENT", "DOCUMENT_REVISION":
	default:
		return Attachment{}, false, ErrInvalid
	}
	if command.EntityKind == "DOCUMENT_REVISION" {
		switch command.DeclaredMIME {
		case "application/pdf", "text/plain":
		default:
			return Attachment{}, false, errors.Join(ErrInvalid, errors.New("unsupported document MIME type"))
		}
	} else {
		switch command.DeclaredMIME {
		case "image/jpeg", "image/png", "image/webp", "audio/m4a",
			"audio/mp4", "audio/mpeg", "audio/wav",
			"video/mp4", "video/webm", "video/quicktime":
		default:
			return Attachment{}, false, errors.Join(ErrInvalid, errors.New("unsupported attachment MIME type"))
		}
	}
	return s.Store.Create(ctx, principal, key, command)
}

func (s Service) ListByEntity(
	ctx context.Context,
	principal identitydomain.Principal,
	entityKind, entityID string,
) ([]Attachment, error) {
	if !principal.Has(identitydomain.PermissionAttachmentWrite) {
		return nil, ErrForbidden
	}
	if entityKind == "" || entityID == "" {
		return nil, ErrInvalid
	}
	switch entityKind {
	case "EXECUTION", "OBSERVATION", "INCIDENT":
	default:
		return nil, ErrInvalid
	}
	return s.Store.ListByEntity(ctx, principal, entityKind, entityID)
}

func (s Service) Complete(
	ctx context.Context,
	principal identitydomain.Principal,
	key, attachmentID string,
) (Attachment, bool, error) {
	if !principal.Has(identitydomain.PermissionAttachmentWrite) {
		return Attachment{}, false, ErrForbidden
	}
	if !validKey(key) {
		return Attachment{}, false, ErrInvalid
	}
	return s.Store.Complete(ctx, principal, key, attachmentID, CompleteUpload{})
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}
