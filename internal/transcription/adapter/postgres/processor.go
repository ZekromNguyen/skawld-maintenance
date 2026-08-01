package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Processor struct {
	Pool     *pgxpool.Pool
	Objects  objectstore.Store
	Provider skawld.TranscriptionProvider
	IDs      id.Generator
	Clock    clock.Clock
	Audit    audit.Sink
}

func (p Processor) ProcessTranscription(ctx context.Context, transcriptionID string) error {
	var organizationID, siteID, objectKey, mimeType, sourceHash string
	err := p.Pool.QueryRow(ctx, `
		UPDATE transcriptions t
		SET state = 'PROCESSING', updated_at = $2
		FROM attachments a
		WHERE t.id = $1::uuid AND t.attachment_id = a.id
		  AND t.state IN ('QUEUED', 'FAILED', 'PROCESSING')
		RETURNING t.organization_id::text, t.site_id::text, a.object_key,
		          a.verified_mime, t.source_sha256
	`, transcriptionID, p.Clock.Now()).Scan(
		&organizationID, &siteID, &objectKey, &mimeType, &sourceHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("transcription is not queued")
	}
	if err != nil {
		return err
	}
	body, _, err := p.Objects.Get(ctx, objectKey)
	if err != nil {
		p.fail(ctx, transcriptionID)
		return fmt.Errorf("read transcription audio: %w", err)
	}
	defer body.Close()
	transcript, err := p.Provider.Transcribe(ctx, body, mimeType)
	if err != nil {
		p.fail(ctx, transcriptionID)
		return err
	}
	if transcript.Metadata != p.Provider.Model() {
		p.fail(ctx, transcriptionID)
		return errors.New("transcription provider metadata changed during processing")
	}
	_, err = database.InTx(ctx, p.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		now := p.Clock.Now()
		tag, err := tx.Exec(ctx, `
			UPDATE transcriptions
			SET state = 'CANDIDATE', transcript = $2, language = nullif($3, ''),
			    confidence = $4, version = version + 1, updated_at = $5
			WHERE id = $1::uuid AND state = 'PROCESSING'
			  AND source_sha256 = $6
		`, transcriptionID, transcript.Text, transcript.Language,
			transcript.Confidence, now, sourceHash)
		if err != nil {
			return struct{}{}, err
		}
		if tag.RowsAffected() != 1 {
			return struct{}{}, errors.New("transcription state conflict")
		}
		if err := p.Audit.Append(ctx, tx, audit.Event{
			ID: p.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
			Action:     "transcription.candidate.generated",
			EntityKind: "transcription", EntityID: transcriptionID,
			AIInvolvement: map[string]any{
				"provider":      transcript.Metadata.Provider,
				"model":         transcript.Metadata.Model,
				"model_version": transcript.Metadata.ModelVersion,
				"confidence":    transcript.Confidence,
			},
			After: map[string]any{
				"state": "CANDIDATE", "verification_status": "UNVERIFIED",
			},
			OccurredAt: now,
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	if err != nil {
		p.fail(ctx, transcriptionID)
	}
	return err
}

func (p Processor) fail(ctx context.Context, transcriptionID string) {
	_, _ = p.Pool.Exec(ctx, `
		UPDATE transcriptions
		SET state = 'FAILED', updated_at = $2
		WHERE id = $1::uuid AND state = 'PROCESSING'
	`, transcriptionID, p.Clock.Now())
}
