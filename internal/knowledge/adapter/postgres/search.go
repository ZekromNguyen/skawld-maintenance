package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

func (s Store) Search(
	ctx context.Context,
	principal identitydomain.Principal,
	query knowledgedomain.SearchQuery,
) (knowledgedomain.SearchResult, error) {
	if s.Embeddings == nil {
		return knowledgedomain.SearchResult{}, errors.New("embedding provider is unavailable")
	}
	if !principal.CanAccessSite(principal.OrganizationID, query.SiteID) {
		return knowledgedomain.SearchResult{}, knowledgeapp.ErrForbidden
	}
	started := time.Now()
	vectors, err := s.Embeddings.Embed(ctx, []string{query.Query})
	if err != nil {
		return knowledgedomain.SearchResult{}, fmt.Errorf("embed search query: %w", err)
	}
	if len(vectors) != 1 {
		return knowledgedomain.SearchResult{}, errors.New("embedding provider returned an invalid vector count")
	}
	model := s.Embeddings.Model()
	vector := vectorLiteral(vectors[0])
	rows, err := s.Pool.Query(ctx, `
		WITH asset_context AS (
			SELECT id, asset_class, manufacturer, model
			FROM assets
			WHERE id = nullif($3, '')::uuid
			  AND organization_id = $1::uuid AND site_id = $2::uuid
		),
		eligible_documents AS (
			SELECT
				c.id, 'DOCUMENT_CHUNK'::text AS kind, c.id AS source_id,
				r.document_id, c.revision_id, r.revision, d.title,
				c.locator, d.authority, c.content, c.content_sha256,
				c.search_vector, e.embedding
			FROM document_chunks c
			JOIN document_revisions r ON r.id = c.revision_id
			JOIN documents d ON d.id = r.document_id
			JOIN embeddings e
			  ON e.source_kind = 'DOCUMENT_CHUNK' AND e.source_id = c.id
			 AND e.organization_id = c.organization_id AND e.state = 'ACTIVE'
			 AND e.provider = $6 AND e.model = $7 AND e.model_version = $8
			LEFT JOIN asset_context ac ON true
			WHERE c.organization_id = $1::uuid
			  AND (d.site_id IS NULL OR d.site_id = $2::uuid)
			  AND r.approval_status = 'APPROVED'
			  AND r.ingestion_state = 'READY'
			  AND r.superseded_by_id IS NULL
			  AND (r.effective_at IS NULL OR r.effective_at <= $9)
			  AND (r.expires_at IS NULL OR r.expires_at > $9)
			  AND ($5 = '' OR r.language = $5)
			  AND EXISTS (
			    SELECT 1
			    FROM document_applicability a
			    WHERE a.revision_id = r.id
			      AND (a.site_id IS NULL OR a.site_id = $2::uuid)
			      AND (a.asset_id IS NULL OR a.asset_id = nullif($3, '')::uuid)
			      AND (a.asset_class IS NULL OR lower(a.asset_class) = lower(coalesce(ac.asset_class, '')))
			      AND (a.manufacturer IS NULL OR lower(a.manufacturer) = lower(coalesce(ac.manufacturer, '')))
			      AND (a.model IS NULL OR lower(a.model) = lower(coalesce(ac.model, '')))
			      AND (a.process_service IS NULL OR lower(a.process_service) = lower($10))
			  )
		),
		eligible_incidents AS (
			SELECT
				i.id, 'HISTORICAL_INCIDENT'::text AS kind, i.id AS source_id,
				NULL::uuid AS document_id, NULL::uuid AS revision_id,
				i.version::text AS revision,
				concat(i.number, ' · ', a.tag) AS title,
				'incident resolution'::text AS locator,
				'SITE_APPROVED'::text AS authority,
				concat(i.summary, E'\nResolution: ', i.resolution_summary) AS content,
				''::text AS content_sha256,
				to_tsvector(
				  'english', concat(i.summary, ' ', i.resolution_summary, ' ', a.tag)
				) AS search_vector,
				e.embedding
			FROM incidents i
			JOIN assets a ON a.id = i.asset_id
			LEFT JOIN embeddings e
			  ON e.source_kind = 'INCIDENT' AND e.source_id = i.id
			 AND e.organization_id = i.organization_id AND e.state = 'ACTIVE'
			 AND e.provider = $6 AND e.model = $7 AND e.model_version = $8
			LEFT JOIN asset_context ac ON true
			WHERE i.organization_id = $1::uuid AND i.site_id = $2::uuid
			  AND i.state = 'RESOLVED'
			  AND (
			    nullif($3, '') IS NULL OR
			    lower(a.asset_class) = lower(coalesce(ac.asset_class, ''))
			  )
		),
		eligible AS (
			SELECT * FROM eligible_documents
			UNION ALL
			SELECT * FROM eligible_incidents
		),
		lexical AS (
			SELECT id,
			       row_number() OVER (
			         ORDER BY ts_rank_cd(search_vector, websearch_to_tsquery('english', $4)) DESC, id
			       )::integer AS rank,
			       ts_rank_cd(search_vector, websearch_to_tsquery('english', $4))::float8 AS score
			FROM eligible
			WHERE search_vector @@ websearch_to_tsquery('english', $4)
			ORDER BY score DESC, id
			LIMIT 100
		),
		semantic AS (
			SELECT id,
			       row_number() OVER (ORDER BY embedding <=> $11::vector, id)::integer AS rank,
			       (1 - (embedding <=> $11::vector))::float8 AS score
			FROM eligible
			WHERE embedding IS NOT NULL
			ORDER BY embedding <=> $11::vector, id
			LIMIT 100
		),
		fused AS (
			SELECT
				coalesce(l.id, v.id) AS id,
				l.rank AS lexical_rank, v.rank AS vector_rank,
				l.score AS lexical_score, v.score AS vector_score,
				(coalesce(1.0 / (60 + l.rank), 0) +
				 coalesce(1.0 / (60 + v.rank), 0))::float8 AS rrf_score
			FROM lexical l
			FULL OUTER JOIN semantic v ON v.id = l.id
		)
		SELECT e.kind, e.source_id::text, coalesce(e.document_id::text, ''),
		       coalesce(e.revision_id::text, ''),
		       e.revision, e.title, e.locator, e.authority, left(e.content, 4000),
		       e.content_sha256, f.lexical_rank, f.vector_rank,
		       f.lexical_score, f.vector_score, f.rrf_score
		FROM fused f
		JOIN eligible e ON e.id = f.id
		ORDER BY f.rrf_score DESC, e.id
		LIMIT $12
	`, principal.OrganizationID, query.SiteID, query.AssetID, query.Query,
		query.Language, model.Provider, model.Model, model.ModelVersion,
		s.Clock.Now(), query.ProcessService, vector, query.Limit)
	if err != nil {
		return knowledgedomain.SearchResult{}, fmt.Errorf("hybrid evidence query: %w", err)
	}
	defer rows.Close()
	items := make([]knowledgedomain.Evidence, 0, query.Limit)
	for rows.Next() {
		var item knowledgedomain.Evidence
		var lexicalRank, vectorRank *int
		var lexicalScore, vectorScore *float64
		if err := rows.Scan(
			&item.Kind, &item.SourceID, &item.DocumentID, &item.RevisionID, &item.Revision,
			&item.Title, &item.Locator, &item.Authority, &item.Content,
			&item.ContentHash, &lexicalRank, &vectorRank,
			&lexicalScore, &vectorScore, &item.Score.RRFScore,
		); err != nil {
			return knowledgedomain.SearchResult{}, err
		}
		item.ID = strings.ToLower(item.Kind) + ":" + item.SourceID
		if item.ContentHash == "" {
			item.ContentHash = skawld.HashBytes([]byte(item.Content))
		}
		if lexicalRank != nil {
			item.Score.LexicalRank = *lexicalRank
		}
		if vectorRank != nil {
			item.Score.VectorRank = *vectorRank
		}
		if lexicalScore != nil {
			item.Score.LexicalScore = *lexicalScore
		}
		if vectorScore != nil {
			item.Score.VectorScore = *vectorScore
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return knowledgedomain.SearchResult{}, err
	}
	runID := s.IDs.New()
	filters := map[string]any{
		"site_id": query.SiteID, "asset_id": query.AssetID,
		"language": query.Language, "process_service": query.ProcessService,
		"eligibility": []string{
			"organization", "authorized_site", "approved", "current",
			"ingestion_ready", "applicable",
		},
	}
	ranked, _ := json.Marshal(items)
	filterJSON, _ := json.Marshal(filters)
	contextJSON, _ := json.Marshal(map[string]string{
		"site_id": query.SiteID, "asset_id": query.AssetID,
	})
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO retrieval_runs (
			id, organization_id, site_id, actor_id, query_sha256,
			context_sha256, filters, ranked_evidence, embedding_provider,
			embedding_model, embedding_version, latency_ms, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6,
			$7::jsonb, $8::jsonb, $9, $10, $11, $12, $13
		)
	`, runID, principal.OrganizationID, query.SiteID, principal.ID,
		skawld.HashBytes([]byte(query.Query)), skawld.HashBytes(contextJSON),
		filterJSON, ranked, model.Provider, model.Model, model.ModelVersion,
		time.Since(started).Milliseconds(), s.Clock.Now())
	if err != nil {
		return knowledgedomain.SearchResult{}, err
	}
	return knowledgedomain.SearchResult{RetrievalRunID: runID, Items: items}, nil
}

func vectorLiteral(values []float32) string {
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
