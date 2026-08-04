package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	copilotapp "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
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

type recommendationResult struct {
	Value  copilotapp.Recommendation
	Replay bool
}

func (s Store) LoadIncidentContext(
	ctx context.Context,
	principal identitydomain.Principal,
	incidentID, requestedExecutionID string,
) (copilotapp.IncidentContext, error) {
	var value copilotapp.IncidentContext
	var snapshot []byte
	err := s.Pool.QueryRow(ctx, `
		WITH selected_execution AS (
			SELECT e.id
			FROM maintenance_executions e
			WHERE e.incident_id = $1::uuid
			  AND (
			    nullif($4, '') IS NULL OR e.id::text = nullif($4, '')
			  )
			ORDER BY
			  CASE WHEN e.id::text = nullif($4, '') THEN 0 ELSE 1 END,
			  e.created_at DESC
			LIMIT 1
		)
		SELECT
			i.organization_id::text, i.site_id::text, i.id::text,
			coalesce(se.id::text, ''), i.asset_id::text, i.summary,
			jsonb_build_object(
			  'incident', jsonb_build_object(
			    'id', i.id, 'number', i.number, 'summary', i.summary,
			    'severity', i.severity, 'state', i.state, 'detected_at', i.detected_at
			  ),
			  'asset', jsonb_build_object(
			    'id', a.id, 'tag', a.tag, 'name', a.name, 'class', a.asset_class,
			    'manufacturer', a.manufacturer, 'model', a.model
			  ),
			  'measurements', coalesce((
			    SELECT jsonb_agg(jsonb_build_object(
			      'id', m.id, 'type', m.measurement_type, 'value', m.original_value,
			      'unit', m.original_unit, 'quality', m.data_quality,
			      'verification', m.verification_status, 'observed_at', m.observed_at
			    ) ORDER BY m.observed_at)
			    FROM measurements m
			    WHERE m.execution_id = se.id
			  ), '[]'::jsonb),
			  'observations', coalesce((
			    SELECT jsonb_agg(jsonb_build_object(
			      'id', o.id, 'narrative', o.narrative, 'source', o.source,
			      'verification', o.verification_status, 'observed_at', o.observed_at
			    ) ORDER BY o.observed_at)
			    FROM observations o
			    WHERE o.execution_id = se.id
			  ), '[]'::jsonb),
			  'summary', i.summary
			)
		FROM incidents i
		JOIN assets a ON a.id = i.asset_id
		LEFT JOIN selected_execution se ON true
		WHERE i.id = $1::uuid AND i.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR i.site_id = ANY($3::uuid[]))
	`, incidentID, principal.OrganizationID, principal.SiteIDs,
		requestedExecutionID).Scan(
		&value.OrganizationID, &value.SiteID, &value.IncidentID,
		&value.ExecutionID, &value.AssetID, &value.Summary, &snapshot,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return copilotapp.IncidentContext{}, copilotapp.ErrNotFound
	}
	if err != nil {
		return copilotapp.IncidentContext{}, err
	}
	value.Snapshot = snapshot
	return value, nil
}

func (s Store) SaveRecommendation(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command copilotapp.GenerateRecommendation,
	value copilotapp.Recommendation,
	contextSnapshot json.RawMessage,
) (copilotapp.Recommendation, bool, error) {
	requestHash, err := idempotency.HashRequest(map[string]any{
		"incident_id": value.IncidentID,
		"command":     command,
	})
	if err != nil {
		return copilotapp.Recommendation{}, false, err
	}
	scope := "copilot.recommendation.generate.v1:" + value.IncidentID
	stored, err := database.InTx(
		ctx, s.Pool, pgx.TxOptions{},
		func(tx pgx.Tx) (recommendationResult, error) {
			now := s.Clock.Now()
			record, err := s.Idempotency.Begin(
				ctx, tx, principal.ID, scope, key, requestHash, now,
			)
			if err != nil {
				return recommendationResult{}, err
			}
			if record.Replay {
				var replay copilotapp.Recommendation
				if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
					return recommendationResult{}, err
				}
				return recommendationResult{Value: replay, Replay: true}, nil
			}
			value.ID = s.IDs.New()
			value.CreatedAt = now
			output, _ := json.Marshal(value.Output)
			evidence, _ := json.Marshal(value.Evidence)
			_, err = tx.Exec(ctx, `
				INSERT INTO recommendations (
					id, organization_id, site_id, incident_id, execution_id,
					status, risk_level, structured_output, evidence_snapshot,
					context_snapshot, provider, model, model_version, prompt_version,
					input_sha256, output_sha256, latency_ms, created_by, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4::uuid, nullif($5, '')::uuid,
					$6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12, $13,
					$14, $15, $16, $17, $18::uuid, $19
				)
			`, value.ID, value.OrganizationID, value.SiteID, value.IncidentID,
				value.ExecutionID, value.Output.Status, value.Output.RiskLevel,
				output, evidence, contextSnapshot, value.Provider, value.Model,
				value.ModelVersion, value.PromptVersion, value.InputSHA256,
				value.OutputSHA256, value.LatencyMS, principal.ID, value.CreatedAt)
			if err != nil {
				return recommendationResult{}, err
			}
			if err := s.Audit.Append(ctx, tx, audit.Event{
				ID: s.IDs.New(), OrganizationID: value.OrganizationID, SiteID: value.SiteID,
				ActorID: principal.ID, Action: "copilot.recommendation.generated",
				EntityKind: "recommendation", EntityID: value.ID,
				AIInvolvement: map[string]any{
					"provider": value.Provider, "model": value.Model,
					"model_version": value.ModelVersion, "prompt_version": value.PromptVersion,
				},
				After: map[string]any{
					"status": value.Output.Status, "risk_level": value.Output.RiskLevel,
					"evidence_ids": value.Output.EvidenceIDs,
				},
				OccurredAt: value.CreatedAt,
			}); err != nil {
				return recommendationResult{}, err
			}
			if value.ExecutionID != "" {
				if err := events.Append(ctx, tx, events.Event{
					ID: s.IDs.New(), OrganizationID: value.OrganizationID,
					SiteID: value.SiteID, ActorID: principal.ID,
					Type:          "copilot.recommendation.generated",
					AggregateType: "recommendation", AggregateID: value.ID,
					AggregateVersion: 1, SubjectKind: "EXECUTION",
					SubjectID: value.ExecutionID,
					Payload: map[string]any{
						"execution_id": value.ExecutionID,
						"value":        value,
					},
					OccurredAt: value.CreatedAt,
				}); err != nil {
					return recommendationResult{}, err
				}
			}
			response, _ := json.Marshal(value)
			if err := s.Idempotency.Complete(
				ctx, tx, principal.ID, scope, key,
				http.StatusCreated, response, now,
			); err != nil {
				return recommendationResult{}, err
			}
			return recommendationResult{Value: value}, nil
		},
	)
	if err != nil {
		return copilotapp.Recommendation{}, false, err
	}
	return stored.Value, stored.Replay, nil
}

