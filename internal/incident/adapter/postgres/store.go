package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	incidentdomain "github.com/ZekromNguyen/skawld-maintenance/internal/incident/domain"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	command incidentapp.CreateIncident,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	type outcome struct {
		Value  incidentapp.Incident
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, "incident.create.v1", key, hash, now)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay incidentapp.Incident
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, err
			}
			return outcome{Value: replay, Replay: true}, nil
		}
		var assetOrganizationID, assetSiteID, assetTag string
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text, tag
			FROM assets WHERE id = $1::uuid
		`, command.AssetID).Scan(&assetOrganizationID, &assetSiteID, &assetTag)
		if errors.Is(err, pgx.ErrNoRows) {
			return outcome{}, incidentapp.ErrNotFound
		}
		if err != nil {
			return outcome{}, err
		}
		if assetOrganizationID != principal.OrganizationID ||
			assetSiteID != command.SiteID ||
			!principal.CanAccessSite(assetOrganizationID, assetSiteID) {
			return outcome{}, incidentapp.ErrForbidden
		}
		incidentID := s.IDs.New()
		number := fmt.Sprintf("INC-%s-%s", now.UTC().Format("20060102"), strings.ToUpper(incidentID[:8]))
		incident, err := incidentdomain.New(incidentdomain.Incident{
			ID: incidentID, OrganizationID: principal.OrganizationID,
			SiteID: command.SiteID, AssetID: command.AssetID, Number: number,
			Summary: command.Summary, Severity: incidentdomain.Severity(command.Severity),
			SourceOfTruth:  integrationdomain.SourceOfTruth(command.SourceOfTruth),
			ExternalSystem: command.ExternalSystem, ExternalID: command.ExternalID,
			ExternalVersion: command.ExternalVersion, OccurredAt: command.OccurredAt,
			DetectedAt: command.DetectedAt,
		})
		if err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO incidents (
				id, organization_id, site_id, asset_id, number, summary, severity,
				state, source_of_truth, external_system, external_id, external_version,
				occurred_at, detected_at, version, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7,
				$8, $9, nullif($10, ''), nullif($11, ''), nullif($12, ''),
				$13, $14, $15, $16::uuid, $17, $17
			)
		`, incident.ID, incident.OrganizationID, incident.SiteID, incident.AssetID,
			incident.Number, incident.Summary, incident.Severity, incident.State,
			incident.SourceOfTruth, incident.ExternalSystem, incident.ExternalID,
			incident.ExternalVersion, incident.OccurredAt, incident.DetectedAt,
			incident.Version, principal.ID, now)
		if err != nil {
			return outcome{}, fmt.Errorf("insert incident: %w", err)
		}
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentCreated", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.created", EntityKind: "incident",
			EntityID: incident.ID, After: response, OccurredAt: now,
		}); err != nil {
			return outcome{}, err
		}
		body, _ := json.Marshal(response)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, "incident.create.v1", key,
			http.StatusCreated, body, now,
		); err != nil {
			return outcome{}, err
		}
		return outcome{Value: response}, nil
	})
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	return result.Value, result.Replay, nil
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
) (incidentapp.Incident, error) {
	value, err := scanIncident(s.Pool.QueryRow(ctx, incidentSelect+`
		WHERE i.id = $1::uuid
		  AND i.organization_id = $2::uuid
		  AND (cardinality($3::uuid[]) = 0 OR i.site_id = ANY($3::uuid[]))
	`, id, principal.OrganizationID, principal.SiteIDs))
	if errors.Is(err, pgx.ErrNoRows) {
		return incidentapp.Incident{}, incidentapp.ErrNotFound
	}
	return value, err
}

