package postgres

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	executiondomain "github.com/ZekromNguyen/skawld-maintenance/internal/execution/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/jackc/pgx/v5"
)

func (s Store) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command executionapp.CreateExecution,
) (executionapp.Execution, bool, error) {
	return runCommand(ctx, s, principal, "execution.create.v1", key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Execution, error) {
			var organizationID, siteID, assetID, assetTag, assetClass, incidentState string
			err := tx.QueryRow(ctx, `
				SELECT i.organization_id::text, i.site_id::text, i.asset_id::text,
				       a.tag, a.asset_class, i.status
				FROM incidents i
				JOIN assets a ON a.id = i.asset_id
				WHERE i.id = $1::uuid
				FOR UPDATE OF i
			`, command.IncidentID).Scan(
				&organizationID, &siteID, &assetID, &assetTag, &assetClass, &incidentState,
			)
			if err != nil {
				return executionapp.Execution{}, mapStoreError(err)
			}
			if organizationID != principal.OrganizationID ||
				!principal.CanAccessSite(organizationID, siteID) {
				return executionapp.Execution{}, executionapp.ErrForbidden
			}
			if incidentState == "RESOLVED" {
				return executionapp.Execution{}, errors.Join(executionapp.ErrInvalid, errors.New("resolved incident cannot start an execution"))
			}
			if !strings.Contains(strings.ToUpper(assetClass), "PUMP") {
				return executionapp.Execution{}, errors.Join(executionapp.ErrInvalid, errors.New("Phase 1 execution template is validated for pump assets only"))
			}
			executionID := s.IDs.New()
			if command.AssignedTo == "" {
				command.AssignedTo = principal.ID
			}
			stepIDs := []string{s.IDs.New(), s.IDs.New(), s.IDs.New(), s.IDs.New()}
			steps := executiondomain.PumpInspectionSteps(stepIDs)
			aggregate, err := executiondomain.NewExecution(executiondomain.Execution{
				ID: executionID, OrganizationID: organizationID, SiteID: siteID,
				IncidentID: command.IncidentID, AssetID: assetID, Purpose: command.Purpose,
				AssignedTo: command.AssignedTo, Steps: steps,
			})
			if err != nil {
				return executionapp.Execution{}, errors.Join(executionapp.ErrInvalid, err)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO maintenance_executions (
					id, organization_id, site_id, incident_id, asset_id, purpose, state,
					assigned_to, version, created_by, created_at, updated_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7,
					nullif($8, '')::uuid, $9, $10::uuid, $11, $11
				)
			`, aggregate.ID, aggregate.OrganizationID, aggregate.SiteID,
				aggregate.IncidentID, aggregate.AssetID, aggregate.Purpose, aggregate.State,
				aggregate.AssignedTo, aggregate.Version, principal.ID, now)
			if err != nil {
				return executionapp.Execution{}, err
			}
			for _, step := range aggregate.Steps {
				_, err = tx.Exec(ctx, `
					INSERT INTO execution_steps (
						id, organization_id, execution_id, step_key, sequence, title,
						state, risk_level, required_prerequisite, version
					) VALUES (
						$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, nullif($9, ''), $10
					)
				`, step.ID, aggregate.OrganizationID, aggregate.ID, step.Key,
					step.Sequence, step.Title, step.State, step.RiskLevel,
					step.RequiredPrerequisite, step.Version)
				if err != nil {
					return executionapp.Execution{}, err
				}
			}
			if incidentState == "OPEN" {
				if _, err := tx.Exec(ctx, `
					UPDATE incidents
					SET status = 'IN_PROGRESS', version = version + 1, updated_at = $2
					WHERE id = $1::uuid
				`, command.IncidentID, now); err != nil {
					return executionapp.Execution{}, err
				}
			}
			response, err := loadExecution(ctx, tx, principal, executionID, false)
			if err != nil {
				return executionapp.Execution{}, err
			}
			response.AssetTag = assetTag
			if err := s.appendExecutionAuditAndEvent(
				ctx, tx, principal, response, "ExecutionCreated", "execution.created", now,
			); err != nil {
				return executionapp.Execution{}, err
			}
			return response, nil
		})
}

func (s Store) Start(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.StartExecution,
) (executionapp.Execution, bool, error) {
	return runCommand(ctx, s, principal, "execution.start.v1:"+executionID, key, command, http.StatusOK,
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Execution, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Execution{}, mapStoreError(err)
			}
			if current.Version != command.ExpectedVersion {
				return executionapp.Execution{}, executionapp.ErrVersionConflict
			}
			aggregate := toDomain(current)
			if err := aggregate.Start(now); err != nil {
				return executionapp.Execution{}, errors.Join(executionapp.ErrInvalid, err)
			}
			tag, err := tx.Exec(ctx, `
				UPDATE maintenance_executions
				SET state = $2, started_at = $3, version = $4, updated_at = $3
				WHERE id = $1::uuid AND version = $5
			`, executionID, aggregate.State, aggregate.StartedAt, aggregate.Version, command.ExpectedVersion)
			if err != nil {
				return executionapp.Execution{}, err
			}
			if tag.RowsAffected() != 1 {
				return executionapp.Execution{}, executionapp.ErrVersionConflict
			}
			response, err := loadExecution(ctx, tx, principal, executionID, false)
			if err != nil {
				return executionapp.Execution{}, err
			}
			if err := s.appendExecutionAuditAndEvent(
				ctx, tx, principal, response, "ExecutionStarted", "execution.started", now,
			); err != nil {
				return executionapp.Execution{}, err
			}
			if err := appendCommandSync(
				ctx, tx, response.OrganizationID, key, command.SyncMetadata,
				command, response, response.Version, now,
			); err != nil {
				return executionapp.Execution{}, err
			}
			return response, nil
		})
}

func (s Store) VerifyPrerequisite(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.VerifyPrerequisite,
) (executionapp.Prerequisite, bool, error) {
	return runCommand(ctx, s, principal, "execution.prerequisite.verify.v1:"+executionID,
		key, command, createdStatus(),
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Prerequisite, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Prerequisite{}, mapStoreError(err)
			}
			if current.State == string(executiondomain.ExecutionCompleted) {
				return executionapp.Prerequisite{}, errors.Join(executionapp.ErrInvalid, errors.New("completed execution cannot accept prerequisite evidence"))
			}
			switch command.Type {
			case "WORK_PERMIT", "ENERGY_ISOLATION", "GAS_TEST", "SUPERVISOR", "COMPETENCY":
			default:
				return executionapp.Prerequisite{}, errors.Join(executionapp.ErrInvalid, errors.New("unsupported prerequisite type"))
			}
			switch command.Status {
			case "VERIFIED", "REJECTED", "EXPIRED":
			default:
				return executionapp.Prerequisite{}, errors.Join(executionapp.ErrInvalid, errors.New("unsupported prerequisite status"))
			}
			if command.ValidUntil != nil && !command.ValidUntil.After(command.VerifiedAt) {
				return executionapp.Prerequisite{}, errors.Join(executionapp.ErrInvalid, errors.New("valid_until must be after verified_at"))
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO prerequisite_verifications (
					id, organization_id, execution_id, prerequisite_type, status,
					external_reference, verified_by, verified_at, valid_until, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7::uuid, $8, $9, $10
				)
			`, s.IDs.New(), current.OrganizationID, current.ID, command.Type,
				command.Status, command.ExternalReference, principal.ID,
				command.VerifiedAt.UTC(), command.ValidUntil, now)
			if err != nil {
				return executionapp.Prerequisite{}, err
			}
			response := executionapp.Prerequisite{
				Type: command.Type, Status: command.Status,
				ExternalReference: command.ExternalReference, VerifiedBy: principal.ID,
				VerifiedAt: command.VerifiedAt.UTC(), ValidUntil: command.ValidUntil,
			}
			if err := s.Audit.Append(ctx, tx, audit.Event{
				ID: s.IDs.New(), OrganizationID: current.OrganizationID, SiteID: current.SiteID,
				ActorID: principal.ID, Action: "execution.prerequisite.recorded",
				EntityKind: "execution", EntityID: current.ID, ExecutionID: current.ID,
				After: response, OccurredAt: now,
			}); err != nil {
				return executionapp.Prerequisite{}, err
			}
			if err := s.appendExecutionSemanticEvent(
				ctx, tx, principal, current, "execution.prerequisite.recorded",
				"maintenance_execution", current.ID, response, now,
			); err != nil {
				return executionapp.Prerequisite{}, err
			}
			return response, nil
		})
}

