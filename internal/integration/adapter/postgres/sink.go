package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/integration/application"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sink projects validated external records into Skawld-owned tables. It
// never re-owns, modifies, or deletes an OWNED_BY_SKAWLD record, and it
// rejects record kinds it does not know how to project.
type Sink struct {
	Pool  *pgxpool.Pool
	IDs   id.Generator
	Clock clock.Clock
	Audit audit.Sink
}

var _ application.ProjectionSink = Sink{}

func (s Sink) Apply(
	ctx context.Context,
	principal identitydomain.Principal,
	records []integrationdomain.ExternalRecord,
) error {
	now := s.Clock.Now()
	type outcome struct{}
	_, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		for _, record := range records {
			if record.Kind != "ASSET" {
				return outcome{}, fmt.Errorf(
					"sink cannot project external record kind %q", record.Kind,
				)
			}
			tag, err := requiredString(record.Attributes, "tag")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			name, err := requiredString(record.Attributes, "name")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			assetClass, err := requiredString(record.Attributes, "asset_class")
			if err != nil {
				return outcome{}, fmt.Errorf("external asset %s: %w", record.ExternalID, err)
			}
			attributesJSON, err := json.Marshal(record.Attributes)
			if err != nil {
				return outcome{}, fmt.Errorf("encode external asset attributes: %w", err)
			}

			var existingSource, existingVersion string
			err = tx.QueryRow(ctx, `
				SELECT source_of_truth, coalesce(external_version, '')
				FROM assets
				WHERE organization_id = $1::uuid
				  AND external_system = $2
				  AND external_id = $3
			`, principal.OrganizationID, record.ExternalSystem, record.ExternalID).
				Scan(&existingSource, &existingVersion)
			switch {
			case err == nil && existingSource == "OWNED_BY_SKAWLD":
				// An owned record can never be re-owned or modified by an
				// external projection.
				continue
			case err == nil && existingVersion == record.ExternalVersion:
				continue
			case err == nil:
				if _, err := tx.Exec(ctx, `
					UPDATE assets
					SET name = $1, asset_class = $2, external_version = $3,
					    attributes = $4, sync_status = 'IN_SYNC',
					    version = version + 1, updated_at = $5
					WHERE organization_id = $6::uuid
					  AND external_system = $7
					  AND external_id = $8
				`, name, assetClass, record.ExternalVersion, attributesJSON, now,
					principal.OrganizationID, record.ExternalSystem, record.ExternalID); err != nil {
					return outcome{}, fmt.Errorf("update external asset: %w", err)
				}
			case errors.Is(err, pgx.ErrNoRows):
				if _, err := tx.Exec(ctx, `
					INSERT INTO assets (
						id, organization_id, site_id, tag, name, asset_class,
						manufacturer, model, status, source_of_truth,
						external_system, external_id, external_version, sync_status,
						attributes, version, created_at, updated_at
					) VALUES (
						$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
						nullif($7, ''), nullif($8, ''), coalesce(nullif($9, ''), 'ACTIVE'),
						'EXTERNAL_REFERENCE', $10, $11, $12, 'IN_SYNC',
						$13, 1, $14, $14
					)
				`, s.IDs.New(), principal.OrganizationID, record.SiteID, tag, name, assetClass,
					optionalString(record.Attributes, "manufacturer"),
					optionalString(record.Attributes, "model"),
					optionalString(record.Attributes, "status"),
					record.ExternalSystem, record.ExternalID, record.ExternalVersion,
					attributesJSON, now); err != nil {
					return outcome{}, fmt.Errorf("insert external asset: %w", err)
				}
			default:
				return outcome{}, fmt.Errorf("lookup external asset: %w", err)
			}
			if err := s.Audit.Append(ctx, tx, audit.Event{
				ID:             s.IDs.New(),
				OrganizationID: principal.OrganizationID,
				SiteID:         record.SiteID,
				ActorID:        principal.ID,
				Action:         "integration.external-asset-projected",
				EntityKind:     "asset",
				EntityID:       record.ExternalID,
				Attributes: map[string]any{
					"external_system":  record.ExternalSystem,
					"external_version": record.ExternalVersion,
				},
				OccurredAt: now,
			}); err != nil {
				return outcome{}, err
			}
		}
		return outcome{}, nil
	})
	return err
}

func requiredString(attributes map[string]any, key string) (string, error) {
	value := strings.TrimSpace(fmt.Sprint(attributes[key]))
	if value == "" {
		return "", fmt.Errorf("missing required attribute %q", key)
	}
	return value, nil
}

func optionalString(attributes map[string]any, key string) string {
	value, ok := attributes[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
