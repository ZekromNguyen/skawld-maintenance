package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Capability string

const (
	CapabilityReadAssets             Capability = "READ_ASSETS"
	CapabilityReadWorkReferences     Capability = "READ_WORK_REFERENCES"
	CapabilityReadMaintenanceHistory Capability = "READ_MAINTENANCE_HISTORY"
	CapabilityReadDocumentMetadata   Capability = "READ_DOCUMENT_METADATA"
)

type ConnectorIdentity struct {
	System       string       `json:"system"`
	Instance     string       `json:"instance"`
	Version      string       `json:"version"`
	ReadOnly     bool         `json:"read_only"`
	Capabilities []Capability `json:"capabilities"`
}

type PullRequest struct {
	OrganizationID string   `json:"organization_id"`
	SiteIDs        []string `json:"site_ids"`
	Cursor         string   `json:"cursor,omitempty"`
	Limit          int      `json:"limit"`
}

type ExternalRecord struct {
	Kind            string                 `json:"kind"`
	OrganizationID  string                 `json:"organization_id"`
	SiteID          string                 `json:"site_id"`
	ExternalSystem  string                 `json:"external_system"`
	ExternalID      string                 `json:"external_id"`
	ExternalVersion string                 `json:"external_version,omitempty"`
	ObservedAt      time.Time              `json:"observed_at"`
	Attributes      map[string]interface{} `json:"attributes"`
}

type Page struct {
	Records    []ExternalRecord `json:"records"`
	NextCursor string           `json:"next_cursor,omitempty"`
	Complete   bool             `json:"complete"`
}

// ReadConnector is intentionally pull-only. It cannot create/approve work
// orders, permits, isolations, inventory movements, or operational actions.
type ReadConnector interface {
	Identity() ConnectorIdentity
	Pull(context.Context, PullRequest) (Page, error)
}

func (identity ConnectorIdentity) Validate() error {
	if strings.TrimSpace(identity.System) == "" ||
		strings.TrimSpace(identity.Instance) == "" ||
		strings.TrimSpace(identity.Version) == "" {
		return errors.New("connector identity is incomplete")
	}
	if !identity.ReadOnly {
		return errors.New("pilot connectors must be read-only")
	}
	if len(identity.Capabilities) == 0 {
		return errors.New("connector must declare a read capability")
	}
	for _, capability := range identity.Capabilities {
		switch capability {
		case CapabilityReadAssets,
			CapabilityReadWorkReferences,
			CapabilityReadMaintenanceHistory,
			CapabilityReadDocumentMetadata:
		default:
			return fmt.Errorf(
				"connector capability %q is not permitted", capability,
			)
		}
	}
	return nil
}

func (record ExternalRecord) Validate(
	organizationID string,
	siteAllowed func(string) bool,
) error {
	if record.OrganizationID != organizationID {
		return errors.New("connector returned a cross-tenant record")
	}
	if record.SiteID == "" || !siteAllowed(record.SiteID) {
		return errors.New("connector returned a record outside site scope")
	}
	switch record.Kind {
	case "ASSET", "WORK_REFERENCE", "MAINTENANCE_HISTORY", "DOCUMENT_METADATA":
	default:
		return fmt.Errorf("unsupported external record kind %q", record.Kind)
	}
	if strings.TrimSpace(record.ExternalSystem) == "" ||
		strings.TrimSpace(record.ExternalID) == "" ||
		record.ObservedAt.IsZero() {
		return errors.New("external provenance is incomplete")
	}
	return nil
}
