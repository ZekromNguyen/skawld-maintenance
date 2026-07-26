package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	Pool       *pgxpool.Pool
	Objects    objectstore.Store
	Extractor  Extractor
	Embeddings skawld.EmbeddingProvider
	IDs        id.Generator
	Clock      clock.Clock
	Audit      audit.Sink
}

type source struct {
	OrganizationID string
	SiteID         string
	RevisionID     string
	ObjectKey      string
	MIME           string
	ContentSHA256  string
}

func (p Processor) ProcessDocument(ctx context.Context, revisionID string) error {
	value, ready, err := p.claim(ctx, revisionID)
	if err != nil || ready {
		return err
	}
	body, object, err := p.Objects.Get(ctx, value.ObjectKey)
	if err != nil {
		p.fail(ctx, revisionID, "OBJECT_READ_FAILED")
		return fmt.Errorf("read document object: %w", err)
	}
	defer body.Close()
	if object.Size <= 0 || object.Size > maxDocumentBytes {
		p.fail(ctx, revisionID, "OBJECT_SIZE_INVALID")
		return ErrDocumentTooLarge
	}
	pages, err := p.Extractor.Extract(ctx, body, value.MIME)
	if err != nil {
		p.fail(ctx, revisionID, classifyExtractionError(err))
		return err
	}
	chunks := ChunkPages(pages)
	if len(chunks) == 0 {
		p.fail(ctx, revisionID, "NO_EXTRACTABLE_TEXT")
		return ErrNoExtractableText
	}
	texts := make([]string, len(chunks))
	for index := range chunks {
		texts[index] = chunks[index].Content
	}
	vectors, err := p.Embeddings.Embed(ctx, texts)
	if err != nil {
		p.fail(ctx, revisionID, "EMBEDDING_PROVIDER_FAILED")
		return fmt.Errorf("embed document chunks: %w", err)
	}
	if len(vectors) != len(chunks) {
		p.fail(ctx, revisionID, "EMBEDDING_COUNT_INVALID")
		return errors.New("embedding provider returned an invalid vector count")
	}
	return p.commit(ctx, value, chunks, vectors)
}

func (p Processor) claim(
	ctx context.Context,
	revisionID string,
) (source, bool, error) {
	var value source
	var state string
	err := p.Pool.QueryRow(ctx, `
		UPDATE document_revisions r
		SET ingestion_state = CASE
		      WHEN ingestion_state = 'READY' THEN 'READY'
		      ELSE 'PROCESSING'
		    END,
		    ingestion_error = CASE
		      WHEN ingestion_state = 'READY' THEN ingestion_error
		      ELSE NULL
		    END,
		    updated_at = $2
		FROM documents d, attachments a
		WHERE r.id = $1::uuid
		  AND d.id = r.document_id
		  AND a.id = r.attachment_id
		  AND a.state = 'AVAILABLE'
		  AND r.ingestion_state IN ('QUEUED', 'FAILED', 'PROCESSING', 'READY')
		RETURNING r.organization_id::text, d.site_id::text, r.id::text,
		          a.object_key, a.verified_mime, r.content_sha256,
		          r.ingestion_state
	`, revisionID, p.Clock.Now()).Scan(
		&value.OrganizationID, &value.SiteID, &value.RevisionID,
		&value.ObjectKey, &value.MIME, &value.ContentSHA256, &state,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return source{}, false, errors.New("document revision is not ready for ingestion")
	}
	return value, state == "READY", err
}

