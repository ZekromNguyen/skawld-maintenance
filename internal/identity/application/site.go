package application

import (
	"context"
	"errors"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var ErrSiteNotFound = errors.New("site not found")

type Site struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Timezone       string `json:"timezone"`
	Status         string `json:"status"`
}

type SiteReader interface {
	Get(ctx context.Context, principal domain.Principal, siteID string) (Site, error)
}

type SiteService struct {
	Reader SiteReader
}

func (s SiteService) Get(
	ctx context.Context,
	principal domain.Principal,
	siteID string,
) (Site, error) {
	if s.Reader == nil || principal.OrganizationID == "" {
		return Site{}, ErrSiteNotFound
	}
	return s.Reader.Get(ctx, principal, siteID)
}
