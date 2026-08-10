package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SiteReader struct {
	Pool *pgxpool.Pool
}

func (r SiteReader) Get(
	ctx context.Context,
	principal domain.Principal,
	siteID string,
) (application.Site, error) {
	var site application.Site
	err := r.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, code, name, timezone, status
		FROM sites
		WHERE id = $1::uuid
		  AND organization_id = $2::uuid
		  AND (
		      COALESCE(cardinality($3::uuid[]), 0) = 0
		      OR id = ANY($3::uuid[])
		  )
	`, siteID, principal.OrganizationID, principal.SiteIDs).Scan(
		&site.ID,
		&site.OrganizationID,
		&site.Code,
		&site.Name,
		&site.Timezone,
		&site.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Site{}, application.ErrSiteNotFound
	}
	if err != nil {
		return application.Site{}, fmt.Errorf("get authorized site: %w", err)
	}
	if !principal.CanAccessSite(site.OrganizationID, site.ID) {
		return application.Site{}, application.ErrSiteNotFound
	}
	return site, nil
}