func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter incidentapp.Filter,
) ([]incidentapp.Incident, error) {
	rows, err := s.Pool.Query(ctx, incidentSelect+`
		WHERE i.organization_id = $1::uuid
		  AND (cardinality($2::uuid[]) = 0 OR i.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
		  AND (nullif($4, '') IS NULL OR i.asset_id = $4::uuid)
		  AND (nullif($5, '') IS NULL OR i.state = $5)
		ORDER BY i.detected_at DESC
		LIMIT 200
	`, principal.OrganizationID, principal.SiteIDs, filter.SiteID, filter.AssetID, filter.State)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []incidentapp.Incident
	for rows.Next() {
		value, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s Store) Resolve(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.ResolveIncident,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.resolve.v1:" + incidentID
	type outcome struct {
		Value  incidentapp.Incident
		Replay bool
	}
	result, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (outcome, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return outcome{}, err
		}
		if record.Replay {
			var replay incidentapp.Incident
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return outcome{}, err
			}
			return outcome{Value: replay, Replay: true}, nil
		}
		incident, assetTag, err := loadIncidentForUpdate(ctx, tx, principal, incidentID)
		if err != nil {
			return outcome{}, err
		}
		if incident.Version != command.ExpectedVersion {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		before := mapIncident(incident, assetTag)
		if err := incident.Resolve(command.ResolutionSummary, now); err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET state = $2, resolved_at = $3, resolution_summary = $4,
			    version = $5, updated_at = $3
			WHERE id = $1::uuid AND version = $6
		`, incident.ID, incident.State, incident.ResolvedAt,
			incident.ResolutionSummary, incident.Version, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentResolved", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.resolved", EntityKind: "incident",
			EntityID: incident.ID, Before: before, After: response, OccurredAt: now,
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
		return incidentapp.Incident{}, false, err
	}
	return result.Value, result.Replay, nil
}

const incidentSelect = `
	SELECT
		i.id::text, i.organization_id::text, i.site_id::text, i.asset_id::text,
		a.tag, i.number, i.summary, i.severity, i.state, i.source_of_truth,
		coalesce(i.external_system, ''), coalesce(i.external_id, ''),
		coalesce(i.external_version, ''), i.occurred_at, i.detected_at,
		i.resolved_at, coalesce(i.resolution_summary, ''), i.version
	FROM incidents i
	JOIN assets a ON a.id = i.asset_id
`

type scanner interface {
	Scan(...any) error
}

func scanIncident(row scanner) (incidentapp.Incident, error) {
	var value incidentapp.Incident
	err := row.Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.AssetID,
		&value.AssetTag, &value.Number, &value.Summary, &value.Severity,
		&value.State, &value.SourceOfTruth, &value.ExternalSystem,
		&value.ExternalID, &value.ExternalVersion, &value.OccurredAt,
		&value.DetectedAt, &value.ResolvedAt, &value.ResolutionSummary,
		&value.Version,
	)
	return value, err
}

func loadIncidentForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	incidentID string,
) (incidentdomain.Incident, string, error) {
	var value incidentapp.Incident
	err := tx.QueryRow(ctx, incidentSelect+`
		WHERE i.id = $1::uuid AND i.organization_id = $2::uuid
		FOR UPDATE OF i
	`, incidentID, principal.OrganizationID).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.AssetID,
		&value.AssetTag, &value.Number, &value.Summary, &value.Severity,
		&value.State, &value.SourceOfTruth, &value.ExternalSystem,
		&value.ExternalID, &value.ExternalVersion, &value.OccurredAt,
		&value.DetectedAt, &value.ResolvedAt, &value.ResolutionSummary,
		&value.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return incidentdomain.Incident{}, "", incidentapp.ErrNotFound
	}
	if err != nil {
		return incidentdomain.Incident{}, "", err
	}
	if !principal.CanAccessSite(value.OrganizationID, value.SiteID) {
		return incidentdomain.Incident{}, "", incidentapp.ErrForbidden
	}
	return incidentdomain.Incident{
		ID: value.ID, OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		AssetID: value.AssetID, Number: value.Number, Summary: value.Summary,
		Severity: incidentdomain.Severity(value.Severity), State: incidentdomain.State(value.State),
		SourceOfTruth:  integrationdomain.SourceOfTruth(value.SourceOfTruth),
		ExternalSystem: value.ExternalSystem, ExternalID: value.ExternalID,
		ExternalVersion: value.ExternalVersion, OccurredAt: value.OccurredAt,
		DetectedAt: value.DetectedAt, ResolvedAt: value.ResolvedAt,
		ResolutionSummary: value.ResolutionSummary, Version: value.Version,
	}, value.AssetTag, nil
}

func mapIncident(value incidentdomain.Incident, assetTag string) incidentapp.Incident {
	return incidentapp.Incident{
		ID: value.ID, OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		AssetID: value.AssetID, AssetTag: assetTag, Number: value.Number,
		Summary: value.Summary, Severity: string(value.Severity), State: string(value.State),
		SourceOfTruth: string(value.SourceOfTruth), ExternalSystem: value.ExternalSystem,
		ExternalID: value.ExternalID, ExternalVersion: value.ExternalVersion,
		OccurredAt: value.OccurredAt, DetectedAt: value.DetectedAt,
		ResolvedAt: value.ResolvedAt, ResolutionSummary: value.ResolutionSummary,
		Version: value.Version,
	}
}

func appendEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventID, organizationID, eventType, aggregateID string,
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
		) VALUES ($1::uuid, $2::uuid, $3, 'incident', $4::uuid, $5, $6::jsonb, $7)
	`, eventID, organizationID, eventType, aggregateID, version, encoded, occurredAt)
	return err
}
