package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

func (s Store) ApproveRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command knowledgeapp.ApproveRevision,
) (knowledgedomain.Revision, bool, error) {
	return s.changeRevisionState(
		ctx, principal, key, revisionID, "approve", command,
		func(ctx context.Context, tx pgx.Tx, value *knowledgedomain.Revision, siteID string) error {
			if value.ApprovalStatus != knowledgedomain.ApprovalDraft &&
				value.ApprovalStatus != knowledgedomain.ApprovalReviewRequired {
				return knowledgeapp.ErrConflict
			}
			now := s.Clock.Now()
			_, err := tx.Exec(ctx, `
				UPDATE document_revisions
				SET approval_status = 'SUPERSEDED', superseded_by_id = $1::uuid,
				    version = version + 1, updated_at = $2
				WHERE document_id = $3::uuid AND approval_status = 'APPROVED'
				  AND id <> $1::uuid
			`, value.ID, now, value.DocumentID)
			if err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `
				UPDATE document_revisions
				SET approval_status = 'APPROVED', approved_by = $2::uuid,
				    approved_at = $3, version = version + 1, updated_at = $3
				WHERE id = $1::uuid AND version = $4
			`, value.ID, principal.ID, now, command.ExpectedVersion)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return knowledgeapp.ErrConflict
			}
			value.ApprovalStatus = knowledgedomain.ApprovalApproved
			value.ApprovedBy = principal.ID
			value.ApprovedAt = &now
			value.Version++
			return nil
		},
	)
}

func (s Store) RetireRevision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command knowledgeapp.RetireRevision,
) (knowledgedomain.Revision, bool, error) {
	return s.changeRevisionState(
		ctx, principal, key, revisionID, "retire", command,
		func(ctx context.Context, tx pgx.Tx, value *knowledgedomain.Revision, siteID string) error {
			if value.ApprovalStatus == knowledgedomain.ApprovalRetired ||
				value.ApprovalStatus == knowledgedomain.ApprovalSuperseded {
				return knowledgeapp.ErrConflict
			}
			now := s.Clock.Now()
			tag, err := tx.Exec(ctx, `
				UPDATE document_revisions
				SET approval_status = 'RETIRED', version = version + 1, updated_at = $2
				WHERE id = $1::uuid AND version = $3
			`, value.ID, now, command.ExpectedVersion)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return knowledgeapp.ErrConflict
			}
			value.ApprovalStatus = knowledgedomain.ApprovalRetired
			value.Version++
			return nil
		},
	)
}

func (s Store) changeRevisionState(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID, action string,
	command any,
	change func(context.Context, pgx.Tx, *knowledgedomain.Revision, string) error,
) (knowledgedomain.Revision, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	scope := "knowledge.revision." + action + ".v1:" + revisionID
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (revisionResult, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return revisionResult{}, err
		}
		if record.Replay {
			var replay knowledgedomain.Revision
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return revisionResult{}, err
			}
			return revisionResult{Value: replay, Replay: true}, nil
		}
		value, siteID, err := loadRevisionForUpdate(ctx, tx, principal, revisionID)
		if err != nil {
			return revisionResult{}, err
		}
		before := value
		if err := change(ctx, tx, &value, siteID); err != nil {
			return revisionResult{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: principal.OrganizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "knowledge.revision." + action + "d",
			EntityKind: "document_revision", EntityID: value.ID,
			Before: before, After: value, OccurredAt: now,
		}); err != nil {
			return revisionResult{}, err
		}
		if err := encodeReplay(
			ctx, s.Idempotency, tx, principal.ID, scope, key,
			http.StatusOK, value, now,
		); err != nil {
			return revisionResult{}, err
		}
		return revisionResult{Value: value}, nil
	})
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	return result.Value, result.Replay, nil
}

