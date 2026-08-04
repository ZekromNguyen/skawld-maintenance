package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	assetdomain "github.com/ZekromNguyen/skawld-maintenance/internal/asset/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	createAssetScope        = "asset.create.v1"
	approveCriticalityScope = "asset.criticality.approve.v1"
)

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Idempotency idempotency.Store
	Audit       audit.Sink
}

func (s Store) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command application.CreateAsset,
) (application.Asset, bool, error) {
	requestHash, err := idempotency.HashRequest(command)
	if err != nil {
		return application.Asset{}, false, err
	}
	type outcome struct {
		Asset  application.Asset
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(
			ctx, tx, principal.ID, createAssetScope, key, requestHash, now,
		)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay application.Asset
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, fmt.Errorf("decode asset replay: %w", err)
			}
			return outcome{Asset: replay, Replay: true}, nil
		}

		var siteOrganizationID string
		if err := tx.QueryRow(ctx,
			`SELECT organization_id::text FROM sites WHERE id = $1::uuid`,
			command.SiteID,
		).Scan(&siteOrganizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return outcome{}, application.ErrNotFound
			}
			return outcome{}, fmt.Errorf("load asset site: %w", err)
		}
		if siteOrganizationID != principal.OrganizationID ||
			!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
			return outcome{}, application.ErrForbidden
		}

		var external *assetdomain.ExternalReference
		if command.ExternalReference != nil {
			external = &assetdomain.ExternalReference{
				System:  strings.TrimSpace(command.ExternalReference.System),
				ID:      strings.TrimSpace(command.ExternalReference.ID),
				Version: strings.TrimSpace(command.ExternalReference.Version),
			}
		}
		asset, err := assetdomain.Create(assetdomain.NewAsset{
			ID:                s.IDs.New(),
			OrganizationID:    principal.OrganizationID,
			SiteID:            command.SiteID,
			Tag:               command.Tag,
			Name:              command.Name,
			Class:             command.Class,
			Manufacturer:      command.Manufacturer,
			Model:             command.Model,
			SourceOfTruth:     integrationdomain.SourceOfTruth(command.SourceOfTruth),
			ExternalReference: external,
			Now:               now,
		})
		if err != nil {
			return outcome{}, errors.Join(application.ErrInvalid, err)
		}
		var externalSystem, externalID, externalVersion *string
		if external != nil {
			externalSystem = &external.System
			externalID = &external.ID
			externalVersion = &external.Version
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO assets (
				id, organization_id, site_id, tag, name, asset_class,
				manufacturer, model, status, source_of_truth,
				external_system, external_id, external_version,
				version, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
				nullif($7, ''), nullif($8, ''), $9, $10,
				$11, $12, nullif($13, ''), $14, $15, $15
			)
		`, asset.ID, asset.OrganizationID, asset.SiteID, asset.Tag, asset.Name, asset.Class,
			asset.Manufacturer, asset.Model, asset.Status, asset.SourceOfTruth,
			externalSystem, externalID, externalVersion, asset.Version, now)
		if err != nil {
			return outcome{}, fmt.Errorf("insert asset: %w", err)
		}

		if command.ParentAssetID != "" {
			var parentOrganizationID, parentSiteID string
			err := tx.QueryRow(ctx, `
				SELECT organization_id::text, site_id::text
				FROM assets
				WHERE id = $1::uuid
			`, command.ParentAssetID).Scan(&parentOrganizationID, &parentSiteID)
			if errors.Is(err, pgx.ErrNoRows) {
				return outcome{}, application.ErrNotFound
			}
			if err != nil {
				return outcome{}, fmt.Errorf("load parent asset: %w", err)
			}
			if parentOrganizationID != asset.OrganizationID || parentSiteID != asset.SiteID {
				return outcome{}, application.ErrForbidden
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO asset_relationships (
					id, organization_id, parent_asset_id, child_asset_id,
					relationship_type, created_at
				) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'CONTAINS', $5)
			`, s.IDs.New(), asset.OrganizationID, command.ParentAssetID, asset.ID, now)
			if err != nil {
				return outcome{}, fmt.Errorf("insert asset relationship: %w", err)
			}
		}

		response := mapAsset(asset, command.ParentAssetID)
		for _, input := range command.Components {
			component, err := assetdomain.NewComponent(
				s.IDs.New(), asset.OrganizationID, asset.ID,
				input.Code, input.Name, input.Type,
			)
			if err != nil {
				return outcome{}, errors.Join(application.ErrInvalid, err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO asset_components (
					id, organization_id, asset_id, code, name, component_type, created_at
				) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
			`, component.ID, component.OrganizationID, component.AssetID,
				component.Code, component.Name, component.Type, now)
			if err != nil {
				return outcome{}, fmt.Errorf("insert asset component: %w", err)
			}
			response.Components = append(response.Components, application.Component{
				ID: component.ID, Code: component.Code, Name: component.Name, Type: component.Type,
			})
		}
		if err := appendEvent(ctx, tx, s.IDs.New(), asset.OrganizationID,
			"AssetCreated", "asset", asset.ID, asset.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID:             s.IDs.New(),
			OrganizationID: asset.OrganizationID,
			SiteID:         asset.SiteID,
			ActorID:        principal.ID,
			Action:         "asset.created",
			EntityKind:     "asset",
			EntityID:       asset.ID,
			After:          response,
			OccurredAt:     now,
		}); err != nil {
			return outcome{}, err
		}
		body, err := json.Marshal(response)
		if err != nil {
			return outcome{}, err
		}
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, createAssetScope, key,
			http.StatusCreated, body, now,
		); err != nil {
			return outcome{}, err
		}
		return outcome{Asset: response}, nil
	})
	if err != nil {
		return application.Asset{}, false, err
	}
	return result.Asset, result.Replay, nil
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	assetID string,
) (application.Asset, error) {
	asset, err := scanAsset(s.Pool.QueryRow(ctx, `
		SELECT
			a.id::text, a.organization_id::text, a.site_id::text,
			coalesce(r.parent_asset_id::text, ''), a.tag, a.name, a.asset_class,
			coalesce(a.manufacturer, ''), coalesce(a.model, ''), a.status,
			a.source_of_truth, coalesce(a.external_system, ''),
			coalesce(a.external_id, ''), coalesce(a.external_version, ''), a.version
		FROM assets a
		LEFT JOIN asset_relationships r
		  ON r.child_asset_id = a.id AND r.relationship_type = 'CONTAINS'
		WHERE a.id = $1::uuid
		  AND a.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR a.site_id = ANY($3::uuid[]))
	`, assetID, principal.OrganizationID, principal.SiteIDs))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.Asset{}, application.ErrNotFound
	}
	if err != nil {
		return application.Asset{}, fmt.Errorf("get asset: %w", err)
	}
	if err := s.loadDetails(ctx, &asset); err != nil {
		return application.Asset{}, err
	}
	return asset, nil
}

func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter application.Filter,
) ([]application.Asset, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT
			a.id::text, a.organization_id::text, a.site_id::text,
			coalesce(r.parent_asset_id::text, ''), a.tag, a.name, a.asset_class,
			coalesce(a.manufacturer, ''), coalesce(a.model, ''), a.status,
			a.source_of_truth, coalesce(a.external_system, ''),
			coalesce(a.external_id, ''), coalesce(a.external_version, ''), a.version
		FROM assets a
		LEFT JOIN asset_relationships r
		  ON r.child_asset_id = a.id AND r.relationship_type = 'CONTAINS'
		WHERE a.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR a.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR a.site_id = $3::uuid)
		  AND (
		      nullif($4, '') IS NULL
		      OR a.tag ILIKE '%' || $4 || '%'
		      OR a.name ILIKE '%' || $4 || '%'
		      OR a.asset_class ILIKE '%' || $4 || '%'
		  )
		ORDER BY a.tag
		LIMIT 200
	`, principal.OrganizationID, principal.SiteIDs, filter.SiteID, strings.TrimSpace(filter.Query))
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()
	var result []application.Asset
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}
	return result, nil
}

