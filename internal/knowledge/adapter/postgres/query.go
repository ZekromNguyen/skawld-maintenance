package postgres

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/jackc/pgx/v5"
)

func (s Store) GetDocument(
	ctx context.Context,
	principal identitydomain.Principal,
	documentID string,
) (knowledgedomain.Document, error) {
	var value knowledgedomain.Document
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, coalesce(site_id::text, ''),
		       document_type, title, authority, coalesce(source_reference, ''),
		       created_at, updated_at
		FROM documents
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (
		    COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id IS NULL OR site_id = ANY($3::uuid[])
		  )
	`, documentID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.Type, &value.Title,
		&value.Authority, &value.SourceReference, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return knowledgedomain.Document{}, knowledgeapp.ErrNotFound
	}
	if err != nil {
		return knowledgedomain.Document{}, err
	}
	value.Revisions, err = s.loadRevisions(ctx, value.ID)
	return value, err
}

func (s Store) ListDocuments(
	ctx context.Context,
	principal identitydomain.Principal,
	filter knowledgeapp.Filter,
) ([]knowledgedomain.Document, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text
		FROM documents
		WHERE organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR site_id IS NULL OR site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR site_id::text = nullif($3, ''))
		ORDER BY updated_at DESC
		LIMIT 100
	`, principal.OrganizationID, principal.SiteIDs, filter.SiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var documentID string
		if err := rows.Scan(&documentID); err != nil {
			return nil, err
		}
		ids = append(ids, documentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]knowledgedomain.Document, 0, len(ids))
	for _, documentID := range ids {
		value, err := s.GetDocument(ctx, principal, documentID)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s Store) loadRevisions(
	ctx context.Context,
	documentID string,
) ([]knowledgedomain.Revision, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, document_id::text, revision, approval_status,
		       effective_at, expires_at, coalesce(superseded_by_id::text, ''),
		       coalesce(attachment_id::text, ''), ingestion_state,
		       coalesce(ingestion_error, ''), coalesce(content_sha256, ''),
		       language, coalesce(approved_by::text, ''), approved_at, version
		FROM document_revisions
		WHERE document_id = $1::uuid
		ORDER BY created_at DESC
	`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []knowledgedomain.Revision
	for rows.Next() {
		var value knowledgedomain.Revision
		if err := rows.Scan(
			&value.ID, &value.DocumentID, &value.Revision, &value.ApprovalStatus,
			&value.EffectiveAt, &value.ExpiresAt, &value.SupersededByID,
			&value.AttachmentID, &value.IngestionState, &value.IngestionError,
			&value.ContentSHA256, &value.Language, &value.ApprovedBy,
			&value.ApprovedAt, &value.Version,
		); err != nil {
			return nil, err
		}
		value.Applicability, err = s.loadApplicability(ctx, value.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) loadApplicability(
	ctx context.Context,
	revisionID string,
) ([]knowledgedomain.Applicability, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, coalesce(site_id::text, ''), coalesce(asset_id::text, ''),
		       coalesce(asset_class, ''), coalesce(manufacturer, ''),
		       coalesce(model, ''), coalesce(process_service, '')
		FROM document_applicability
		WHERE revision_id = $1::uuid
		ORDER BY created_at, id
	`, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []knowledgedomain.Applicability
	for rows.Next() {
		var value knowledgedomain.Applicability
		if err := rows.Scan(
			&value.ID, &value.SiteID, &value.AssetID, &value.AssetClass,
			&value.Manufacturer, &value.Model, &value.ProcessService,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) RevisionScope(
	ctx context.Context,
	principal identitydomain.Principal,
	revisionID string,
) (string, string, error) {
	var siteID, documentID string
	err := s.Pool.QueryRow(ctx, `
		SELECT d.site_id::text, d.id::text
		FROM document_revisions r
		JOIN documents d ON d.id = r.document_id
		WHERE r.id = $1::uuid AND r.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR d.site_id = ANY($3::uuid[]))
	`, revisionID, principal.OrganizationID, principal.SiteIDs).Scan(&siteID, &documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", knowledgeapp.ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("load revision scope: %w", err)
	}
	return siteID, documentID, nil
}