func (s Store) GetRecommendation(
	ctx context.Context,
	principal identitydomain.Principal,
	recommendationID string,
) (copilotapp.Recommendation, error) {
	var value copilotapp.Recommendation
	var output, evidence []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, site_id::text,
		       incident_id::text, coalesce(execution_id::text, ''),
		       structured_output, evidence_snapshot, provider, model,
		       model_version, prompt_version, input_sha256, output_sha256,
		       latency_ms, created_at
		FROM recommendations
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
	`, recommendationID, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.ID, &value.OrganizationID, &value.SiteID, &value.IncidentID,
		&value.ExecutionID, &output, &evidence, &value.Provider, &value.Model,
		&value.ModelVersion, &value.PromptVersion, &value.InputSHA256,
		&value.OutputSHA256, &value.LatencyMS, &value.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return copilotapp.Recommendation{}, copilotapp.ErrNotFound
	}
	if err != nil {
		return copilotapp.Recommendation{}, err
	}
	if err := json.Unmarshal(output, &value.Output); err != nil {
		return copilotapp.Recommendation{}, err
	}
	if err := json.Unmarshal(evidence, &value.Evidence); err != nil {
		return copilotapp.Recommendation{}, err
	}
	return value, nil
}

func (s Store) RecordCall(
	ctx context.Context,
	principal identitydomain.Principal,
	incident copilotapp.IncidentContext,
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
	`, s.IDs.New(), incident.OrganizationID, incident.SiteID,
		skawld.CapabilityRecommendation, generation.Metadata.Provider,
		generation.Metadata.Model, generation.Metadata.ModelVersion,
		generation.Prompt, generation.InputHash, generation.OutputHash,
		outcome, generation.Latency.Milliseconds(), generation.Metadata.TokensIn,
		generation.Metadata.TokensOut, generation.Metadata.EstimatedCostMicros,
		errorCode, s.Clock.Now())
	return err
}

func (s Store) RecordFeedback(
	ctx context.Context,
	principal identitydomain.Principal,
	key, recommendationID string,
	command copilotapp.Feedback,
) error {
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return err
	}
	scope := "copilot.recommendation.feedback.v1:" + recommendationID
	_, err = database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return struct{}{}, err
		}
		if record.Replay {
			return struct{}{}, nil
		}
		var organizationID, siteID, executionID string
		err = tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text,
			       coalesce(execution_id::text, '')
			FROM recommendations
			WHERE id = $1::uuid AND organization_id = $2::uuid
			  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
		`, recommendationID, principal.OrganizationID, principal.SiteIDs).Scan(
			&organizationID, &siteID, &executionID,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return struct{}{}, copilotapp.ErrNotFound
		}
		if err != nil {
			return struct{}{}, err
		}
		feedbackID := s.IDs.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO recommendation_feedback (
				id, organization_id, recommendation_id, outcome,
				correction, reason, material_claims, supported_claims,
				retrieved_evidence, relevant_evidence, actor_id, created_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, nullif($5, ''),
				nullif($6, ''), $7, $8, $9, $10, $11::uuid, $12
			)
		`, feedbackID, organizationID, recommendationID, command.Outcome,
			command.Correction, command.Reason, command.MaterialClaims,
			command.SupportedClaims, command.RetrievedEvidence,
			command.RelevantEvidence, principal.ID, now)
		if err != nil {
			return struct{}{}, err
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID: s.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
			ActorID: principal.ID, Action: "copilot.recommendation.reviewed",
			EntityKind: "recommendation", EntityID: recommendationID,
			After: command, OccurredAt: now,
		}); err != nil {
			return struct{}{}, err
		}
		if executionID != "" {
			if err := events.Append(ctx, tx, events.Event{
				ID: s.IDs.New(), OrganizationID: organizationID, SiteID: siteID,
				ActorID: principal.ID, Type: "copilot.recommendation.reviewed",
				AggregateType: "recommendation_feedback", AggregateID: feedbackID,
				AggregateVersion: 1, SubjectKind: "EXECUTION",
				SubjectID: executionID,
				Payload: map[string]any{
					"execution_id":      executionID,
					"recommendation_id": recommendationID,
					"feedback_id":       feedbackID,
					"value":             command,
				},
				OccurredAt: now,
			}); err != nil {
				return struct{}{}, err
			}
		}
		response, _ := json.Marshal(map[string]string{"id": feedbackID})
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, http.StatusCreated, response, now,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}