func (s Store) CompleteStep(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID, stepID string,
	command executionapp.CompleteStep,
) (executionapp.Step, bool, error) {
	return runCommand(ctx, s, principal, "execution.step.complete.v1:"+executionID+":"+stepID,
		key, command, http.StatusOK,
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Step, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Step{}, mapStoreError(err)
			}
			if current.Version != command.ExpectedExecutionVersion {
				return executionapp.Step{}, executionapp.ErrVersionConflict
			}
			aggregate := toDomain(current)
			err = aggregate.CompleteStep(stepID, command.ExpectedStepVersion, now)
			if err != nil {
				var target *executiondomain.Step
				for index := range aggregate.Steps {
					if aggregate.Steps[index].ID == stepID {
						target = &aggregate.Steps[index]
						break
					}
				}
				if target == nil {
					return executionapp.Step{}, executionapp.ErrNotFound
				}
				if target.State != executiondomain.StepBlocked {
					if strings.Contains(err.Error(), "version conflict") {
						return executionapp.Step{}, executionapp.ErrVersionConflict
					}
					return executionapp.Step{}, errors.Join(executionapp.ErrInvalid, err)
				}
				_, updateErr := tx.Exec(ctx, `
					UPDATE execution_steps
					SET state = 'BLOCKED', blocked_reason = $3
					WHERE id = $1::uuid AND execution_id = $2::uuid
				`, target.ID, executionID, target.BlockedReason)
				if updateErr != nil {
					return executionapp.Step{}, updateErr
				}
				response := mapDomainStep(*target)
				if auditErr := s.Audit.Append(ctx, tx, audit.Event{
					ID: s.IDs.New(), OrganizationID: current.OrganizationID, SiteID: current.SiteID,
					ActorID: principal.ID, Action: "execution.step.blocked",
					EntityKind: "execution_step", EntityID: target.ID, ExecutionID: executionID,
					After: response, OccurredAt: now,
				}); auditErr != nil {
					return executionapp.Step{}, auditErr
				}
				if eventErr := s.appendExecutionSemanticEvent(
					ctx, tx, principal, current, "execution.step.blocked",
					"execution_step", target.ID, response, now,
				); eventErr != nil {
					return executionapp.Step{}, eventErr
				}
				if syncErr := appendCommandSync(
					ctx, tx, current.OrganizationID, key, command.SyncMetadata,
					command, response, current.Version, now,
				); syncErr != nil {
					return executionapp.Step{}, syncErr
				}
				return response, nil
			}
			var completed executiondomain.Step
			for _, step := range aggregate.Steps {
				if step.ID == stepID {
					completed = step
					break
				}
			}
			tag, err := tx.Exec(ctx, `
				UPDATE execution_steps
				SET state = $3, blocked_reason = NULL, completed_at = $4, version = $5
				WHERE id = $1::uuid AND execution_id = $2::uuid AND version = $6
			`, stepID, executionID, completed.State, now, completed.Version,
				command.ExpectedStepVersion)
			if err != nil {
				return executionapp.Step{}, err
			}
			if tag.RowsAffected() != 1 {
				return executionapp.Step{}, executionapp.ErrVersionConflict
			}
			tag, err = tx.Exec(ctx, `
				UPDATE maintenance_executions
				SET version = $2, updated_at = $3
				WHERE id = $1::uuid AND version = $4
			`, executionID, aggregate.Version, now, command.ExpectedExecutionVersion)
			if err != nil {
				return executionapp.Step{}, err
			}
			if tag.RowsAffected() != 1 {
				return executionapp.Step{}, executionapp.ErrVersionConflict
			}
			response := mapDomainStep(completed)
			if err := s.Audit.Append(ctx, tx, audit.Event{
				ID: s.IDs.New(), OrganizationID: current.OrganizationID, SiteID: current.SiteID,
				ActorID: principal.ID, Action: "execution.step.completed",
				EntityKind: "execution_step", EntityID: stepID, ExecutionID: executionID,
				After: response, OccurredAt: now,
			}); err != nil {
				return executionapp.Step{}, err
			}
			if err := s.appendExecutionSemanticEvent(
				ctx, tx, principal, current, "execution.step.completed",
				"execution_step", stepID, response, now,
			); err != nil {
				return executionapp.Step{}, err
			}
			if err := appendCommandSync(
				ctx, tx, current.OrganizationID, key, command.SyncMetadata,
				command, response, aggregate.Version, now,
			); err != nil {
				return executionapp.Step{}, err
			}
			return response, nil
		})
}

