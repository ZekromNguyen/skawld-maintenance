package postgres

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthorityReader struct {
	Pool *pgxpool.Pool
}

func (r AuthorityReader) ForSubject(
	ctx context.Context,
	subjectID, organizationID, siteID string,
) ([]identitydomain.ApprovalAuthority, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT
			id::text,
			subject_id::text,
			organization_id::text,
			coalesce(site_id::text, ''),
			coalesce(scope_kind, ''),
			coalesce(scope_id, ''),
			coalesce(competency, ''),
			maximum_risk,
			valid_from,
			valid_until,
			revoked_at,
			coalesce(delegated_by_subject_id::text, '')
		FROM approval_authorities
		WHERE subject_id = $1::uuid
		  AND organization_id = $2::uuid
		  AND (site_id IS NULL OR site_id::text = nullif($3, ''))
	`, subjectID, organizationID, siteID)
	if err != nil {
		return nil, fmt.Errorf("query approval authorities: %w", err)
	}
	defer rows.Close()
	var result []identitydomain.ApprovalAuthority
	for rows.Next() {
		var authority identitydomain.ApprovalAuthority
		var validFrom, validUntil *time.Time
		var maximumRisk int
		if err := rows.Scan(
			&authority.ID,
			&authority.SubjectID,
			&authority.OrganizationID,
			&authority.SiteID,
			&authority.ScopeKind,
			&authority.ScopeID,
			&authority.Competency,
			&maximumRisk,
			&validFrom,
			&validUntil,
			&authority.RevokedAt,
			&authority.DelegatedBySubjectID,
		); err != nil {
			return nil, fmt.Errorf("scan approval authority: %w", err)
		}
		authority.MaximumRisk = identitydomain.RiskLevel(maximumRisk)
		if validFrom != nil {
			authority.ValidFrom = *validFrom
		}
		if validUntil != nil {
			authority.ValidUntil = *validUntil
		}
		result = append(result, authority)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate approval authorities: %w", err)
	}
	return result, nil
}
