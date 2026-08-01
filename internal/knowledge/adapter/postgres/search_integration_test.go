package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHybridSearchFiltersScopeValidityAndApplicabilityBeforeRanking(t *testing.T) {
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
	organizationID, siteA, siteB := uuid.NewString(), uuid.NewString(), uuid.NewString()
	principalID, assetA, assetB := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Knowledge Test', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES
		  ($2::uuid, $1::uuid, 'KA', 'Knowledge A', 'UTC', 'ACTIVE', 1, $4, $4),
		  ($3::uuid, $1::uuid, 'KB', 'Knowledge B', 'UTC', 'ACTIVE', 1, $4, $4)
	`, organizationID, siteA, siteB, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, status, created_at, updated_at
		) VALUES ($1::uuid, $1, 'Knowledge Tester', 'ACTIVE', $2, $2)
	`, principalID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO assets (
			id, organization_id, site_id, tag, name, asset_class, status,
			source_of_truth, sync_status, attributes, version, created_at, updated_at
		) VALUES
		  ($4::uuid, $1::uuid, $2::uuid, 'P-A', 'Pump A', 'CENTRIFUGAL_PUMP',
		   'ACTIVE', 'OWNED_BY_SKAWLD', 'NOT_APPLICABLE', '{}'::jsonb, 1, $6, $6),
		  ($5::uuid, $1::uuid, $3::uuid, 'P-B', 'Pump B', 'CENTRIFUGAL_PUMP',
		   'ACTIVE', 'OWNED_BY_SKAWLD', 'NOT_APPLICABLE', '{}'::jsonb, 1, $6, $6)
	`, organizationID, siteA, siteB, assetA, assetB, now)
	if err != nil {
		t.Fatal(err)
	}

	provider := skawld.DeterministicEmbeddingProvider{Dimensions: 64}
	insertDocument := func(
		title, siteID, status, assetClass string,
		expiresAt *time.Time,
	) string {
		t.Helper()
		documentID, revisionID, chunkID := uuid.NewString(), uuid.NewString(), uuid.NewString()
		content := "High vibration troubleshooting requires lubrication inspection."
		vectors, err := provider.Embed(ctx, []string{content})
		if err != nil {
			t.Fatal(err)
		}
		model := provider.Model()
		effectiveAt := now
		if expiresAt != nil {
			effectiveAt = expiresAt.Add(-time.Hour)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO documents (
				id, organization_id, site_id, document_type, title, authority,
				created_by, created_at, updated_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, 'SOP', $4, 'SITE_APPROVED',
				$5::uuid, $6, $6)
		`, documentID, organizationID, siteID, title, principalID, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO document_revisions (
				id, organization_id, document_id, revision, approval_status,
				effective_at, expires_at, ingestion_state, content_sha256,
				approved_by, approved_at, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, 'R1', $4::text, $5::timestamptz,
				$6::timestamptz, 'READY',
				$7, CASE WHEN $4 = 'APPROVED' THEN $8::uuid ELSE NULL END,
				CASE WHEN $4 = 'APPROVED' THEN $5::timestamptz ELSE NULL END,
				$8::uuid, $5::timestamptz, $5::timestamptz
			)
		`, revisionID, organizationID, documentID, status, effectiveAt, expiresAt,
			skawld.HashBytes([]byte(content)), principalID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO document_applicability (
				id, organization_id, revision_id, site_id, asset_class, created_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6)
		`, uuid.NewString(), organizationID, revisionID, siteID, assetClass, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO document_chunks (
				id, organization_id, revision_id, ordinal, locator,
				content, content_sha256, token_estimate, created_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, 1, 'page:1/chunk:1',
				$4, $5, 10, $6)
		`, chunkID, organizationID, revisionID, content,
			skawld.HashBytes([]byte(content)), now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO embeddings (
				id, organization_id, source_kind, source_id, source_revision,
				provider, model, model_version, dimensions, distance_metric,
				embedding, input_sha256, state, created_at
			) VALUES ($1::uuid, $2::uuid, 'DOCUMENT_CHUNK', $3::uuid, $4,
				$5, $6, $7, $8, 'COSINE', $9::vector, $4, 'ACTIVE', $10)
		`, uuid.NewString(), organizationID, chunkID,
			skawld.HashBytes([]byte(content)), model.Provider, model.Model,
			model.ModelVersion, model.Dimensions, vectorLiteral(vectors[0]), now)
		if err != nil {
			t.Fatal(err)
		}
		return chunkID
	}

	eligibleChunk := insertDocument(
		"Current pump SOP", siteA, "APPROVED", "CENTRIFUGAL_PUMP", nil,
	)
	insertDocument(
		"Cross-site SOP", siteB, "APPROVED", "CENTRIFUGAL_PUMP", nil,
	)
	expired := now.Add(-time.Minute)
	insertDocument(
		"Expired SOP", siteA, "APPROVED", "CENTRIFUGAL_PUMP", &expired,
	)
	insertDocument(
		"Wrong class SOP", siteA, "APPROVED", "ELECTRIC_MOTOR", nil,
	)
	insertDocument(
		"Draft SOP", siteA, "DRAFT", "CENTRIFUGAL_PUMP", nil,
	)

	_, err = pool.Exec(ctx, `
		INSERT INTO incidents (
			id, organization_id, site_id, asset_id, number, summary, severity,
			state, source_of_truth, detected_at, resolved_at, resolution_summary,
			version, created_by, created_at, updated_at
		) VALUES
		  ($1::uuid, $3::uuid, $4::uuid, $5::uuid, 'INC-ELIGIBLE',
		   'High vibration', 'HIGH', 'RESOLVED', 'OWNED_BY_SKAWLD',
		   $7, $7, 'Corrected lubrication condition', 2, $6::uuid, $7, $7),
		  ($2::uuid, $3::uuid, $8::uuid, $9::uuid, 'INC-CROSS-SITE',
		   'High vibration', 'HIGH', 'RESOLVED', 'OWNED_BY_SKAWLD',
		   $7, $7, 'Corrected lubrication condition', 2, $6::uuid, $7, $7)
	`, uuid.NewString(), uuid.NewString(), organizationID, siteA, assetA,
		principalID, now, siteB, assetB)
	if err != nil {
		t.Fatal(err)
	}

	store := Store{
		Pool: pool, IDs: id.UUID{}, Clock: clock.Fixed{Time: now},
		Embeddings: provider,
	}
	result, err := store.Search(
		ctx,
		identitydomain.Principal{
			ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteA},
		},
		knowledgedomain.SearchQuery{
			SiteID: siteA, AssetID: assetA,
			Query: "high vibration lubrication", Limit: 10,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var foundDocument, foundHistory bool
	for _, item := range result.Items {
		switch {
		case item.SourceID == eligibleChunk:
			foundDocument = true
		case item.Kind == "HISTORICAL_INCIDENT":
			foundHistory = true
			if item.Title == "INC-CROSS-SITE · P-B" {
				t.Fatal("cross-site incident escaped eligible set")
			}
		case item.Title == "Cross-site SOP", item.Title == "Expired SOP",
			item.Title == "Wrong class SOP", item.Title == "Draft SOP":
			t.Fatalf("ineligible evidence returned: %s", item.Title)
		}
	}
	if !foundDocument || !foundHistory {
		t.Fatalf("found document=%v history=%v; items=%+v", foundDocument, foundHistory, result.Items)
	}
}
