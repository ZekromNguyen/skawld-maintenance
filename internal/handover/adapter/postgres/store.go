package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Audit       audit.Sink
	Idempotency idempotency.Store
}

type result struct {
	Value  handoverapp.Handover
	Replay bool
}

func (s Store) LoadWindowContext(
	ctx context.Context,
	principal identitydomain.Principal,
	command handoverapp.PrepareDraft,
) (handoverapp.WindowContext, error) {
	var snapshot []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT jsonb_build_object(
		  'summary', concat('Shift handover for ', s.name, ' from ', $3::timestamptz, ' to ', $4::timestamptz),
		  'open_incidents', coalesce((
		    SELECT jsonb_agg(jsonb_build_object(
		      'id', i.id, 'number', i.number, 'asset_tag', a.tag,
		      'summary', i.summary, 'severity', i.severity, 'state', i.state
		    ) ORDER BY i.severity DESC, i.detected_at)
		    FROM incidents i JOIN assets a ON a.id = i.asset_id
		    WHERE i.organization_id = $1::uuid AND i.site_id = $2::uuid
		      AND i.state IN ('OPEN', 'IN_PROGRESS')
		  ), '[]'::jsonb),
		  'active_executions', coalesce((
		    SELECT jsonb_agg(jsonb_build_object(
		      'id', e.id, 'asset_tag', a.tag, 'purpose', e.purpose,
		      'state', e.state, 'outcome', coalesce(e.outcome_summary, '')
		    ) ORDER BY e.created_at)
		    FROM maintenance_executions e JOIN assets a ON a.id = e.asset_id
		    WHERE e.organization_id = $1::uuid AND e.site_id = $2::uuid
		      AND e.state IN ('ASSIGNED', 'IN_PROGRESS')
		  ), '[]'::jsonb),
		  'safety_concerns', coalesce((
		    SELECT jsonb_agg(jsonb_build_object(
		      'execution_id', p.execution_id, 'type', p.prerequisite_type,
		      'status', p.status, 'external_reference', coalesce(p.external_reference, '')
		    ) ORDER BY p.created_at)
		    FROM prerequisite_verifications p
		    JOIN maintenance_executions e ON e.id = p.execution_id
		    WHERE p.organization_id = $1::uuid AND e.site_id = $2::uuid
		      AND p.status <> 'VERIFIED'
		  ), '[]'::jsonb),
		  'follow_up', coalesce((
		    SELECT jsonb_agg(jsonb_build_object(
		      'execution_id', st.execution_id, 'step', st.title,
		      'state', st.state, 'blocked_reason', coalesce(st.blocked_reason, '')
		    ) ORDER BY st.sequence)
		    FROM execution_steps st
		    JOIN maintenance_executions e ON e.id = st.execution_id
		    WHERE st.organization_id = $1::uuid AND e.site_id = $2::uuid
		      AND st.state IN ('PENDING', 'IN_PROGRESS', 'BLOCKED')
		  ), '[]'::jsonb)
		)
		FROM sites s
		WHERE s.id = $2::uuid AND s.organization_id = $1::uuid
		  AND (COALESCE(cardinality($5::uuid[]), 0) = 0 OR s.id = ANY($5::uuid[]))
	`, principal.OrganizationID, command.SiteID, command.ShiftStart.UTC(),
		command.ShiftEnd.UTC(), principal.SiteIDs).Scan(&snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return handoverapp.WindowContext{}, handoverapp.ErrNotFound
	}
	if err != nil {
		return handoverapp.WindowContext{}, err
	}
	evidence, err := s.loadEvidence(ctx, principal, command.SiteID)
	if err != nil {
		return handoverapp.WindowContext{}, err
	}
	return handoverapp.WindowContext{
		OrganizationID: principal.OrganizationID, SiteID: command.SiteID,
		Snapshot: snapshot, Evidence: evidence,
	}, nil
}

func (s Store) loadEvidence(
	ctx context.Context,
	principal identitydomain.Principal,
	siteID string,
) ([]knowledgedomain.Evidence, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT kind, source_id::text, locator, content
		FROM (
		  SELECT 'INCIDENT' AS kind, i.id AS source_id, i.number AS locator,
		         concat(a.tag, ': ', i.summary, ' [', i.severity, '/', i.state, ']') AS content,
		         i.detected_at AS occurred_at
		  FROM incidents i JOIN assets a ON a.id = i.asset_id
		  WHERE i.organization_id = $1::uuid AND i.site_id = $2::uuid
		    AND i.state IN ('OPEN', 'IN_PROGRESS')
		  UNION ALL
		  SELECT 'EXECUTION', e.id, a.tag,
		         concat(e.purpose, ' [', e.state, ']'), e.created_at
		  FROM maintenance_executions e JOIN assets a ON a.id = e.asset_id
		  WHERE e.organization_id = $1::uuid AND e.site_id = $2::uuid
		    AND e.state IN ('ASSIGNED', 'IN_PROGRESS')
		  UNION ALL
		  SELECT 'PREREQUISITE', p.id, p.prerequisite_type,
		         concat(p.status, ' ', coalesce(p.external_reference, '')), p.created_at
		  FROM prerequisite_verifications p
		  JOIN maintenance_executions e ON e.id = p.execution_id
		  WHERE p.organization_id = $1::uuid AND e.site_id = $2::uuid
		    AND p.status <> 'VERIFIED'
		) facts
		ORDER BY occurred_at DESC
		LIMIT 200
	`, principal.OrganizationID, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []knowledgedomain.Evidence
	for rows.Next() {
		var value knowledgedomain.Evidence
		if err := rows.Scan(
			&value.Kind, &value.SourceID, &value.Locator, &value.Content,
		); err != nil {
			return nil, err
		}
		value.ID = strings.ToLower(value.Kind) + ":" + value.SourceID
		value.Title = value.Kind
		value.Authority = knowledgedomain.AuthoritySiteApproved
		value.ContentHash = skawld.HashBytes([]byte(value.Content))
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s Store) SaveDraft(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	value handoverapp.Handover,
) (handoverapp.Handover, bool, error) {
	hash, err := idempotency.HashRequest(map[string]any{
		"site_id": value.SiteID, "shift_start": value.ShiftStart,
		"shift_end": value.ShiftEnd, "input_sha256": value.InputSHA256,
	})
	if err != nil {
		return handoverapp.Handover{}, false, err
	}
	scope := "handover.prepare.v1:" + value.SiteID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay handoverapp.Handover
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		value.ID, value.Version, value.CreatedBy = s.IDs.New(), 1, principal.ID
		value.CreatedAt, value.UpdatedAt = now, now
		content, _ := json.Marshal(value.Content)
		evidence, _ := json.Marshal(value.Evidence)
		_, err = tx.Exec(ctx, `
			INSERT INTO shift_handovers (
				id, organization_id, site_id, shift_start, shift_end, state,
				structured_content, evidence_snapshot, provider, model,
				model_version, prompt_version, input_sha256, output_sha256,
				version, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5, 'DRAFT',
				$6::jsonb, $7::jsonb, $8, $9, $10, $11, $12, $13,
				1, $14::uuid, $15, $15
			)
		`, value.ID, value.OrganizationID, value.SiteID, value.ShiftStart,
			value.ShiftEnd, content, evidence, value.Provider, value.Model,
			value.ModelVersion, value.PromptVersion, value.InputSHA256,
			value.OutputSHA256, principal.ID, now)
		if err != nil {
			return result{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: "handover.draft.generated",
			EntityKind: "shift_handover", EntityID: value.ID,
			AIInvolvement: map[string]any{
				"provider": value.Provider, "model": value.Model,
				"model_version": value.ModelVersion, "prompt_version": value.PromptVersion,
			},
			After: value, OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		if err := events.Append(ctx, tx, events.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID,
			SiteID: value.SiteID, ActorID: principal.ID,
			Type: "handover.draft.generated", AggregateType: "shift_handover",
			AggregateID: value.ID, AggregateVersion: value.Version,
			SubjectKind: "HANDOVER", SubjectID: value.ID, Payload: value,
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusCreated, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: value}, nil
	})
	if err != nil {
		return handoverapp.Handover{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	handoverID string,
) (handoverapp.Handover, error) {
	return get(ctx, s.Pool, principal, handoverID, false)
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter handoverapp.HandoverFilter,
) ([]handoverapp.Handover, bool, error) {
	query := `
		SELECT id::text, organization_id::text, site_id::text,
		       shift_start, shift_end, state, structured_content,
		       evidence_snapshot, provider, model, model_version,
		       prompt_version, input_sha256, output_sha256, version,
		       created_by::text, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(accepted_by::text, ''), accepted_at,
		       coalesce(acknowledged_by::text, ''), acknowledged_at,
		       created_at, updated_at
		FROM shift_handovers
		WHERE organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR site_id = ANY($2::uuid[]))`
	args := []any{principal.OrganizationID, principal.SiteIDs}
	if filter.SiteID != "" {
		args = append(args, filter.SiteID)
		query += fmt.Sprintf(" AND site_id = $%d::uuid", len(args))
	}
	if len(filter.States) > 0 {
		args = append(args, filter.States)
		query += fmt.Sprintf(" AND state = ANY($%d::text[])", len(args))
	}
	if filter.Cursor != "" {
		cut := strings.LastIndex(filter.Cursor, "|")
		if cut < 0 {
			return nil, false, handoverapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (shift_start, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY shift_start DESC, id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]handoverapp.Handover, 0, filter.PageSize+1)
	for rows.Next() {
		var value handoverapp.Handover
		var content, evidence []byte
		if err := rows.Scan(
			&value.ID, &value.OrganizationID, &value.SiteID,
			&value.ShiftStart, &value.ShiftEnd, &value.State, &content, &evidence,
			&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
			&value.InputSHA256, &value.OutputSHA256, &value.Version, &value.CreatedBy,
			&value.SubmittedBy, &value.SubmittedAt, &value.AcceptedBy, &value.AcceptedAt,
			&value.AcknowledgedBy, &value.AcknowledgedAt, &value.CreatedAt, &value.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(content, &value.Content); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
			return nil, false, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
	}
	return items, hasMore, nil
}

func get(
	ctx context.Context,
	query rowQuerier,
	principal identitydomain.Principal,
	handoverID string,
	forUpdate bool,
) (handoverapp.Handover, error) {
	statement := `
		SELECT id::text, organization_id::text, site_id::text,
		       shift_start, shift_end, state, structured_content,
		       evidence_snapshot, provider, model, model_version,
		       prompt_version, input_sha256, output_sha256, version,
		       created_by::text, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(accepted_by::text, ''), accepted_at,
		       coalesce(acknowledged_by::text, ''), acknowledged_at,
		       created_at, updated_at
		FROM shift_handovers
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))`
	if forUpdate {
		statement += " FOR UPDATE"
	}
	var value handoverapp.Handover
	var content, evidence []byte
	err := query.QueryRow(
		ctx, statement, handoverID, principal.OrganizationID, principal.SiteIDs,
	).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID,
		&value.ShiftStart, &value.ShiftEnd, &value.State, &content, &evidence,
		&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
		&value.InputSHA256, &value.OutputSHA256, &value.Version, &value.CreatedBy,
		&value.SubmittedBy, &value.SubmittedAt, &value.AcceptedBy, &value.AcceptedAt,
		&value.AcknowledgedBy, &value.AcknowledgedAt, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return handoverapp.Handover{}, handoverapp.ErrNotFound
	}
	if err != nil {
		return handoverapp.Handover{}, err
	}
	if err := json.Unmarshal(content, &value.Content); err != nil {
		return handoverapp.Handover{}, err
	}
	if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
		return handoverapp.Handover{}, err
	}
	return value, nil
}

func (s Store) Edit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID string,
	command handoverapp.Edit,
) (handoverapp.Handover, bool, error) {
	return s.change(ctx, principal, key, handoverID, "edit", command,
		func(ctx context.Context, tx pgx.Tx, value *handoverapp.Handover) error {
			if value.State != "DRAFT" {
				return handoverapp.ErrConflict
			}
			content, _ := json.Marshal(command.Content)
			now := s.Clock.Now()
			tag, err := tx.Exec(ctx, `
				UPDATE shift_handovers
				SET structured_content = $2::jsonb, version = version + 1, updated_at = $3
				WHERE id = $1::uuid AND state = 'DRAFT' AND version = $4
			`, value.ID, content, now, command.ExpectedVersion)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return handoverapp.ErrConflict
			}
			value.Content, value.Version, value.UpdatedAt = command.Content, value.Version+1, now
			return nil
		})
}

func (s Store) Submit(ctx context.Context, principal identitydomain.Principal, key, id string, command handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return s.transition(ctx, principal, key, id, "submit", "DRAFT", "SUBMITTED", command)
}

func (s Store) Accept(ctx context.Context, principal identitydomain.Principal, key, id string, command handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return s.transition(ctx, principal, key, id, "accept", "SUBMITTED", "ACCEPTED", command)
}

func (s Store) Acknowledge(ctx context.Context, principal identitydomain.Principal, key, id string, command handoverapp.Transition) (handoverapp.Handover, bool, error) {
	return s.transition(ctx, principal, key, id, "acknowledge", "ACCEPTED", "ACKNOWLEDGED", command)
}

func (s Store) transition(
	ctx context.Context,
	principal identitydomain.Principal,
	key, id, action, from, to string,
	command handoverapp.Transition,
) (handoverapp.Handover, bool, error) {
	return s.change(ctx, principal, key, id, action, command,
		func(ctx context.Context, tx pgx.Tx, value *handoverapp.Handover) error {
			if value.State != from {
				return handoverapp.ErrConflict
			}
			column := map[string]string{
				"submit": "submitted", "accept": "accepted",
				"acknowledge": "acknowledged",
			}[action]
			now := s.Clock.Now()
			statement := `
				UPDATE shift_handovers
				SET state = $2, ` + column + `_by = $3::uuid, ` + column + `_at = $4,
				    version = version + 1, updated_at = $4
				WHERE id = $1::uuid AND state = $5 AND version = $6`
			tag, err := tx.Exec(
				ctx, statement, value.ID, to, principal.ID, now,
				from, command.ExpectedVersion,
			)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return handoverapp.ErrConflict
			}
			value.State, value.Version, value.UpdatedAt = to, value.Version+1, now
			switch action {
			case "submit":
				value.SubmittedBy, value.SubmittedAt = principal.ID, &now
			case "accept":
				value.AcceptedBy, value.AcceptedAt = principal.ID, &now
			case "acknowledge":
				value.AcknowledgedBy, value.AcknowledgedAt = principal.ID, &now
			}
			return nil
		})
}