func (s Store) ApproveCriticality(
	ctx context.Context,
	principal identitydomain.Principal,
	key, assetID string,
	command application.ApproveCriticality,
) (application.Criticality, bool, error) {
	requestHash, err := idempotency.HashRequest(command)
	if err != nil {
		return application.Criticality{}, false, err
	}
	scope := approveCriticalityScope + ":" + assetID
	type outcome struct {
		Value  application.Criticality
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, requestHash, now)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay application.Criticality
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, err
			}
			return outcome{Value: replay, Replay: true}, nil
		}
		var organizationID, siteID string
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text FROM assets
			WHERE id = $1::uuid AND organization_id = $2::uuid
		`, assetID, principal.OrganizationID).Scan(&organizationID, &siteID)
		if errors.Is(err, pgx.ErrNoRows) {
			return outcome{}, application.ErrNotFound
		}
		if err != nil {
			return outcome{}, err
		}
		if !principal.CanAccessSite(organizationID, siteID) {
			return outcome{}, application.ErrForbidden
		}
		value, err := assetdomain.NewCriticality(assetdomain.Criticality{
			ID:                  s.IDs.New(),
			OrganizationID:      organizationID,
			AssetID:             assetID,
			Rating:              assetdomain.CriticalityRating(command.Rating),
			SafetyImpact:        command.SafetyImpact,
			ProductionImpact:    command.ProductionImpact,
			EnvironmentalImpact: command.EnvironmentalImpact,
			FinancialImpact:     command.FinancialImpact,
			Redundancy:          command.Redundancy,
			Rationale:           command.Rationale,
			ApprovedBy:          principal.ID,
			ApprovedAt:          now,
		})
		if err != nil {
			return outcome{}, errors.Join(application.ErrInvalid, err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE asset_criticalities SET superseded_at = $2
			WHERE asset_id = $1::uuid AND superseded_at IS NULL
		`, assetID, now); err != nil {
			return outcome{}, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO asset_criticalities (
				id, organization_id, asset_id, rating, safety_impact,
				production_impact, environmental_impact, financial_impact,
				redundancy, rationale, approved_by, approved_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8,
				$9, $10, $11::uuid, $12
			)
		`, value.ID, value.OrganizationID, value.AssetID, value.Rating,
			value.SafetyImpact, value.ProductionImpact, value.EnvironmentalImpact,
			value.FinancialImpact, value.Redundancy, value.Rationale,
			value.ApprovedBy, value.ApprovedAt)
		if err != nil {
			return outcome{}, err
		}
		response := application.Criticality{
			Rating: string(value.Rating), SafetyImpact: value.SafetyImpact,
			ProductionImpact:    value.ProductionImpact,
			EnvironmentalImpact: value.EnvironmentalImpact,
			FinancialImpact:     value.FinancialImpact, Redundancy: value.Redundancy,
			Rationale: value.Rationale, ApprovedBy: value.ApprovedBy,
			ApprovedAt: value.ApprovedAt,
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "asset.criticality.approved",
			EntityKind: "asset", EntityID: assetID, After: response, OccurredAt: now,
		}); err != nil {
			return outcome{}, err
		}
		body, _ := json.Marshal(response)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return outcome{}, err
		}
		return outcome{Value: response}, nil
	})
	if err != nil {
		return application.Criticality{}, false, err
	}
	return result.Value, result.Replay, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAsset(row scanner) (application.Asset, error) {
	var asset application.Asset
	var externalSystem, externalID, externalVersion string
	err := row.Scan(
		&asset.ID, &asset.OrganizationID, &asset.SiteID, &asset.ParentAssetID,
		&asset.Tag, &asset.Name, &asset.Class, &asset.Manufacturer, &asset.Model,
		&asset.Status, &asset.SourceOfTruth, &externalSystem, &externalID,
		&externalVersion, &asset.Version,
	)
	if err != nil {
		return application.Asset{}, err
	}
	if externalSystem != "" {
		asset.ExternalReference = &application.ExternalReference{
			System: externalSystem, ID: externalID, Version: externalVersion,
		}
	}
	asset.Components = []application.Component{}
	return asset, nil
}

func mapAsset(asset assetdomain.Asset, parentID string) application.Asset {
	result := application.Asset{
		ID: asset.ID, OrganizationID: asset.OrganizationID, SiteID: asset.SiteID,
		ParentAssetID: parentID, Tag: asset.Tag, Name: asset.Name, Class: asset.Class,
		Manufacturer: asset.Manufacturer, Model: asset.Model, Status: string(asset.Status),
		SourceOfTruth: string(asset.SourceOfTruth), Version: asset.Version,
		Components: []application.Component{},
	}
	if asset.ExternalReference != nil {
		result.ExternalReference = &application.ExternalReference{
			System:  asset.ExternalReference.System,
			ID:      asset.ExternalReference.ID,
			Version: asset.ExternalReference.Version,
		}
	}
	return result
}

func (s Store) loadDetails(ctx context.Context, asset *application.Asset) error {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, code, name, component_type
		FROM asset_components WHERE asset_id = $1::uuid ORDER BY code
	`, asset.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var component application.Component
		if err := rows.Scan(&component.ID, &component.Code, &component.Name, &component.Type); err != nil {
			return err
		}
		asset.Components = append(asset.Components, component)
	}
	var criticality application.Criticality
	err = s.Pool.QueryRow(ctx, `
		SELECT rating, safety_impact, production_impact, environmental_impact,
		       financial_impact, redundancy, rationale, approved_by::text, approved_at
		FROM asset_criticalities
		WHERE asset_id = $1::uuid AND superseded_at IS NULL
	`, asset.ID).Scan(
		&criticality.Rating, &criticality.SafetyImpact,
		&criticality.ProductionImpact, &criticality.EnvironmentalImpact,
		&criticality.FinancialImpact, &criticality.Redundancy,
		&criticality.Rationale, &criticality.ApprovedBy, &criticality.ApprovedAt,
	)
	if err == nil {
		asset.Criticality = &criticality
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func appendEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventID, organizationID, eventType, aggregateType, aggregateID string,
	version int64,
	payload any,
	occurredAt time.Time,
) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO domain_events (
			id, organization_id, event_type, aggregate_type, aggregate_id,
			aggregate_version, payload, occurred_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7::jsonb, $8)
	`, eventID, organizationID, eventType, aggregateType, aggregateID,
		version, encoded, occurredAt)
	return err
}