func appendCommandSync(
	ctx context.Context,
	tx pgx.Tx,
	organizationID, key string,
	metadata executionapp.SyncMetadata,
	payload, result any,
	serverVersion int64,
	now time.Time,
) error {
	var baseVersion *int64
	if metadata.BaseServerVersion > 0 {
		value := metadata.BaseServerVersion
		baseVersion = &value
	}
	return appendSyncInbox(
		ctx, tx, organizationID, metadata.DeviceID, metadata.ClientEventID,
		key, payload, result, baseVersion, serverVersion,
		metadata.CreatedAtDevice, now,
	)
}

func (s Store) Complete(
	ctx context.Context,
	principal identitydomain.Principal,
	key, executionID string,
	command executionapp.CompleteExecution,
) (executionapp.Execution, bool, error) {
	return runCommand(ctx, s, principal, "execution.complete.v1:"+executionID,
		key, command, http.StatusOK,
		func(ctx context.Context, tx pgx.Tx, now time.Time) (executionapp.Execution, error) {
			current, err := loadExecution(ctx, tx, principal, executionID, true)
			if err != nil {
				return executionapp.Execution{}, mapStoreError(err)
			}
			if current.Version != command.ExpectedVersion {
				return executionapp.Execution{}, executionapp.ErrVersionConflict
			}
			aggregate := toDomain(current)
			if err := aggregate.Finish(command.OutcomeSummary, now); err != nil {
				return executionapp.Execution{}, errors.Join(executionapp.ErrInvalid, err)
			}
			tag, err := tx.Exec(ctx, `
				UPDATE maintenance_executions
				SET state = $2, completed_at = $3, outcome_summary = $4,
				    version = $5, updated_at = $3
				WHERE id = $1::uuid AND version = $6
			`, executionID, aggregate.State, now, aggregate.OutcomeSummary,
				aggregate.Version, command.ExpectedVersion)
			if err != nil {
				return executionapp.Execution{}, err
			}
			if tag.RowsAffected() != 1 {
				return executionapp.Execution{}, executionapp.ErrVersionConflict
			}
			response, err := loadExecution(ctx, tx, principal, executionID, false)
			if err != nil {
				return executionapp.Execution{}, err
			}
			if err := s.appendExecutionAuditAndEvent(
				ctx, tx, principal, response, "ExecutionCompleted", "execution.completed", now,
			); err != nil {
				return executionapp.Execution{}, err
			}
			return response, nil
		})
}

