package ingest

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type memoryObject struct {
	content string
}

func (s memoryObject) Get(context.Context, string) (io.ReadCloser, objectstore.Object, error) {
	return io.NopCloser(strings.NewReader(s.content)), objectstore.Object{
		Key: "document", Size: int64(len(s.content)), ContentType: "text/plain",
	}, nil
}
func (memoryObject) Put(context.Context, objectstore.PutRequest) (objectstore.Object, error) {
	panic("not used")
}
func (memoryObject) Head(context.Context, string) (objectstore.Object, error) {
	panic("not used")
}
func (memoryObject) Delete(context.Context, string) error {
	panic("not used")
}
func (memoryObject) SignedUploadURL(context.Context, string, objectstore.UploadConstraints, time.Duration) (objectstore.SignedURL, error) {
	panic("not used")
}
func (memoryObject) SignedDownloadURL(context.Context, string, time.Duration) (string, error) {
	panic("not used")
}

func TestProcessorCommitsChunksEmbeddingsAndReadyState(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	now := time.Now().UTC().Truncate(time.Second)
	organizationID, siteID, principalID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	documentID, revisionID, attachmentID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	content := "High vibration procedure.\n\nInspect lubrication condition and bearing temperature."
	checksum := skawld.HashBytes([]byte(content))
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Ingestion Test', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, 'IG', 'Ingestion Site', 'UTC', 'ACTIVE', 1, $3, $3)
	`, siteID, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Ingestion User', 'ACTIVE', $2, $2)
	`, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO documents (
			id, organization_id, site_id, document_type, title, authority,
			created_by, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, 'SOP', 'Pump SOP',
			'SITE_APPROVED', $4::uuid, $5, $5)
	`, documentID, organizationID, siteID, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO document_revisions (
			id, organization_id, document_id, revision, approval_status,
			ingestion_state, content_sha256, language, created_by, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, 'R1', 'DRAFT',
			'QUEUED', $4, 'en', $5::uuid, $6, $6)
	`, revisionID, organizationID, documentID, checksum, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO attachments (
			id, organization_id, site_id, entity_kind, entity_id, object_key,
			original_filename, declared_mime, verified_mime, size_bytes,
			checksum_sha256, state, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'DOCUMENT_REVISION', $4::uuid,
			$5, 'pump.txt', 'text/plain', 'text/plain', $6, $7,
			'AVAILABLE', $8::uuid, $9, $9
		)
	`, attachmentID, organizationID, siteID, revisionID,
		"test/"+attachmentID, len(content), checksum, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		UPDATE document_revisions SET attachment_id = $2::uuid WHERE id = $1::uuid
	`, revisionID, attachmentID)
	if err != nil {
		t.Fatal(err)
	}
	processor := Processor{
		Pool: pool, Objects: memoryObject{content: content},
		Extractor:  BoundedExtractor{},
		Embeddings: skawld.DeterministicEmbeddingProvider{Dimensions: 64},
		IDs:        id.UUID{}, Clock: clock.Fixed{Time: now}, Audit: audit.Sink{},
	}
	if err := processor.ProcessDocument(ctx, revisionID); err != nil {
		t.Fatal(err)
	}
	var state string
	var chunkCount, embeddingCount int
	if err := pool.QueryRow(ctx, `
		SELECT ingestion_state,
		       (SELECT count(*) FROM document_chunks WHERE revision_id = r.id),
		       (SELECT count(*) FROM embeddings e
		        JOIN document_chunks c ON c.id = e.source_id
		        WHERE c.revision_id = r.id AND e.state = 'ACTIVE')
		FROM document_revisions r WHERE r.id = $1::uuid
	`, revisionID).Scan(&state, &chunkCount, &embeddingCount); err != nil {
		t.Fatal(err)
	}
	if state != "READY" || chunkCount == 0 || embeddingCount != chunkCount {
		t.Fatalf("state=%s chunks=%d embeddings=%d", state, chunkCount, embeddingCount)
	}
}