func (s Store) change(
	ctx context.Context,
	principal identitydomain.Principal,
	key, handoverID, action string,
	command any,
	mutate func(context.Context, pgx.Tx, *handoverapp.Handover) error,
) (handoverapp.Handover, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return handoverapp.Handover{}, false, err
	}
	scope := "handover." + action + ".v1:" + handoverID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay handoverapp.Handover
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		value, err := get(ctx, tx, principal, handoverID, true)
		if err != nil {
			return result{}, err
		}
		before := value
		if err := mutate(ctx, tx, &value); err != nil {
			return result{}, err
		}
		auditAction := map[string]string{
			"edit": "handover.edited", "submit": "handover.submitted",
			"accept": "handover.accepted", "acknowledge": "handover.acknowledged",
		}[action]
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: auditAction,
			EntityKind: "shift_handover", EntityID: value.ID,
			Before: before, After: value, OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		if err := events.Append(ctx, tx, events.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID,
			SiteID: value.SiteID, ActorID: principal.ID, Type: auditAction,
			AggregateType: "shift_handover", AggregateID: value.ID,
			AggregateVersion: value.Version, SubjectKind: "HANDOVER",
			SubjectID:  value.ID,
			Payload:    map[string]any{"before": before, "value": value},
			OccurredAt: now,
		}); err != nil {
			return result{}, err
		}
		body, _ := json.Marshal(value)
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusOK, body, now,
		); err != nil {
			return result{}, err
		}
		return result{Value: value}, nil
	})
	if err != nil {
		return handoverapp.Handover{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func (s Store) RecordCall(
	ctx context.Context,
	principal identitydomain.Principal,
	window handoverapp.WindowContext,
	generation skawld.Generation,
	outcome, errorCode string,
) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO ai_call_records (
			id, organization_id, site_id, capability, provider, model,
			model_version, prompt_version, input_sha256, output_sha256,
			outcome, latency_ms, tokens_in, tokens_out,
			estimated_cost_micros, error_code, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9,
			nullif($10, ''), $11, $12, nullif($13, 0), nullif($14, 0),
			nullif($15, 0), nullif($16, ''), $17
		)
	`, s.IDs.New(), window.OrganizationID, window.SiteID,
		skawld.CapabilityShiftHandover, generation.Metadata.Provider,
		generation.Metadata.Model, generation.Metadata.ModelVersion,
		generation.Prompt, generation.InputHash, generation.OutputHash,
		outcome, generation.Latency.Milliseconds(), generation.Metadata.TokensIn,
		generation.Metadata.TokensOut, generation.Metadata.EstimatedCostMicros,
		errorCode, s.Clock.Now())
	return err
}
