package application

import (
	"context"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

func TestImporterRejectsCrossTenantRecordBeforeProjection(t *testing.T) {
	t.Parallel()
	sink := &fakeSink{}
	importer := Importer{
		Connector: fakeConnector{page: integrationdomain.Page{
			Records: []integrationdomain.ExternalRecord{{
				Kind: "ASSET", OrganizationID: "organization-b",
				SiteID: "site-a", ExternalSystem: "SAP",
				ExternalID: "P-302", ObservedAt: time.Now(),
			}},
		}},
		Sink: sink,
	}
	_, err := importer.Pull(
		context.Background(),
		identitydomain.Principal{
			OrganizationID: "organization-a", SiteIDs: []string{"site-a"},
			Permissions: map[identitydomain.Permission]struct{}{
				identitydomain.PermissionExternalImport: {},
			},
		},
		"", 100,
	)
	if !errors.Is(err, ErrInvalid) || sink.called {
		t.Fatalf("error = %v, sink called = %v", err, sink.called)
	}
}

func TestImporterRejectsWriteCapableConnector(t *testing.T) {
	t.Parallel()
	importer := Importer{
		Connector: fakeConnector{writable: true},
		Sink:      &fakeSink{},
	}
	_, err := importer.Pull(
		context.Background(),
		identitydomain.Principal{
			OrganizationID: "organization-a",
			Permissions: map[identitydomain.Permission]struct{}{
				identitydomain.PermissionExternalImport: {},
			},
		},
		"", 100,
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want invalid", err)
	}
}

type fakeConnector struct {
	page     integrationdomain.Page
	writable bool
}

func (connector fakeConnector) Identity() integrationdomain.ConnectorIdentity {
	return integrationdomain.ConnectorIdentity{
		System: "SAP", Instance: "fixture", Version: "v1",
		ReadOnly: !connector.writable,
		Capabilities: []integrationdomain.Capability{
			integrationdomain.CapabilityReadAssets,
		},
	}
}

func (connector fakeConnector) Pull(
	context.Context,
	integrationdomain.PullRequest,
) (integrationdomain.Page, error) {
	return connector.page, nil
}

type fakeSink struct {
	called bool
}

func (sink *fakeSink) Apply(
	context.Context,
	identitydomain.Principal,
	[]integrationdomain.ExternalRecord,
) error {
	sink.called = true
	return nil
}
