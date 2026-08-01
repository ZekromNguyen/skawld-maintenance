package application

import (
	"context"
	"errors"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

var (
	ErrForbidden = errors.New("external import forbidden")
	ErrInvalid   = errors.New("external import invalid")
)

type ProjectionSink interface {
	Apply(
		context.Context,
		identitydomain.Principal,
		[]integrationdomain.ExternalRecord,
	) error
}

type Importer struct {
	Connector integrationdomain.ReadConnector
	Sink      ProjectionSink
}

func (i Importer) Pull(
	ctx context.Context,
	principal identitydomain.Principal,
	cursor string,
	limit int,
) (integrationdomain.Page, error) {
	if !principal.Has(identitydomain.PermissionExternalImport) {
		return integrationdomain.Page{}, ErrForbidden
	}
	if i.Connector == nil || i.Sink == nil || limit < 1 || limit > 500 {
		return integrationdomain.Page{}, ErrInvalid
	}
	if err := i.Connector.Identity().Validate(); err != nil {
		return integrationdomain.Page{}, errors.Join(ErrInvalid, err)
	}
	page, err := i.Connector.Pull(ctx, integrationdomain.PullRequest{
		OrganizationID: principal.OrganizationID,
		SiteIDs:        principal.SiteIDs,
		Cursor:         cursor,
		Limit:          limit,
	})
	if err != nil {
		return integrationdomain.Page{}, err
	}
	for _, record := range page.Records {
		if err := record.Validate(
			principal.OrganizationID,
			func(siteID string) bool {
				return principal.CanAccessSite(principal.OrganizationID, siteID)
			},
		); err != nil {
			return integrationdomain.Page{}, errors.Join(ErrInvalid, err)
		}
	}
	if err := i.Sink.Apply(ctx, principal, page.Records); err != nil {
		return integrationdomain.Page{}, err
	}
	return page, nil
}