func toDomain(value executionapp.Execution) executiondomain.Execution {
	result := executiondomain.Execution{
		ID: value.ID, OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		IncidentID: value.IncidentID, AssetID: value.AssetID, Purpose: value.Purpose,
		State: executiondomain.ExecutionState(value.State), AssignedTo: value.AssignedTo,
		Version: value.Version, StartedAt: value.StartedAt, CompletedAt: value.CompletedAt,
		OutcomeSummary: value.OutcomeSummary,
	}
	for _, step := range value.Steps {
		result.Steps = append(result.Steps, executiondomain.Step{
			ID: step.ID, Key: step.Key, Sequence: step.Sequence, Title: step.Title,
			State: executiondomain.StepState(step.State), RiskLevel: step.RiskLevel,
			RequiredPrerequisite: step.RequiredPrerequisite,
			BlockedReason:        step.BlockedReason, Version: step.Version,
		})
	}
	for _, prerequisite := range value.Prerequisites {
		result.Verifications = append(result.Verifications, executiondomain.Verification{
			Type: prerequisite.Type, Status: prerequisite.Status,
			ExternalReference: prerequisite.ExternalReference,
			VerifiedBy:        prerequisite.VerifiedBy, VerifiedAt: prerequisite.VerifiedAt,
			ValidUntil: prerequisite.ValidUntil,
		})
	}
	return result
}

func mapDomainStep(value executiondomain.Step) executionapp.Step {
	return executionapp.Step{
		ID: value.ID, Key: value.Key, Sequence: value.Sequence, Title: value.Title,
		State: string(value.State), RiskLevel: value.RiskLevel,
		RequiredPrerequisite: value.RequiredPrerequisite,
		BlockedReason:        value.BlockedReason, Version: value.Version,
	}
}

func (s Store) appendExecutionAuditAndEvent(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	value executionapp.Execution,
	eventType, action string,
	now time.Time,
) error {
	if err := events.Append(ctx, tx, events.Event{
		ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		ActorID: principal.ID, Type: eventType,
		AggregateType: "maintenance_execution", AggregateID: value.ID,
		AggregateVersion: value.Version, SubjectKind: "EXECUTION",
		SubjectID: value.ID, Payload: value, OccurredAt: now,
	}); err != nil {
		return err
	}
	return s.Audit.Append(ctx, tx, audit.Event{
		ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
		ActorID: principal.ID, Action: action, EntityKind: "maintenance_execution",
		EntityID: value.ID, ExecutionID: value.ID, After: value, OccurredAt: now,
	})
}

func (s Store) appendExecutionSemanticEvent(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	execution executionapp.Execution,
	eventType, aggregateType, aggregateID string,
	payload any,
	now time.Time,
) error {
	return events.Append(ctx, tx, events.Event{
		ID: s.IDs.New(), OrganizationID: execution.OrganizationID,
		SiteID: execution.SiteID, ActorID: principal.ID, Type: eventType,
		AggregateType: aggregateType, AggregateID: aggregateID,
		AggregateVersion: execution.Version, SubjectKind: "EXECUTION",
		SubjectID: execution.ID,
		Payload: map[string]any{
			"execution_id": execution.ID,
			"asset_id":     execution.AssetID,
			"value":        payload,
		},
		OccurredAt: now,
	})
}
