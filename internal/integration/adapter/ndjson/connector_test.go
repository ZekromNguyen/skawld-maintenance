package ndjson

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
)

func TestConnectorPaginatesOneImmutableSnapshot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "eam-export.ndjson")
	content := strings.Join([]string{
		`{"kind":"ASSET","organization_id":"org-a","site_id":"site-a","external_system":"SAP","external_id":"P-302","observed_at":"2026-07-26T00:00:00Z","attributes":{"name":"Pump P-302"}}`,
		`{"kind":"WORK_REFERENCE","organization_id":"org-a","site_id":"site-a","external_system":"SAP","external_id":"WO-42","observed_at":"2026-07-26T00:01:00Z","attributes":{"state":"OPEN"}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	connector, err := New(path, root, identity())
	if err != nil {
		t.Fatal(err)
	}
	first, err := connector.Pull(
		context.Background(),
		integrationdomain.PullRequest{Limit: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Complete || len(first.Records) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}
	second, err := connector.Pull(
		context.Background(),
		integrationdomain.PullRequest{Cursor: first.NextCursor, Limit: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Complete || len(second.Records) != 1 ||
		second.Records[0].ExternalID != "WO-42" {
		t.Fatalf("second page = %+v", second)
	}
}

func TestConnectorRejectsEscapesAndChangedSnapshots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.ndjson")
	if err := os.WriteFile(outside, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(outside, root, identity()); err == nil {
		t.Fatal("snapshot outside allowed root must be rejected")
	}

	path := filepath.Join(root, "export.ndjson")
	record := `{"kind":"ASSET","organization_id":"org-a","site_id":"site-a","external_system":"SAP","external_id":"P-302","observed_at":"2026-07-26T00:00:00Z","attributes":{}}`
	if err := os.WriteFile(path, []byte(record+"\n"+record+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	connector, err := New(path, root, identity())
	if err != nil {
		t.Fatal(err)
	}
	page, err := connector.Pull(
		context.Background(),
		integrationdomain.PullRequest{Limit: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(record+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := connector.Pull(
		context.Background(),
		integrationdomain.PullRequest{Cursor: page.NextCursor, Limit: 1},
	); err == nil {
		t.Fatal("cursor from a changed snapshot must be rejected")
	}
}

func identity() integrationdomain.ConnectorIdentity {
	return integrationdomain.ConnectorIdentity{
		System: "SAP", Instance: "pilot-export", Version: "v1",
		ReadOnly: true,
		Capabilities: []integrationdomain.Capability{
			integrationdomain.CapabilityReadAssets,
			integrationdomain.CapabilityReadWorkReferences,
		},
	}
}
