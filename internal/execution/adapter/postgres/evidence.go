package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	executiondomain "github.com/ZekromNguyen/skawld-maintenance/internal/execution/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/jackc/pgx/v5"
)

func (s Store) RecordMeasurement(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.RecordMeasurement,
) (executionapp.Measurement, bool, error) {
	return runCommand(ctx, s, principal, "execution.measurement.create.v1:"+executionID,
		key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Measurement, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Measurement{}, mapStoreError(err)
			}
			if current.State != string(executiondomain.ExecutionInProgress) {
				return executionapp.Measurement{}, errors.Join(executionapp.ErrInvalid, errors.New("execution must be in progress"))
			}
			if err := validateExecutionComponent(
				ctx, tx, current.OrganizationID, current.AssetID, command.ComponentID,
			); err != nil {
				return executionapp.Measurement{}, err
			}
			measurement, err := executiondomain.NewMeasurement(executiondomain.Measurement{
				ID: s.IDs.New(), OrganizationID: current.OrganizationID, SiteID: current.SiteID,
				ExecutionID: current.ID, AssetID: current.AssetID,
				ComponentID:   command.ComponentID,
				Type:          executiondomain.MeasurementType(command.MeasurementType),
				OriginalValue: command.Value, OriginalUnit: executiondomain.Unit(command.Unit),
				Source: command.Source, DataQuality: command.DataQuality,
				InstrumentReference: command.InstrumentReference,
				VerificationStatus:  command.VerificationStatus, ObservedAt: command.ObservedAt,
				RecordedBy: principal.ID, ClientEventID: command.ClientEventID,
				DeviceID: command.DeviceID, CreatedAtDevice: command.CreatedAtDevice,
				ReceivedAtServer: now,
			})
			if err != nil {
				return executionapp.Measurement{}, errors.Join(executionapp.ErrInvalid, err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO measurements (
					id, organization_id, site_id, execution_id, asset_id, component_id,
					measurement_type, value, unit, original_value, original_unit,
					source, data_quality, instrument_reference, verification_status,
					observed_at, recorded_by, client_event_id, device_id,
					created_at_device, received_at_server, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, nullif($6, '')::uuid,
					$7, $8::numeric, $9, $10, $11, $12, $13, nullif($14, ''), $15,
					$16, $17::uuid, $18::uuid, nullif($19, ''), $20, $21, $21
				)
			`, measurement.ID, measurement.OrganizationID, measurement.SiteID,
				measurement.ExecutionID, measurement.AssetID, measurement.ComponentID,
				measurement.Type, measurement.Value, measurement.Unit,
				measurement.OriginalValue, measurement.OriginalUnit, measurement.Source,
				measurement.DataQuality, measurement.InstrumentReference,
				measurement.VerificationStatus, measurement.ObservedAt,
				measurement.RecordedBy, measurement.ClientEventID, measurement.DeviceID,
				measurement.CreatedAtDevice, measurement.ReceivedAtServer)
			if err != nil {
				return executionapp.Measurement{}, err
			}
			response := mapMeasurement(measurement)
			if err := appendSyncInbox(
				ctx, tx, current.OrganizationID, command.DeviceID,
				command.ClientEventID, key, command, response, nil,
				current.Version, command.CreatedAtDevice, now,
			); err != nil {
				return executionapp.Measurement{}, err
			}
			if err := s.appendEvidenceAudit(
				ctx, tx, principal, current, "measurement.recorded",
				"measurement", response.ID, response, now,
			); err != nil {
				return executionapp.Measurement{}, err
			}
			return response, nil
		})
}

func (s Store) RecordObservation(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.RecordObservation,
) (executionapp.Observation, bool, error) {
	return runCommand(ctx, s, principal, "execution.observation.create.v1:"+executionID,
		key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Observation, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Observation{}, mapStoreError(err)
			}
			if current.State != string(executiondomain.ExecutionInProgress) {
				return executionapp.Observation{}, errors.Join(executionapp.ErrInvalid, errors.New("execution must be in progress"))
			}
			if err := validateExecutionComponent(
				ctx, tx, current.OrganizationID, current.AssetID, command.ComponentID,
			); err != nil {
				return executionapp.Observation{}, err
			}
			switch command.Source {
			case "TECHNICIAN", "VOICE_TRANSCRIPT", "VISION_CANDIDATE", "EXTERNAL_SYSTEM":
			default:
				return executionapp.Observation{}, errors.Join(executionapp.ErrInvalid, errors.New("unsupported observation source"))
			}
			switch command.VerificationStatus {
			case "UNVERIFIED", "VERIFIED", "REJECTED":
			default:
				return executionapp.Observation{}, errors.Join(executionapp.ErrInvalid, errors.New("unsupported verification status"))
			}
			if command.Source == "VISION_CANDIDATE" && command.VerificationStatus == "VERIFIED" {
				return executionapp.Observation{}, errors.Join(executionapp.ErrInvalid, errors.New("vision output must be stored as an unverified observation candidate"))
			}
			if command.ObservedAt.IsZero() {
				return executionapp.Observation{}, errors.Join(executionapp.ErrInvalid, errors.New("observed_at is required"))
			}
			response := executionapp.Observation{
				ID: s.IDs.New(), ComponentID: command.ComponentID,
				Property: strings.TrimSpace(command.Property), Status: strings.TrimSpace(command.Status),
				Narrative: strings.TrimSpace(command.Narrative), Source: command.Source,
				VerificationStatus: command.VerificationStatus,
				ObservedAt:         command.ObservedAt.UTC(), ClientEventID: command.ClientEventID,
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO observations (
					id, organization_id, site_id, execution_id, asset_id, component_id,
					property, status, narrative, source, verification_status, observed_at,
					recorded_by, client_event_id, device_id, created_at_device,
					received_at_server, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, nullif($6, '')::uuid,
					nullif($7, ''), nullif($8, ''), $9, $10, $11, $12, $13::uuid,
					$14::uuid, nullif($15, ''), $16, $17, $17
				)
			`, response.ID, current.OrganizationID, current.SiteID, current.ID,
				current.AssetID, response.ComponentID, response.Property, response.Status,
				response.Narrative, response.Source, response.VerificationStatus,
				response.ObservedAt, principal.ID, response.ClientEventID, command.DeviceID,
				command.CreatedAtDevice, now)
			if err != nil {
				return executionapp.Observation{}, err
			}
			if err := appendSyncInbox(
				ctx, tx, current.OrganizationID, command.DeviceID,
				command.ClientEventID, key, command, response, nil,
				current.Version, command.CreatedAtDevice, now,
			); err != nil {
				return executionapp.Observation{}, err
			}
			if err := s.appendEvidenceAudit(
				ctx, tx, principal, current, "observation.recorded",
				"observation", response.ID, response, now,
			); err != nil {
				return executionapp.Observation{}, err
			}
			return response, nil
		})
}

