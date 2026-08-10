package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
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
	Value  reportapp.Report
	Replay bool
}

func (s Store) LoadExecutionContext(
	ctx context.Context,
	principal identitydomain.Principal,
	executionID string,
) (reportapp.ExecutionContext, error) {
	var value reportapp.ExecutionContext
	var snapshot []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT e.organization_id::text, e.site_id::text, e.id::text,
		       e.asset_id::text, e.purpose,
		       jsonb_build_object(
		         'summary', e.purpose,
		         'state', e.state,
		         'outcome', coalesce(e.outcome_summary, ''),
		         'measurements', coalesce((
		           SELECT jsonb_agg(jsonb_build_object(
		             'id', m.id, 'type', m.measurement_type,
		             'value', m.original_value, 'unit', m.original_unit,
		             'quality', m.data_quality, 'observed_at', m.observed_at
		           ) ORDER BY m.observed_at)
		           FROM measurements m WHERE m.execution_id = e.id
		         ), '[]'::jsonb),
		         'observations', coalesce((
		           SELECT jsonb_agg(jsonb_build_object(
		             'id', o.id, 'narrative', o.narrative,
		             'verification', o.verification_status, 'observed_at', o.observed_at
		           ) ORDER BY o.observed_at)
		           FROM observations o WHERE o.execution_id = e.id
		         ), '[]'::jsonb),
		         'actions', coalesce((
		           SELECT jsonb_agg(jsonb_build_object(
		             'id', a.id, 'type', a.action_type,
		             'narrative', a.narrative, 'outcome', a.outcome,
		             'performed_at', a.performed_at
		           ) ORDER BY a.performed_at)
		           FROM maintenance_actions a WHERE a.execution_id = e.id
		         ), '[]'::jsonb)
		       )
		FROM maintenance_executions e
		WHERE e.id = $1::uuid AND e.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR e.site_id = ANY($3::uuid[]))
		  AND (
		    e.assigned_to = $4::uuid OR
		    $5 = true
		  )
	`, executionID, principal.OrganizationID, principal.SiteIDs, principal.ID,
		principal.Has(identitydomain.PermissionExecutionReadAll)).Scan(
		&value.OrganizationID, &value.SiteID, &value.ExecutionID,
		&value.AssetID, &value.Summary, &snapshot,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return reportapp.ExecutionContext{}, reportapp.ErrNotFound
	}
	if err != nil {
		return reportapp.ExecutionContext{}, err
	}
	value.Snapshot = snapshot
	value.Facts, err = s.loadFacts(ctx, principal, executionID)
	return value, err
}

func (s Store) loadFacts(
	ctx context.Context,
	principal identitydomain.Principal,
	executionID string,
) ([]knowledgedomain.Evidence, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT kind, id::text, locator, content
		FROM (
		  SELECT 'MEASUREMENT' AS kind, m.id,
		         lower(m.measurement_type) AS locator,
		         concat(m.original_value, ' ', m.original_unit, ' (', m.data_quality, ')') AS content
		  FROM measurements m
		  WHERE m.execution_id = $1::uuid AND m.organization_id = $2::uuid
		  UNION ALL
		  SELECT 'OBSERVATION', o.id, coalesce(o.property, 'observation'),
		         o.narrative
		  FROM observations o
		  WHERE o.execution_id = $1::uuid AND o.organization_id = $2::uuid
		  UNION ALL
		  SELECT 'ACTION', a.id, lower(a.action_type), a.narrative
		  FROM maintenance_actions a
		  WHERE a.execution_id = $1::uuid AND a.organization_id = $2::uuid
		) facts
		ORDER BY kind, id
	`, executionID, principal.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []knowledgedomain.Evidence
	for rows.Next() {
		var item knowledgedomain.Evidence
		if err := rows.Scan(
			&item.Kind, &item.SourceID, &item.Locator, &item.Content,
		); err != nil {
			return nil, err
		}
		item.ID = strings.ToLower(item.Kind) + ":" + item.SourceID
		item.Title = item.Kind
		item.Authority = knowledgedomain.AuthoritySiteApproved
		item.ContentHash = skawld.HashBytes([]byte(item.Content))
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) SaveDraft(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	value reportapp.Report,
	_ json.RawMessage,
) (reportapp.Report, bool, error) {
	hash, err := idempotency.HashRequest(map[string]any{
		"execution_id": value.ExecutionID, "input_sha256": value.InputSHA256,
	})
	if err != nil {
		return reportapp.Report{}, false, err
	}
	scope := "report.draft.create.v1:" + value.ExecutionID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay reportapp.Report
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		value.ID = s.IDs.New()
		value.Version = 1
		value.CreatedAt = now
		value.UpdatedAt = now
		if _, err := tx.Exec(ctx, `
			SELECT id FROM maintenance_executions
			WHERE id = $1::uuid AND organization_id = $2::uuid
			FOR UPDATE
		`, value.ExecutionID, value.OrganizationID); err != nil {
			return result{}, err
		}
		if err := tx.QueryRow(ctx, `
			SELECT coalesce(max(revision), 0) + 1
			FROM maintenance_reports
			WHERE execution_id = $1::uuid
		`, value.ExecutionID).Scan(&value.Revision); err != nil {
			return result{}, err
		}
		content, _ := json.Marshal(value.Content)
		evidence, _ := json.Marshal(value.Evidence)
		_, err = tx.Exec(ctx, `
			INSERT INTO maintenance_reports (
				id, organization_id, site_id, execution_id, revision, state,
				structured_content, evidence_snapshot, provider, model,
				model_version, prompt_version, input_sha256, output_sha256,
				generated_by_kind, created_by, created_at, updated_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, 'DRAFT',
				$6::jsonb, $7::jsonb, $8, $9, $10, $11, $12, $13,
				'AI_DRAFT', $14::uuid, $15, $15
			)
		`, value.ID, value.OrganizationID, value.SiteID, value.ExecutionID,
			value.Revision, content, evidence, value.Provider, value.Model,
			value.ModelVersion, value.PromptVersion, value.InputSHA256,
			value.OutputSHA256, principal.ID, now)
		if err != nil {
			return result{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: "report.draft.generated",
			EntityKind: "maintenance_report", EntityID: value.ID,
			ExecutionID: value.ExecutionID,
			AIInvolvement: map[string]any{
				"provider": value.Provider, "model": value.Model,
				"model_version": value.ModelVersion, "prompt_version": value.PromptVersion,
			},
			After: value, OccurredAt: now,
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
		return reportapp.Report{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func (s Store) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	reportID string,
) (reportapp.Report, error) {
	var value reportapp.Report
	var content, evidence []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, site_id::text, execution_id::text,
		       revision, version, state, structured_content, evidence_snapshot,
		       coalesce(provider, ''), coalesce(model, ''),
		       coalesce(model_version, ''), coalesce(prompt_version, ''),
		       coalesce(input_sha256, ''), coalesce(output_sha256, ''),
		       generated_by_kind, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(approved_by::text, ''), approved_at, created_at, updated_at
		FROM maintenance_reports
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
	`, reportID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.ExecutionID,
		&value.Revision, &value.Version, &value.State, &content, &evidence,
		&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
		&value.InputSHA256, &value.OutputSHA256, &value.GeneratedBy,
		&value.SubmittedBy, &value.SubmittedAt, &value.ApprovedBy,
		&value.ApprovedAt, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return reportapp.Report{}, reportapp.ErrNotFound
	}
	if err != nil {
		return reportapp.Report{}, err
	}
	if err := json.Unmarshal(content, &value.Content); err != nil {
		return reportapp.Report{}, err
	}
	if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
		return reportapp.Report{}, err
	}
	return value, nil
}

func (s Store) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter reportapp.ReportFilter,
) ([]reportapp.Report, bool, error) {
	query := `
		SELECT id::text, organization_id::text, site_id::text, execution_id::text,
		       revision, version, state, structured_content, evidence_snapshot,
		       coalesce(provider, ''), coalesce(model, ''),
		       coalesce(model_version, ''), coalesce(prompt_version, ''),
		       coalesce(input_sha256, ''), coalesce(output_sha256, ''),
		       generated_by_kind, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(approved_by::text, ''), approved_at, created_at, updated_at
		FROM maintenance_reports
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
			return nil, false, reportapp.ErrInvalid
		}
		args = append(args, filter.Cursor[:cut], filter.Cursor[cut+1:])
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]reportapp.Report, 0, filter.PageSize+1)
	for rows.Next() {
		var value reportapp.Report
		var content, evidence []byte
		if err := rows.Scan(
			&value.ID, &value.OrganizationID, &value.SiteID, &value.ExecutionID,
			&value.Revision, &value.Version, &value.State, &content, &evidence,
			&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
			&value.InputSHA256, &value.OutputSHA256, &value.GeneratedBy,
			&value.SubmittedBy, &value.SubmittedAt, &value.ApprovedBy,
			&value.ApprovedAt, &value.CreatedAt, &value.UpdatedAt,
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

func (s Store) Edit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command reportapp.Edit,
) (reportapp.Report, bool, error) {
	return s.change(
		ctx, principal, key, reportID, "edit", command,
		func(ctx context.Context, tx pgx.Tx, value *reportapp.Report) error {
			if value.State != "DRAFT" {
				return reportapp.ErrConflict
			}
			now := s.Clock.Now()
			before, _ := json.Marshal(value.Content)
			after, _ := json.Marshal(command.Content)
			var editRevision int
			if err := tx.QueryRow(ctx, `
				SELECT coalesce(max(revision), 0) + 1
				FROM maintenance_report_edits WHERE report_id = $1::uuid
			`, value.ID).Scan(&editRevision); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO maintenance_report_edits (
					id, organization_id, report_id, revision, previous_content,
					new_content, edited_by, edited_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4, $5::jsonb,
					$6::jsonb, $7::uuid, $8
				)
			`, s.IDs.New(), value.OrganizationID, value.ID, editRevision,
				before, after, principal.ID, now)
			if err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `
				UPDATE maintenance_reports
				SET structured_content = $2::jsonb, version = version + 1, updated_at = $3
				WHERE id = $1::uuid AND version = $4 AND state = 'DRAFT'
			`, value.ID, after, now, command.ExpectedVersion)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return reportapp.ErrConflict
			}
			value.Content = command.Content
			value.Version++
			value.UpdatedAt = now
			return nil
		},
	)
}

func (s Store) Submit(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command reportapp.Transition,
) (reportapp.Report, bool, error) {
	return s.transition(ctx, principal, key, reportID, "submit", "DRAFT", "SUBMITTED", command)
}

func (s Store) Approve(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID string,
	command reportapp.Transition,
) (reportapp.Report, bool, error) {
	return s.transition(ctx, principal, key, reportID, "approve", "SUBMITTED", "APPROVED", command)
}

func (s Store) transition(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID, action, from, to string,
	command reportapp.Transition,
) (reportapp.Report, bool, error) {
	return s.change(
		ctx, principal, key, reportID, action, command,
		func(ctx context.Context, tx pgx.Tx, value *reportapp.Report) error {
			if value.State != from {
				return reportapp.ErrConflict
			}
			now := s.Clock.Now()
			column := "submitted"
			if action == "approve" {
				column = "approved"
			}
			statement := `
				UPDATE maintenance_reports
				SET state = $2, ` + column + `_by = $3::uuid, ` + column + `_at = $4,
				    version = version + 1, updated_at = $4
				WHERE id = $1::uuid AND version = $5 AND state = $6
			`
			tag, err := tx.Exec(
				ctx, statement, value.ID, to, principal.ID, now,
				command.ExpectedVersion, from,
			)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return reportapp.ErrConflict
			}
			value.State = to
			value.Version++
			value.UpdatedAt = now
			if action == "submit" {
				value.SubmittedBy, value.SubmittedAt = principal.ID, &now
			} else {
				value.ApprovedBy, value.ApprovedAt = principal.ID, &now
			}
			return nil
		},
	)
}

func (s Store) change(
	ctx context.Context,
	principal identitydomain.Principal,
	key, reportID, action string,
	command any,
	change func(context.Context, pgx.Tx, *reportapp.Report) error,
) (reportapp.Report, bool, error) {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return reportapp.Report{}, false, err
	}
	scope := "report." + action + ".v1:" + reportID
	stored, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (result, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return result{}, err
		}
		if record.Replay {
			var replay reportapp.Report
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return result{}, err
			}
			return result{Value: replay, Replay: true}, nil
		}
		value, err := getForUpdate(ctx, tx, principal, reportID)
		if err != nil {
			return result{}, err
		}
		before := value
		if err := change(ctx, tx, &value); err != nil {
			return result{}, err
		}
		auditAction := map[string]string{
			"edit": "report.edited", "submit": "report.submitted",
			"approve": "report.approved",
		}[action]
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
			ActorID: principal.ID, Action: auditAction,
			EntityKind: "maintenance_report", EntityID: value.ID,
			ExecutionID: value.ExecutionID, Before: before, After: value,
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
		return reportapp.Report{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func getForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	reportID string,
) (reportapp.Report, error) {
	var value reportapp.Report
	var content, evidence []byte
	err := tx.QueryRow(ctx, `
		SELECT id::text, organization_id::text, site_id::text, execution_id::text,
		       revision, version, state, structured_content, evidence_snapshot,
		       coalesce(provider, ''), coalesce(model, ''),
		       coalesce(model_version, ''), coalesce(prompt_version, ''),
		       coalesce(input_sha256, ''), coalesce(output_sha256, ''),
		       generated_by_kind, coalesce(submitted_by::text, ''), submitted_at,
		       coalesce(approved_by::text, ''), approved_at, created_at, updated_at
		FROM maintenance_reports
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
		FOR UPDATE
	`, reportID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.ExecutionID,
		&value.Revision, &value.Version, &value.State, &content, &evidence,
		&value.Provider, &value.Model, &value.ModelVersion, &value.PromptVersion,
		&value.InputSHA256, &value.OutputSHA256, &value.GeneratedBy,
		&value.SubmittedBy, &value.SubmittedAt, &value.ApprovedBy,
		&value.ApprovedAt, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return reportapp.Report{}, reportapp.ErrNotFound
	}
	if err != nil {
		return reportapp.Report{}, err
	}
	if err := json.Unmarshal(content, &value.Content); err != nil {
		return reportapp.Report{}, err
	}
	if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
		return reportapp.Report{}, err
	}
	return value, nil
}

func (s Store) RecordCall(
	ctx context.Context,
	principal identitydomain.Principal,
	execution reportapp.ExecutionContext,
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
	`, s.IDs.New(), execution.OrganizationID, execution.SiteID,
		skawld.CapabilityReportDraft, generation.Metadata.Provider,
		generation.Metadata.Model, generation.Metadata.ModelVersion,
		generation.Prompt, generation.InputHash, generation.OutputHash,
		outcome, generation.Latency.Milliseconds(), generation.Metadata.TokensIn,
		generation.Metadata.TokensOut, generation.Metadata.EstimatedCostMicros,
		errorCode, s.Clock.Now())
	return err
}
