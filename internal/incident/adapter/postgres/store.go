package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
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
		reporterID := command.ReporterID
		if reporterID == "" {
			reporterID = principal.ID
		}
		occurredAt := command.OccurredAt
		if occurredAt == nil {
			occurredAt = &command.DetectedAt
		}
		incidentID := s.IDs.New()
		number := fmt.Sprintf("INC-%s-%s", now.UTC().Format("20060102"), strings.ToUpper(incidentID[:8]))
		customValues := command.CustomValues
		if customValues == nil {
			customValues = map[string]any{}
		}
		incident, err := incidentdomain.New(incidentdomain.Incident{
			ID: incidentID, OrganizationID: principal.OrganizationID,
			SiteID: command.SiteID, AssetID: command.AssetID, Number: number,
			Summary: command.Summary, Details: command.Details,
			Priority:   incidentdomain.Priority(command.Priority),
			Status:     incidentdomain.Status(command.Status),
			AssigneeID: command.AssigneeID, ReporterID: reporterID, TeamID: command.TeamID,
			SourceOfTruth:  integrationdomain.SourceOfTruth(command.SourceOfTruth),
			ExternalSystem: command.ExternalSystem, ExternalID: command.ExternalID,
			ExternalVersion: command.ExternalVersion, OccurredAt: occurredAt,
			DetectedAt: command.DetectedAt, CustomValues: customValues,
		})
		if err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		encodedCustom, err := json.Marshal(customValues)
		if err != nil {
			return outcome{}, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO incidents (
				id, organization_id, site_id, asset_id, number, summary, details, priority,
				status, source_of_truth, external_system, external_id, external_version,
				occurred_at, detected_at, version, created_by, assignee_id, reporter_id, team_id,
				custom_values, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, nullif($7, ''), $8,
				$9, $10, nullif($11, ''), nullif($12, ''), nullif($13, ''),
				$14, $15, $16, $17::uuid, nullif($18, '')::uuid, $19::uuid, nullif($20, '')::uuid,
				$21::jsonb, $22, $22
			)
		`, incident.ID, incident.OrganizationID, incident.SiteID, incident.AssetID,
			incident.Number, incident.Summary, incident.Details, incident.Priority,
			incident.Status, incident.SourceOfTruth, incident.ExternalSystem,
			incident.ExternalID, incident.ExternalVersion, incident.OccurredAt,
			incident.DetectedAt, incident.Version, principal.ID, incident.AssigneeID,
			incident.ReporterID, incident.TeamID, encodedCustom, now)
		if err != nil {
			return outcome{}, fmt.Errorf("insert incident: %w", err)
		}
		for definitionID, value := range command.CustomValues {
			encoded, err := json.Marshal(value)
			if err != nil {
				return outcome{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO custom_value_history (
					incident_id, field_definition_id, principal_id, value_before, value_after, changed_at
				) VALUES ($1::uuid, $2::uuid, $3::uuid, NULL, $4::jsonb, $5)
			`, incident.ID, definitionID, principal.ID, encoded, now); err != nil {
				return outcome{}, err
			}
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
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR i.site_id = ANY($3::uuid[]))
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
) ([]incidentapp.Incident, bool, error) {
	if filter.PageSize < 1 {
		filter.PageSize = 25
	}
	query := incidentSelect + `
		WHERE i.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
		  AND (nullif($4, '') IS NULL OR i.asset_id = $4::uuid)
		  AND (nullif($5, '') IS NULL OR i.status = $5)`
	args := []any{
		principal.OrganizationID, principal.SiteIDs,
		filter.SiteID, filter.AssetID, filter.Status,
	}
	if len(filter.CustomFields) > 0 {
		defIDs := make([]string, 0, len(filter.CustomFields))
		for defID := range filter.CustomFields {
			defIDs = append(defIDs, defID)
		}
		sort.Strings(defIDs)
		for _, defID := range defIDs {
			filterValue := filter.CustomFields[defID]
			keyArg := len(args) + 1
			valueArg := len(args) + 2
			args = append(args, defID, filterValue)
			query += fmt.Sprintf(" AND i.custom_values ? $%d::text AND i.custom_values->>$%d::text = $%d", keyArg, keyArg, valueArg)
		}
	}
	if len(filter.CustomFieldRanges) > 0 {
		defIDs := make([]string, 0, len(filter.CustomFieldRanges))
		for defID := range filter.CustomFieldRanges {
			defIDs = append(defIDs, defID)
		}
		sort.Strings(defIDs)
		for _, defID := range defIDs {
			keyArg := len(args) + 1
			args = append(args, defID)
			query += fmt.Sprintf(" AND i.custom_values ? $%d::text", keyArg)
			rangeFilter := filter.CustomFieldRanges[defID]
			if rangeFilter.Min != nil {
				args = append(args, *rangeFilter.Min)
				query += fmt.Sprintf(" AND (i.custom_values->>$%d::text)::numeric >= $%d", keyArg, len(args))
			}
			if rangeFilter.Max != nil {
				args = append(args, *rangeFilter.Max)
				query += fmt.Sprintf(" AND (i.custom_values->>$%d::text)::numeric <= $%d", keyArg, len(args))
			}
		}
	}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, incidentapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (i.detected_at, i.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY i.detected_at DESC, i.id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	result := make([]incidentapp.Incident, 0, filter.PageSize+1)
	for rows.Next() {
		value, err := scanIncident(rows)
		if err != nil {
			return nil, false, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(result) > filter.PageSize
	if hasMore {
		result = result[:filter.PageSize]
	}
	return result, hasMore, nil
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
			SET status = $2, resolved_at = $3, resolution_summary = $4,
			    version = $5, updated_at = $3
			WHERE id = $1::uuid AND version = $6
		`, incident.ID, incident.Status, incident.ResolvedAt,
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

func (s Store) Close(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.CloseIncident,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.close.v1:" + incidentID
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
		if err := incident.Close(now); err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET status = $2, resolved_at = $3, version = $4, updated_at = $5
			WHERE id = $1::uuid AND version = $6
		`, incident.ID, incident.Status, incident.ResolvedAt,
			incident.Version, now, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentClosed", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.closed", EntityKind: "incident",
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

func (s Store) UpdateCustomValues(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.UpdateCustomValues,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.custom_values.v1:" + incidentID
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
		if len(command.CustomValues) == 0 {
			// Nothing to change: return the current incident without a
			// version bump, event, or audit row so a no-op update cannot
			// invalidate the client's optimistic-lock version.
			body, _ := json.Marshal(before)
			if err := s.Idempotency.Complete(
				ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
			); err != nil {
				return outcome{}, err
			}
			return outcome{Value: before}, nil
		}

		previous := incident.CustomValues
		if previous == nil {
			previous = map[string]any{}
		}
		changed := make(map[string]any, len(command.CustomValues))
		for defID, value := range command.CustomValues {
			if !reflect.DeepEqual(previous[defID], value) {
				changed[defID] = value
			}
		}
		if len(changed) == 0 {
			// No value actually changed: return the current incident without
			// a version bump, history row, event, or audit entry.
			body, _ := json.Marshal(before)
			if err := s.Idempotency.Complete(
				ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
			); err != nil {
				return outcome{}, err
			}
			return outcome{Value: before}, nil
		}
		merged := make(map[string]any, len(previous)+len(changed))
		for defID, value := range previous {
			merged[defID] = value
		}
		for defID, value := range changed {
			merged[defID] = value
		}
		encoded, err := json.Marshal(merged)
		if err != nil {
			return outcome{}, err
		}
		newVersion := incident.Version + 1
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET custom_values = $2::jsonb, version = $3, updated_at = $4
			WHERE id = $1::uuid AND version = $5
		`, incident.ID, encoded, newVersion, now, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		for defID, value := range changed {
			encodedValue, err := json.Marshal(value)
			if err != nil {
				return outcome{}, err
			}
			beforeEncoded, err := json.Marshal(previous[defID])
			if err != nil {
				return outcome{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO custom_value_history (
					incident_id, field_definition_id, principal_id, value_before, value_after, changed_at
				) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::jsonb, $5::jsonb, $6)
			`, incident.ID, defID, principal.ID, beforeEncoded, encodedValue, now); err != nil {
				return outcome{}, err
			}
		}
		incident.CustomValues = merged
		incident.Version = newVersion
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentCustomValuesUpdated", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.custom_values.updated", EntityKind: "incident",
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

func (s Store) Reopen(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command incidentapp.ReopenIncident,
) (incidentapp.Incident, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return incidentapp.Incident{}, false, err
	}
	scope := "incident.reopen.v1:" + incidentID
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
		if err := incident.Reopen(); err != nil {
			return outcome{}, errors.Join(incidentapp.ErrInvalid, err)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE incidents
			SET status = $2, resolved_at = NULL, resolution_summary = NULL,
			    version = $3, updated_at = $4
			WHERE id = $1::uuid AND version = $5
		`, incident.ID, incident.Status, incident.Version, now, command.ExpectedVersion)
		if err != nil {
			return outcome{}, err
		}
		if tag.RowsAffected() != 1 {
			return outcome{}, incidentapp.ErrVersionConflict
		}
		response := mapIncident(incident, assetTag)
		if err := appendEvent(ctx, tx, s.IDs.New(), incident.OrganizationID,
			"IncidentReopened", incident.ID, incident.Version, response, now); err != nil {
			return outcome{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: incident.OrganizationID, SiteID: incident.SiteID,
			ActorID: principal.ID, Action: "incident.reopened", EntityKind: "incident",
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
		a.tag, i.number, i.summary, i.priority, i.status, i.source_of_truth,
		coalesce(i.external_system, ''), coalesce(i.external_id, ''),
		coalesce(i.external_version, ''), i.occurred_at, i.detected_at,
		i.resolved_at, coalesce(i.resolution_summary, ''), i.version,
		coalesce(i.details, ''),
		coalesce(i.assignee_id::text, ''), coalesce(assignee.display_name, ''),
		coalesce(i.reporter_id::text, ''), coalesce(reporter.display_name, ''),
		coalesce(i.team_id::text, ''), coalesce(team.name, ''),
		i.custom_values
	FROM incidents i
	JOIN assets a ON a.id = i.asset_id
	LEFT JOIN principals assignee ON assignee.id = i.assignee_id
	LEFT JOIN principals reporter ON reporter.id = i.reporter_id
	LEFT JOIN teams team ON team.id = i.team_id
`

type scanner interface {
	Scan(...any) error
}

func scanIncident(row scanner) (incidentapp.Incident, error) {
	var value incidentapp.Incident
	var rawCustom []byte
	err := row.Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.AssetID,
		&value.AssetTag, &value.Number, &value.Summary, &value.Priority,
		&value.Status, &value.SourceOfTruth, &value.ExternalSystem,
		&value.ExternalID, &value.ExternalVersion, &value.OccurredAt,
		&value.DetectedAt, &value.ResolvedAt, &value.ResolutionSummary,
		&value.Version, &value.Details, &value.AssigneeID, &value.AssigneeName,
		&value.ReporterID, &value.ReporterName, &value.TeamID, &value.TeamName,
		&rawCustom,
	)
	if err != nil {
		return incidentapp.Incident{}, err
	}
	value.CustomValues = map[string]any{}
	if len(rawCustom) > 0 {
		if err := json.Unmarshal(rawCustom, &value.CustomValues); err != nil {
			return incidentapp.Incident{}, err
		}
	}
	value.TimeToCompleteSeconds = timeToCompleteSeconds(value.DetectedAt, value.ResolvedAt)
	return value, nil
}

func loadIncidentForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	incidentID string,
) (incidentdomain.Incident, string, error) {
	var value incidentapp.Incident
	var rawCustom []byte
	err := tx.QueryRow(ctx, incidentSelect+`
		WHERE i.id = $1::uuid AND i.organization_id = $2::uuid
		FOR UPDATE OF i
	`, incidentID, principal.OrganizationID).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.AssetID,
		&value.AssetTag, &value.Number, &value.Summary, &value.Priority,
		&value.Status, &value.SourceOfTruth, &value.ExternalSystem,
		&value.ExternalID, &value.ExternalVersion, &value.OccurredAt,
		&value.DetectedAt, &value.ResolvedAt, &value.ResolutionSummary,
		&value.Version, &value.Details, &value.AssigneeID, &value.AssigneeName,
		&value.ReporterID, &value.ReporterName, &value.TeamID, &value.TeamName,
		&rawCustom,
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
	customValues := map[string]any{}
	if len(rawCustom) > 0 {
		if err := json.Unmarshal(rawCustom, &customValues); err != nil {
			return incidentdomain.Incident{}, "", err
		}
	}
	return incidentdomain.Incident{
		ID: value.ID, OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		AssetID: value.AssetID, Number: value.Number, Summary: value.Summary,
		Details: value.Details, Priority: incidentdomain.Priority(value.Priority),
		Status: incidentdomain.Status(value.Status), AssigneeID: value.AssigneeID,
		ReporterID: value.ReporterID, TeamID: value.TeamID,
		SourceOfTruth:  integrationdomain.SourceOfTruth(value.SourceOfTruth),
		ExternalSystem: value.ExternalSystem, ExternalID: value.ExternalID,
		ExternalVersion: value.ExternalVersion, OccurredAt: value.OccurredAt,
		DetectedAt: value.DetectedAt, ResolvedAt: value.ResolvedAt,
		ResolutionSummary: value.ResolutionSummary, CustomValues: customValues,
		Version: value.Version,
	}, value.AssetTag, nil
}

func mapIncident(value incidentdomain.Incident, assetTag string) incidentapp.Incident {
	return incidentapp.Incident{
		ID: value.ID, OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		AssetID: value.AssetID, AssetTag: assetTag, Number: value.Number,
		Summary: value.Summary, Details: value.Details, Priority: string(value.Priority),
		Status: string(value.Status), AssigneeID: value.AssigneeID,
		ReporterID: value.ReporterID, TeamID: value.TeamID,
		SourceOfTruth: string(value.SourceOfTruth), ExternalSystem: value.ExternalSystem,
		ExternalID: value.ExternalID, ExternalVersion: value.ExternalVersion,
		OccurredAt: value.OccurredAt, DetectedAt: value.DetectedAt,
		ResolvedAt: value.ResolvedAt, ResolutionSummary: value.ResolutionSummary,
		CustomValues:          value.CustomValues,
		TimeToCompleteSeconds: timeToCompleteSeconds(value.DetectedAt, value.ResolvedAt),
		Version:               value.Version,
	}
}

func timeToCompleteSeconds(detectedAt time.Time, resolvedAt *time.Time) *int64 {
	if resolvedAt == nil {
		return nil
	}
	seconds := int64(resolvedAt.Sub(detectedAt).Seconds())
	return &seconds
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
