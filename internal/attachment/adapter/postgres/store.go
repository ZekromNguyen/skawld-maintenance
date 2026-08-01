package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uploadTTL = 15 * time.Minute

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Idempotency idempotency.Store
	Audit       audit.Sink
	Objects     objectstore.Store
}

type result struct {
	Value  attachmentapp.Attachment
	Replay bool
}

func (s Store) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command attachmentapp.CreateManifest,
) (attachmentapp.Attachment, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return attachmentapp.Attachment{}, false, err
	}
	value, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(
			ctx, tx, principal.ID, "attachment.create.v1", key, hash, now,
		)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay attachmentapp.Attachment
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		if err := validateEntityScope(ctx, tx, principal, command); err != nil {
			return result{}, err
		}
		attachmentID := s.IDs.New()
		objectKey := fmt.Sprintf(
			"organizations/%s/sites/%s/attachments/%s",
			principal.OrganizationID, command.SiteID, attachmentID,
		)
		signed, err := s.Objects.SignedUploadURL(
			ctx, objectKey,
			objectstore.UploadConstraints{
				ContentType: command.DeclaredMIME,
				Size:        command.SizeBytes,
				Metadata: map[string]string{
					"checksum-sha256": command.ChecksumSHA256,
					"attachment-id":   attachmentID,
				},
			},
			uploadTTL,
		)
		if err != nil {
			return result{}, fmt.Errorf("sign attachment upload: %w", err)
		}
		response := attachmentapp.Attachment{
			ID: attachmentID, OrganizationID: principal.OrganizationID,
			SiteID: command.SiteID, EntityKind: command.EntityKind,
			EntityID: command.EntityID, ClientEventID: command.ClientEventID,
			OriginalFilename: command.OriginalFilename, DeclaredMIME: command.DeclaredMIME,
			SizeBytes: command.SizeBytes, ChecksumSHA256: command.ChecksumSHA256,
			State: "PENDING_UPLOAD", UploadURL: signed.URL,
			UploadHeaders: signed.Headers, UploadExpiresAt: signed.ExpiresAt,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO attachments (
				id, organization_id, site_id, entity_kind, entity_id, object_key,
				original_filename, declared_mime, size_bytes, checksum_sha256,
				state, client_event_id, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5::uuid, $6,
				$7, $8, $9, $10, $11, $12::uuid, $13::uuid, $14, $14
			)
		`, response.ID, response.OrganizationID, response.SiteID, response.EntityKind,
			response.EntityID, objectKey, response.OriginalFilename, response.DeclaredMIME,
			response.SizeBytes, response.ChecksumSHA256, response.State,
			response.ClientEventID, principal.ID, now)
		if err != nil {
			return result{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: response.OrganizationID, SiteID: response.SiteID,
			ActorID: principal.ID, Action: "attachment.upload.requested",
			EntityKind: "attachment", EntityID: response.ID, After: response,
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(response)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, "attachment.create.v1", key,
			http.StatusCreated, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: response}, nil
	})
	if err != nil {
		return attachmentapp.Attachment{}, false, err
	}
	return value.Value, value.Replay, nil
}

func (s Store) Complete(
	ctx context.Context,
	principal identitydomain.Principal,
	key, attachmentID string,
	command attachmentapp.CompleteUpload,
) (attachmentapp.Attachment, bool, error) {
	pending, objectKey, err := s.load(ctx, principal, attachmentID)
	if err != nil {
		return attachmentapp.Attachment{}, false, err
	}
	if pending.State == "AVAILABLE" {
		return pending, true, nil
	}
	if pending.State != "PENDING_UPLOAD" {
		return attachmentapp.Attachment{}, false, attachmentapp.ErrConflict
	}
	object, err := s.Objects.Head(ctx, objectKey)
	if err != nil {
		return attachmentapp.Attachment{}, false, fmt.Errorf("inspect uploaded attachment metadata: %w", err)
	}
	if object.Size != pending.SizeBytes || object.Size <= 0 || object.Size > 50<<20 {
		_ = s.reject(ctx, principal, pending, "uploaded object failed size verification")
		return attachmentapp.Attachment{}, false, errors.Join(
			attachmentapp.ErrInvalid,
			errors.New("uploaded object does not match declared manifest"),
		)
	}
	verifiedMIME, size, checksum, err := inspectObject(
		ctx, s.Objects, objectKey, pending.SizeBytes,
	)
	if err != nil {
		return attachmentapp.Attachment{}, false, fmt.Errorf("inspect uploaded attachment: %w", err)
	}
	if size != pending.SizeBytes || checksum != pending.ChecksumSHA256 ||
		!mimeMatches(pending.DeclaredMIME, verifiedMIME) {
		_ = s.reject(ctx, principal, pending, "uploaded object failed size, checksum, or MIME verification")
		return attachmentapp.Attachment{}, false, errors.Join(
			attachmentapp.ErrInvalid,
			errors.New("uploaded object does not match declared manifest"),
		)
	}
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return attachmentapp.Attachment{}, false, err
	}
	scope := "attachment.complete.v1:" + attachmentID
	value, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay attachmentapp.Attachment
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		tag, err := tx.Exec(ctx, `
			UPDATE attachments
			SET state = 'AVAILABLE', verified_mime = $4, updated_at = $5
			WHERE id = $1::uuid AND organization_id = $2::uuid
			  AND site_id = $3::uuid AND state = 'PENDING_UPLOAD'
		`, attachmentID, principal.OrganizationID, pending.SiteID, verifiedMIME, now)
		if err != nil {
			return result{}, err
		}
		if tag.RowsAffected() != 1 {
			return result{}, attachmentapp.ErrConflict
		}
		pending.State = "AVAILABLE"
		pending.VerifiedMIME = verifiedMIME
		pending.UploadURL = ""
		pending.UploadHeaders = nil
		pending.UploadExpiresAt = time.Time{}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: pending.OrganizationID, SiteID: pending.SiteID,
			ActorID: principal.ID, Action: "attachment.available",
			EntityKind: "attachment", EntityID: pending.ID, After: pending,
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(pending)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: pending}, nil
	})
	if err != nil {
		return attachmentapp.Attachment{}, false, err
	}
	return value.Value, value.Replay, nil
}

func (s Store) load(
	ctx context.Context,
	principal identitydomain.Principal,
	attachmentID string,
) (attachmentapp.Attachment, string, error) {
	var value attachmentapp.Attachment
	var objectKey string
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, site_id::text, entity_kind,
		       entity_id::text, coalesce(client_event_id::text, ''),
		       original_filename, declared_mime, coalesce(verified_mime, ''),
		       size_bytes, checksum_sha256, state, object_key
		FROM attachments
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (cardinality($3::uuid[]) = 0 OR site_id = ANY($3::uuid[]))
	`, attachmentID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.EntityKind,
		&value.EntityID, &value.ClientEventID, &value.OriginalFilename,
		&value.DeclaredMIME, &value.VerifiedMIME, &value.SizeBytes,
		&value.ChecksumSHA256, &value.State, &objectKey,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return attachmentapp.Attachment{}, "", attachmentapp.ErrNotFound
	}
	return value, objectKey, err
}