func (s Store) RecordAction(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.RecordAction,
) (executionapp.Action, bool, error) {
	return runCommand(ctx, s, principal, "execution.action.create.v1:"+executionID,
		key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Action, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Action{}, mapStoreError(err)
			}
			if current.State != string(executiondomain.ExecutionInProgress) {
				return executionapp.Action{}, errors.Join(executionapp.ErrInvalid, errors.New("execution must be in progress"))
			}
			if err := validateExecutionComponent(
				ctx, tx, current.OrganizationID, current.AssetID, command.ComponentID,
			); err != nil {
				return executionapp.Action{}, err
			}
			switch command.ActionType {
			case "INSPECTED", "CLEANED", "LUBRICATED", "ADJUSTED", "REPLACED", "OTHER":
			default:
				return executionapp.Action{}, errors.Join(executionapp.ErrInvalid, errors.New("unsupported maintenance action"))
			}
			if command.StepID != "" {
				var exists bool
				if err := tx.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM execution_steps
						WHERE id = $1::uuid AND execution_id = $2::uuid
					)
				`, command.StepID, executionID).Scan(&exists); err != nil {
					return executionapp.Action{}, err
				}
				if !exists {
					return executionapp.Action{}, errors.Join(executionapp.ErrInvalid, errors.New("step does not belong to execution"))
				}
			}
			response := executionapp.Action{
				ID: s.IDs.New(), StepID: command.StepID, ComponentID: command.ComponentID,
				ActionType: command.ActionType, Narrative: strings.TrimSpace(command.Narrative),
				Outcome: strings.TrimSpace(command.Outcome), PerformedBy: principal.ID,
				PerformedAt: command.PerformedAt.UTC(),
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO maintenance_actions (
					id, organization_id, execution_id, step_id, component_id,
					action_type, narrative, outcome, performed_by, performed_at, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, nullif($4, '')::uuid,
					nullif($5, '')::uuid, $6, $7, nullif($8, ''),
					$9::uuid, $10, $11
				)
			`, response.ID, current.OrganizationID, current.ID, response.StepID,
				response.ComponentID, response.ActionType, response.Narrative,
				response.Outcome, response.PerformedBy, response.PerformedAt, now)
			if err != nil {
				return executionapp.Action{}, err
			}
			if err := s.appendEvidenceAudit(
				ctx, tx, principal, current, "maintenance.action.recorded",
				"maintenance_action", response.ID, response, now,
			); err != nil {
				return executionapp.Action{}, err
			}
			return response, nil
		})
}