func (p Processor) commit(
	ctx context.Context,
	value source,
	chunks []Chunk,
	vectors [][]float32,
) error {
	model := p.Embeddings.Model()
	_, err := database.InTx(ctx, p.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		now := p.Clock.Now()
		var currentHash, state string
		err := tx.QueryRow(ctx, `
			SELECT content_sha256, ingestion_state
			FROM document_revisions
			WHERE id = $1::uuid
			FOR UPDATE
		`, value.RevisionID).Scan(&currentHash, &state)
		if err != nil {
			return struct{}{}, err
		}
		if currentHash != value.ContentSHA256 || state != "PROCESSING" {
			return struct{}{}, errors.New("document revision changed during ingestion")
		}
		_, err = tx.Exec(ctx, `
			UPDATE embeddings
			SET state = 'SUPERSEDED'
			WHERE source_kind = 'DOCUMENT_CHUNK'
			  AND source_id IN (
			    SELECT id FROM document_chunks WHERE revision_id = $1::uuid
			  )
			  AND state = 'ACTIVE'
		`, value.RevisionID)
		if err != nil {
			return struct{}{}, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM document_chunks WHERE revision_id = $1::uuid
		`, value.RevisionID); err != nil {
			return struct{}{}, err
		}
		for index, chunk := range chunks {
			chunkID := p.IDs.New()
			_, err := tx.Exec(ctx, `
				INSERT INTO document_chunks (
					id, organization_id, revision_id, ordinal, locator,
					content, content_sha256, token_estimate, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9
				)
			`, chunkID, value.OrganizationID, value.RevisionID, chunk.Ordinal,
				chunk.Locator, chunk.Content, chunk.ContentSHA256,
				chunk.TokenEstimate, now)
			if err != nil {
				return struct{}{}, err
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO embeddings (
					id, organization_id, source_kind, source_id, source_revision,
					provider, model, model_version, dimensions, distance_metric,
					embedding, input_sha256, state, created_at
				) VALUES (
					$1::uuid, $2::uuid, 'DOCUMENT_CHUNK', $3::uuid, $4,
					$5, $6, $7, $8, $9, $10::vector, $11, 'ACTIVE', $12
				)
			`, p.IDs.New(), value.OrganizationID, chunkID, value.ContentSHA256,
				model.Provider, model.Model, model.ModelVersion, model.Dimensions,
				model.Metric, embeddingLiteral(vectors[index]),
				chunk.ContentSHA256, now)
			if err != nil {
				return struct{}{}, err
			}
		}
		tag, err := tx.Exec(ctx, `
			UPDATE document_revisions
			SET ingestion_state = 'READY', ingestion_error = NULL,
			    version = version + 1, updated_at = $2
			WHERE id = $1::uuid AND ingestion_state = 'PROCESSING'
		`, value.RevisionID, now)
		if err != nil {
			return struct{}{}, err
		}
		if tag.RowsAffected() != 1 {
			return struct{}{}, errors.New("document ingestion state conflict")
		}
		metadata, _ := json.Marshal(map[string]any{
			"provider": model.Provider, "model": model.Model,
			"model_version": model.ModelVersion, "chunks": len(chunks),
		})
		if err := p.Audit.Append(ctx, tx, audit.Event{
			ID: p.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			Action: "knowledge.revision.ingested", EntityKind: "document_revision",
			EntityID: value.RevisionID, AIInvolvement: json.RawMessage(metadata),
			After:      map[string]any{"ingestion_state": "READY", "chunks": len(chunks)},
			OccurredAt: now,
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

func (p Processor) fail(ctx context.Context, revisionID, code string) {
	_, _ = p.Pool.Exec(ctx, `
		UPDATE document_revisions
		SET ingestion_state = 'FAILED', ingestion_error = $2, updated_at = $3
		WHERE id = $1::uuid AND ingestion_state = 'PROCESSING'
	`, revisionID, code, p.Clock.Now())
}

func classifyExtractionError(err error) string {
	switch {
	case errors.Is(err, ErrDocumentTooLarge):
		return "DOCUMENT_LIMIT_EXCEEDED"
	case errors.Is(err, ErrNoExtractableText):
		return "OCR_REQUIRED"
	case errors.Is(err, ErrUnsupportedDocument):
		return "UNSUPPORTED_DOCUMENT"
	default:
		return "EXTRACTION_FAILED"
	}
}

func embeddingLiteral(values []float32) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for index, value := range values {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	builder.WriteByte(']')
	return builder.String()
}