func validateEntityScope(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	command attachmentapp.CreateManifest,
) error {
	var organizationID, siteID string
	var err error
	switch command.EntityKind {
	case "EXECUTION":
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text
			FROM maintenance_executions WHERE id = $1::uuid
		`, command.EntityID).Scan(&organizationID, &siteID)
	case "OBSERVATION":
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text
			FROM observations WHERE id = $1::uuid
		`, command.EntityID).Scan(&organizationID, &siteID)
	case "INCIDENT":
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text
			FROM incidents WHERE id = $1::uuid
		`, command.EntityID).Scan(&organizationID, &siteID)
	case "DOCUMENT_REVISION":
		err = tx.QueryRow(ctx, `
			SELECT r.organization_id::text, d.site_id::text
			FROM document_revisions r
			JOIN documents d ON d.id = r.document_id
			WHERE r.id = $1::uuid
		`, command.EntityID).Scan(&organizationID, &siteID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return attachmentapp.ErrNotFound
	}
	if err != nil {
		return err
	}
	if organizationID != principal.OrganizationID || siteID != command.SiteID ||
		!principal.CanAccessSite(organizationID, siteID) {
		return attachmentapp.ErrForbidden
	}
	return nil
}

func inspectObject(
	ctx context.Context,
	store objectstore.Store,
	key string,
	maxBytes int64,
) (string, int64, string, error) {
	body, _, err := store.Get(ctx, key)
	if err != nil {
		return "", 0, "", err
	}
	defer body.Close()
	limited := io.LimitReader(body, maxBytes+1)
	hash := sha256.New()
	prefix := make([]byte, 512)
	read, readErr := io.ReadFull(limited, prefix)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return "", 0, "", readErr
	}
	prefix = prefix[:read]
	if _, err := hash.Write(prefix); err != nil {
		return "", 0, "", err
	}
	remainder, err := io.Copy(hash, limited)
	if err != nil {
		return "", 0, "", err
	}
	return detectMIME(prefix), int64(read) + remainder, hex.EncodeToString(hash.Sum(nil)), nil
}

func detectMIME(prefix []byte) string {
	detected := http.DetectContentType(prefix)
	if len(prefix) >= 12 && string(prefix[:4]) == "RIFF" && string(prefix[8:12]) == "WAVE" {
		return "audio/wav"
	}
	if len(prefix) >= 12 && string(prefix[4:8]) == "ftyp" {
		return "audio/mp4"
	}
	return detected
}

func mimeMatches(declared, verified string) bool {
	if declared == verified {
		return true
	}
	return declared == "audio/m4a" && verified == "audio/mp4"
}

func (s Store) reject(
	ctx context.Context,
	principal identitydomain.Principal,
	value attachmentapp.Attachment,
	reason string,
) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE attachments
		SET state = 'REJECTED', rejection_reason = $4, updated_at = $5
		WHERE id = $1::uuid AND organization_id = $2::uuid AND site_id = $3::uuid
	`, value.ID, principal.OrganizationID, value.SiteID, reason, s.Clock.Now())
	return err
}
