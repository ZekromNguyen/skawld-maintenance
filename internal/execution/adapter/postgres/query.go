package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/jackc/pgx/v5"
)

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	executionID string,
) (executionapp.Execution, error) {
	value, err := loadExecution(ctx, s.Pool, principal, executionID, false)
	return value, mapStoreError(err)
}

func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter executionapp.Filter,
) ([]executionapp.Execution, bool, error) {
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	query := `
		SELECT e.id::text
		FROM maintenance_executions e
		WHERE e.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR e.site_id = ANY($2::uuid[]))
		  AND (nullif($3, '') IS NULL OR e.site_id = $3::uuid)
		  AND (nullif($4, '') IS NULL OR e.state = $4)
		  AND (nullif($5, '') IS NULL OR e.assigned_to = $5::uuid)`
	args := []any{principal.OrganizationID, principal.SiteIDs,
		filter.SiteID, filter.State, filter.AssignedTo}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, executionapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (e.updated_at, e.id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY e.updated_at DESC, e.id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	var ids []string
	for rows.Next() {
		var executionID string
		if err := rows.Scan(&executionID); err != nil {
			rows.Close()
			return nil, false, err
		}
		ids = append(ids, executionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, false, err
	}
	rows.Close()
	hasMore := len(ids) > filter.PageSize
	if hasMore {
		ids = ids[:filter.PageSize]
	}
	result := make([]executionapp.Execution, 0, len(ids))
	for _, executionID := range ids {
		value, err := loadExecution(ctx, s.Pool, principal, executionID, false)
		if err != nil {
			return nil, false, err
		}
		result = append(result, value)
	}
	return result, hasMore, nil
}

func loadExecution(
	ctx context.Context,
	q queryer,
	principal identitydomain.Principal,
	executionID string,
	forUpdate bool,
) (executionapp.Execution, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE OF e"
	}
	var value executionapp.Execution
	err := q.QueryRow(ctx, `
		SELECT e.id::text, e.organization_id::text, e.site_id::text,
		       coalesce(e.incident_id::text, ''), coalesce(i.number, ''),
		       e.asset_id::text, a.tag,
		       e.purpose, e.state, coalesce(e.assigned_to::text, ''), e.version,
		       e.started_at, e.completed_at, coalesce(e.outcome_summary, ''),
		       e.updated_at
		FROM maintenance_executions e
		JOIN assets a ON a.id = e.asset_id
		LEFT JOIN incidents i ON i.id = e.incident_id
		WHERE e.id = $1::uuid
		  AND e.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR e.site_id = ANY($3::uuid[]))
	`+suffix, executionID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.IncidentID,
		&value.IncidentNumber, &value.AssetID, &value.AssetTag, &value.Purpose,
		&value.State, &value.AssignedTo, &value.Version, &value.StartedAt,
		&value.CompletedAt, &value.OutcomeSummary, &value.UpdatedAt,
	)
	if err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadSteps(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadPrerequisites(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadMeasurements(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadObservations(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadActions(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	if err := loadDecisions(ctx, q, &value); err != nil {
		return executionapp.Execution{}, err
	}
	return value, nil
}

func loadSteps(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, step_key, sequence, title, state, risk_level,
		       coalesce(required_prerequisite, ''), coalesce(blocked_reason, ''),
		       started_at, completed_at, version
		FROM execution_steps
		WHERE execution_id = $1::uuid
		ORDER BY sequence
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Steps = []executionapp.Step{}
	for rows.Next() {
		var step executionapp.Step
		if err := rows.Scan(
			&step.ID, &step.Key, &step.Sequence, &step.Title, &step.State,
			&step.RiskLevel, &step.RequiredPrerequisite, &step.BlockedReason,
			&step.StartedAt, &step.CompletedAt, &step.Version,
		); err != nil {
			return err
		}
		target.Steps = append(target.Steps, step)
	}
	return rows.Err()
}

func loadPrerequisites(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT prerequisite_type, status, external_reference, verified_by::text,
		       verified_at, valid_until
		FROM prerequisite_verifications
		WHERE execution_id = $1::uuid
		ORDER BY verified_at DESC
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Prerequisites = []executionapp.Prerequisite{}
	for rows.Next() {
		var value executionapp.Prerequisite
		if err := rows.Scan(
			&value.Type, &value.Status, &value.ExternalReference,
			&value.VerifiedBy, &value.VerifiedAt, &value.ValidUntil,
		); err != nil {
			return err
		}
		target.Prerequisites = append(target.Prerequisites, value)
	}
	return rows.Err()
}

func loadMeasurements(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, coalesce(component_id::text, ''), measurement_type,
		       value::text, unit, original_value, original_unit, source,
		       data_quality, verification_status, observed_at,
		       coalesce(client_event_id::text, ''), created_at_device, received_at_server
		FROM measurements
		WHERE execution_id = $1::uuid
		ORDER BY observed_at, created_at
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Measurements = []executionapp.Measurement{}
	for rows.Next() {
		var value executionapp.Measurement
		if err := rows.Scan(
			&value.ID, &value.ComponentID, &value.MeasurementType, &value.Value,
			&value.Unit, &value.OriginalValue, &value.OriginalUnit, &value.Source,
			&value.DataQuality, &value.VerificationStatus, &value.ObservedAt,
			&value.ClientEventID, &value.CreatedAtDevice, &value.ReceivedAtServer,
		); err != nil {
			return err
		}
		target.Measurements = append(target.Measurements, value)
	}
	return rows.Err()
}

func loadObservations(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, coalesce(component_id::text, ''), coalesce(property, ''),
		       coalesce(status, ''), narrative, source, verification_status,
		       observed_at, coalesce(client_event_id::text, '')
		FROM observations
		WHERE execution_id = $1::uuid
		ORDER BY observed_at, created_at
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Observations = []executionapp.Observation{}
	for rows.Next() {
		var value executionapp.Observation
		if err := rows.Scan(
			&value.ID, &value.ComponentID, &value.Property, &value.Status,
			&value.Narrative, &value.Source, &value.VerificationStatus,
			&value.ObservedAt, &value.ClientEventID,
		); err != nil {
			return err
		}
		target.Observations = append(target.Observations, value)
	}
	return rows.Err()
}

func loadActions(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, coalesce(step_id::text, ''), coalesce(component_id::text, ''),
		       action_type, narrative, coalesce(outcome, ''), performed_by::text,
		       performed_at
		FROM maintenance_actions
		WHERE execution_id = $1::uuid
		ORDER BY performed_at, created_at
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Actions = []executionapp.Action{}
	for rows.Next() {
		var value executionapp.Action
		if err := rows.Scan(
			&value.ID, &value.StepID, &value.ComponentID, &value.ActionType,
			&value.Narrative, &value.Outcome, &value.PerformedBy, &value.PerformedAt,
		); err != nil {
			return err
		}
		target.Actions = append(target.Actions, value)
	}
	return rows.Err()
}

func loadDecisions(ctx context.Context, q queryer, target *executionapp.Execution) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, coalesce(step_id::text, ''), coalesce(component_id::text, ''),
		       decision, rationale, alternatives, decided_by::text, decided_at,
		       coalesce(client_event_id::text, '')
		FROM execution_decisions
		WHERE execution_id = $1::uuid
		ORDER BY decided_at, created_at
	`, target.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	target.Decisions = []executionapp.Decision{}
	for rows.Next() {
		var value executionapp.Decision
		if err := rows.Scan(
			&value.ID, &value.StepID, &value.ComponentID, &value.Decision,
			&value.Rationale, &value.Alternatives, &value.DecidedBy,
			&value.DecidedAt, &value.ClientEventID,
		); err != nil {
			return err
		}
		target.Decisions = append(target.Decisions, value)
	}
	return rows.Err()
}

func validateExecutionComponent(
	ctx context.Context,
	tx pgx.Tx,
	organizationID, assetID, componentID string,
) error {
	if componentID == "" {
		return nil
	}
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM asset_components
			WHERE id = $1::uuid AND organization_id = $2::uuid AND asset_id = $3::uuid
		)
	`, componentID, organizationID, assetID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.Join(executionapp.ErrInvalid, fmt.Errorf("component does not belong to execution asset"))
	}
	return nil
}