func (s Store) RecordDecision(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.RecordDecision,
) (executionapp.Decision, bool, error) {
	return runCommand(ctx, s, principal, "execution.decision.create.v1:"+executionID,
		key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Decision, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Decision{}, mapStoreError(err)
			}
			if current.State != string(executiondomain.ExecutionInProgress) {
				return executionapp.Decision{}, errors.Join(executionapp.ErrInvalid, errors.New("execution must be in progress"))
			}
			if err := validateExecutionComponent(
				ctx, tx, current.OrganizationID, current.AssetID, command.ComponentID,
			); err != nil {
				return executionapp.Decision{}, err
			}
			if command.StepID != "" {
				var exists bool
				if err := tx.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM execution_steps
						WHERE id = $1::uuid AND execution_id = $2::uuid
					)
				`, command.StepID, executionID).Scan(&exists); err != nil {
					return executionapp.Decision{}, err
				}
				if !exists {
					return executionapp.Decision{}, errors.Join(executionapp.ErrInvalid, errors.New("step does not belong to execution"))
				}
			}
			alternatives, err := json.Marshal(command.Alternatives)
			if err != nil {
				return executionapp.Decision{}, err
			}
			response := executionapp.Decision{
				ID: s.IDs.New(), StepID: command.StepID, ComponentID: command.ComponentID,
				Decision:     strings.TrimSpace(command.Decision),
				Rationale:    strings.TrimSpace(command.Rationale),
				Alternatives: command.Alternatives, DecidedBy: principal.ID,
				DecidedAt: command.DecidedAt.UTC(), ClientEventID: command.ClientEventID,
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO execution_decisions (
					id, organization_id, execution_id, step_id, component_id,
					decision, rationale, alternatives, decided_by, decided_at,
					client_event_id, device_id, created_at_device,
					received_at_server, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, nullif($4, '')::uuid,
					nullif($5, '')::uuid, $6, $7, $8::jsonb, $9::uuid, $10,
					$11::uuid, nullif($12, ''), $13, $14, $14
				)
			`, response.ID, current.OrganizationID, current.ID, response.StepID,
				response.ComponentID, response.Decision, response.Rationale,
				alternatives, response.DecidedBy, response.DecidedAt,
				response.ClientEventID, command.DeviceID, command.CreatedAtDevice, now)
			if err != nil {
				return executionapp.Decision{}, err
			}
			if err := appendSyncInbox(
				ctx, tx, current.OrganizationID, command.DeviceID,
				command.ClientEventID, key, command, response, nil,
				current.Version, command.CreatedAtDevice, now,
			); err != nil {
				return executionapp.Decision{}, err
			}
			if err := s.appendEvidenceAudit(
				ctx, tx, principal, current, "decision.recorded",
				"execution_decision", response.ID, response, now,
			); err != nil {
				return executionapp.Decision{}, err
			}
			return response, nil
		})
}

func mapMeasurement(value executiondomain.Measurement) executionapp.Measurement {
	return executionapp.Measurement{
		ID: value.ID, ComponentID: value.ComponentID,
		MeasurementType: string(value.Type), Value: value.Value, Unit: string(value.Unit),
		OriginalValue: value.OriginalValue, OriginalUnit: string(value.OriginalUnit),
		Source: value.Source, DataQuality: value.DataQuality,
		VerificationStatus: value.VerificationStatus, ObservedAt: value.ObservedAt,
		ClientEventID: value.ClientEventID, CreatedAtDevice: value.CreatedAtDevice,
		ReceivedAtServer: value.ReceivedAtServer,
	}
}

func (s Store) appendEvidenceAudit(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	execution executionapp.Execution,
	action, entityKind, entityID string,
	after any,
	now time.Time,
) error {
	if err := s.Audit.Append(ctx, tx, audit.Event{
		ID: s.IDs.New(), OrganizationID: execution.OrganizationID, SiteID: execution.SiteID,
		ActorID: principal.ID, Action: action, EntityKind: entityKind,
		EntityID: entityID, ExecutionID: execution.ID, After: after, OccurredAt: now,
	}); err != nil {
		return err
	}
	return events.Append(ctx, tx, events.Event{
		ID: s.IDs.New(), OrganizationID: execution.OrganizationID,
		SiteID: execution.SiteID, ActorID: principal.ID, Type: action,
		AggregateType: entityKind, AggregateID: entityID,
		AggregateVersion: execution.Version, SubjectKind: "EXECUTION",
		SubjectID: execution.ID,
		Payload: map[string]any{
			"execution_id": execution.ID,
			"asset_id":     execution.AssetID,
			"value":        after,
		},
		OccurredAt: now,
	})
}
