package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	transcriptionapp "github.com/ZekromNguyen/skawld-maintenance/internal/transcription/application"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Enqueuer interface {
	EnqueueTranscriptionTx(context.Context, pgx.Tx, string) error
}

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Audit       audit.Sink
	Idempotency idempotency.Store
	Enqueuer    Enqueuer
}

type result struct {
	Value  transcriptionapp.Transcription
	Replay bool
}

func (s Store) Request(
	ctx context.Context,
	principal identitydomain.Principal,
	key, attachmentID string,
	model skawld.ProviderMetadata,
) (transcriptionapp.Transcription, bool, error) {
	hash, err := idempotency.HashRequest(map[string]string{
		"attachment_id": attachmentID, "provider": model.Provider,
		"model": model.Model, "model_version": model.ModelVersion,
	})
	if err != nil {
		return transcriptionapp.Transcription{}, false, err
	}
	scope := "transcription.request.v1:" + attachmentID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay transcriptionapp.Transcription
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		var organizationID, siteID, checksum, mimeType string
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text, checksum_sha256, verified_mime
			FROM attachments
			WHERE id = $1::uuid AND organization_id = $2::uuid
			  AND (cardinality($3::uuid[]) = 0 OR site_id = ANY($3::uuid[]))
			  AND state = 'AVAILABLE'
			  AND verified_mime IN ('audio/m4a', 'audio/mp4', 'audio/mpeg', 'audio/wav')
		`, attachmentID, principal.OrganizationID, principal.SiteIDs).Scan(
			&organizationID, &siteID, &checksum, &mimeType,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return result{}, transcriptionapp.ErrNotFound
		}
		if err != nil {
			return result{}, err
		}
		value := transcriptionapp.Transcription{
			ID: s.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
			AttachmentID: attachmentID, State: "QUEUED",
			Provider: model.Provider, Model: model.Model, ModelVersion: model.ModelVersion,
			VerificationStatus: "UNVERIFIED", SourceSHA256: checksum,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO transcriptions (
				id, organization_id, site_id, attachment_id, state,
				provider, model, model_version, verification_status,
				source_sha256, created_by, created_at, updated_at, version
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, 'QUEUED',
				$5, $6, $7, 'UNVERIFIED', $8, $9::uuid, $10, $10, 1
			)
		`, value.ID, value.OrganizationID, value.SiteID, value.AttachmentID,
			value.Provider, value.Model, value.ModelVersion, value.SourceSHA256,
			principal.ID, now)
		if err != nil {
			return result{}, err
		}
		if s.Enqueuer == nil {
			return result{}, errors.New("transcription enqueuer is unavailable")
		}
		if err := s.Enqueuer.EnqueueTranscriptionTx(ctx, tx, value.ID); err != nil {
			return result{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "transcription.requested",
			EntityKind: "transcription", EntityID: value.ID,
			After: map[string]any{
				"attachment_id": attachmentID, "provider": model.Provider,
				"model": model.Model, "model_version": model.ModelVersion,
				"mime": mimeType,
			},
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusAccepted, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: value}, nil
	})
	if err != nil {
		return transcriptionapp.Transcription{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	transcriptionID string,
) (transcriptionapp.Transcription, error) {
	return load(ctx, s.Pool, principal, transcriptionID, false)
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func load(
	ctx context.Context,
	query rowQuerier,
	principal identitydomain.Principal,
	transcriptionID string,
	forUpdate bool,
) (transcriptionapp.Transcription, error) {
	statement := `
		SELECT id::text, organization_id::text, site_id::text, attachment_id::text,
		       state, coalesce(transcript, ''), coalesce(language, ''),
		       provider, model, model_version, coalesce(confidence, 0),
		       verification_status, coalesce(verified_by::text, ''), verified_at,
		       source_sha256, version, created_at, updated_at
		FROM transcriptions
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (cardinality($3::uuid[]) = 0 OR site_id = ANY($3::uuid[]))`
	if forUpdate {
		statement += " FOR UPDATE"
	}
	var value transcriptionapp.Transcription
	err := query.QueryRow(
		ctx, statement, transcriptionID, principal.OrganizationID, principal.SiteIDs,
	).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.AttachmentID,
		&value.State, &value.Transcript, &value.Language, &value.Provider,
		&value.Model, &value.ModelVersion, &value.Confidence,
		&value.VerificationStatus, &value.VerifiedBy, &value.VerifiedAt,
		&value.SourceSHA256, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return transcriptionapp.Transcription{}, transcriptionapp.ErrNotFound
	}
	return value, err
}

func (s Store) Verify(
	ctx context.Context,
	principal identitydomain.Principal,
	key, transcriptionID string,
	command transcriptionapp.Verify,
) (transcriptionapp.Transcription, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return transcriptionapp.Transcription{}, false, err
	}
	scope := "transcription.verify.v1:" + transcriptionID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay transcriptionapp.Transcription
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		value, err := load(ctx, tx, principal, transcriptionID, true)
		if err != nil {
			return result{}, err
		}
		if value.State != "CANDIDATE" || value.Version != command.ExpectedVersion {
			return result{}, transcriptionapp.ErrConflict
		}
		state, verification := "REJECTED", "REJECTED"
		if command.Accept {
			state, verification = "VERIFIED", "VERIFIED"
		}
		tag, err := tx.Exec(ctx, `
			UPDATE transcriptions
			SET state = $2, verification_status = $3,
			    transcript = coalesce(nullif($4, ''), transcript),
			    verified_by = $5::uuid, verified_at = $6,
			    version = version + 1, updated_at = $6
			WHERE id = $1::uuid AND state = 'CANDIDATE' AND version = $7
		`, value.ID, state, verification, command.Transcript,
			principal.ID, now, command.ExpectedVersion)
		if err != nil {
			return result{}, err
		}
		if tag.RowsAffected() != 1 {
			return result{}, transcriptionapp.ErrConflict
		}
		value.State, value.VerificationStatus = state, verification
		if command.Transcript != "" {
			value.Transcript = command.Transcript
		}
		value.VerifiedBy, value.VerifiedAt = principal.ID, &now
		value.Version++
		value.UpdatedAt = now
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: "transcription." + map[bool]string{true: "verified", false: "rejected"}[command.Accept],
			EntityKind: "transcription", EntityID: value.ID,
			After: map[string]any{
				"state": state, "verification_status": verification,
				"human_edited": command.Transcript != "",
			},
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: value}, nil
	})
	if err != nil {
		return transcriptionapp.Transcription{}, false, err
	}
	return stored.Value, stored.Replay, nil
}