func (s Store) RequestIngestion(
	ctx context.Context,
	principal identitydomain.Principal,
	key, revisionID string,
	command knowledgeapp.RequestIngestion,
) (knowledgedomain.Revision, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	scope := "knowledge.revision.ingest.v1:" + revisionID
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (revisionResult, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return revisionResult{}, err
		}
		if record.Replay {
			var replay knowledgedomain.Revision
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return revisionResult{}, err
			}
			return revisionResult{Value: replay, Replay: true}, nil
		}
		value, siteID, err := loadRevisionForUpdate(ctx, tx, principal, revisionID)
		if err != nil {
			return revisionResult{}, err
		}
		var checksum string
		err = tx.QueryRow(ctx, `
			SELECT checksum_sha256
			FROM attachments
			WHERE id = $1::uuid AND entity_kind = 'DOCUMENT_REVISION'
			  AND entity_id = $2::uuid AND organization_id = $3::uuid
			  AND site_id = $4::uuid AND state = 'AVAILABLE'
			  AND verified_mime IN ('application/pdf', 'text/plain')
		`, command.AttachmentID, revisionID, principal.OrganizationID, siteID).Scan(&checksum)
		if errors.Is(err, pgx.ErrNoRows) {
			return revisionResult{}, knowledgeapp.ErrNoAttachment
		}
		if err != nil {
			return revisionResult{}, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE document_revisions
			SET attachment_id = $2::uuid, ingestion_state = 'QUEUED',
			    ingestion_error = NULL, content_sha256 = $3,
			    version = version + 1, updated_at = $4
			WHERE id = $1::uuid
		`, revisionID, command.AttachmentID, checksum, now)
		if err != nil {
			return revisionResult{}, err
		}
		if s.Enqueuer == nil {
			return revisionResult{}, errors.New("document ingestion enqueuer is unavailable")
		}
		if err := s.Enqueuer.EnqueueDocumentIngestionTx(
			ctx, tx, revisionID, command.AttachmentID,
		); err != nil {
			return revisionResult{}, err
		}
		value.AttachmentID = command.AttachmentID
		value.ContentSHA256 = checksum
		value.IngestionState = knowledgedomain.IngestionQueued
		value.IngestionError = ""
		value.Version++
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: principal.OrganizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "knowledge.revision.ingestion_requested",
			EntityKind: "document_revision", EntityID: value.ID,
			After: value, OccurredAt: now,
		}); err != nil {
			return revisionResult{}, err
		}
		if err := encodeReplay(
			ctx, s.Idempotency, tx, principal.ID, scope, key,
			http.StatusAccepted, value, now,
		); err != nil {
			return revisionResult{}, err
		}
		return revisionResult{Value: value}, nil
	})
	if err != nil {
		return knowledgedomain.Revision{}, false, err
	}
	return result.Value, result.Replay, nil
}

func loadRevisionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	revisionID string,
) (knowledgedomain.Revision, string, error) {
	var value knowledgedomain.Revision
	var siteID string
	err := tx.QueryRow(ctx, `
		SELECT r.id::text, r.document_id::text, r.revision, r.approval_status,
		       r.effective_at, r.expires_at, coalesce(r.superseded_by_id::text, ''),
		       coalesce(r.attachment_id::text, ''), r.ingestion_state,
		       coalesce(r.ingestion_error, ''), coalesce(r.content_sha256, ''),
		       r.language, coalesce(r.approved_by::text, ''), r.approved_at,
		       r.version, d.site_id::text
		FROM document_revisions r
		JOIN documents d ON d.id = r.document_id
		WHERE r.id = $1::uuid AND r.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR d.site_id = ANY($3::uuid[]))
		FOR UPDATE OF r
	`, revisionID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.DocumentID, &value.Revision, &value.ApprovalStatus,
		&value.EffectiveAt, &value.ExpiresAt, &value.SupersededByID,
		&value.AttachmentID, &value.IngestionState, &value.IngestionError,
		&value.ContentSHA256, &value.Language, &value.ApprovedBy,
		&value.ApprovedAt, &value.Version, &siteID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return knowledgedomain.Revision{}, "", knowledgeapp.ErrNotFound
	}
	return value, siteID, err
}
