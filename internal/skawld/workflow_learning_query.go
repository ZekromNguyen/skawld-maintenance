package skawld

import (
	"context"
	"encoding/json"
	"time"

	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
	sdkevaluation "github.com/ZekromNguyen/skawld-sdk-go/evaluation"
)

func (g WorkflowLearningGateway) loadWorkflowApplicability(
	ctx context.Context,
	organizationID, workflowID string,
	version int,
) ([]workflowapp.Applicability, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, coalesce(site_id::text, ''),
		       coalesce(asset_id::text, ''), coalesce(asset_class, ''),
		       coalesce(manufacturer, ''), coalesce(model, ''),
		       coalesce(process_service, ''), coalesce(operating_condition, ''),
		       validation_status, approved_by::text, approved_at
		FROM workflow_applicability
		WHERE organization_id = $1::uuid
		  AND workflow_id = $2::uuid AND workflow_version = $3
		ORDER BY created_at, id
	`, organizationID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []workflowapp.Applicability
	for rows.Next() {
		var value workflowapp.Applicability
		var approvedAt time.Time
		if err := rows.Scan(
			&value.ID, &value.SiteID, &value.AssetID, &value.AssetClass,
			&value.Manufacturer, &value.Model, &value.ProcessService,
			&value.OperatingCondition, &value.ValidationStatus,
			&value.ApprovedBy, &approvedAt,
		); err != nil {
			return nil, err
		}
		value.ApprovedAt = approvedAt.UTC().Format(time.RFC3339)
		result = append(result, value)
	}
	return result, rows.Err()
}

func (g WorkflowLearningGateway) loadWorkflowReviews(
	ctx context.Context,
	organizationID, workflowID string,
	version int,
) ([]workflowapp.ReviewRecord, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, candidate_digest, decision, reason,
		       reviewed_by::text, reviewed_at
		FROM workflow_reviews
		WHERE organization_id = $1::uuid
		  AND workflow_id = $2::uuid AND workflow_version = $3
		ORDER BY reviewed_at, id
	`, organizationID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []workflowapp.ReviewRecord
	for rows.Next() {
		var value workflowapp.ReviewRecord
		if err := rows.Scan(
			&value.ID, &value.CandidateDigest, &value.Decision,
			&value.Reason, &value.ReviewedBy, &value.ReviewedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (g WorkflowLearningGateway) loadWorkflowEvaluations(
	ctx context.Context,
	organizationID, workflowID string,
	version int,
) ([]workflowapp.Evaluation, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, suite_name, gates_passed, sdk_payload, completed_at
		FROM workflow_evaluation_reports
		WHERE organization_id = $1::uuid
		  AND workflow_id = $2::uuid AND workflow_version = $3
		ORDER BY completed_at, id
	`, organizationID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []workflowapp.Evaluation
	for rows.Next() {
		var value workflowapp.Evaluation
		var payload []byte
		if err := rows.Scan(
			&value.ID, &value.SuiteName, &value.GatesPassed,
			&payload, &value.CompletedAt,
		); err != nil {
			return nil, err
		}
		var report sdkevaluation.Report
		if err := json.Unmarshal(payload, &report); err != nil {
			return nil, err
		}
		if err := decodeMap(report.Metrics, &value.Metrics); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (g WorkflowLearningGateway) loadWorkflowImprovements(
	ctx context.Context,
	organizationID, workflowID string,
	version int,
) ([]workflowapp.ImprovementCandidate, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, demonstration_id::text, correction_event_id::text,
		       corrected_event_id::text, corrected_action, coalesce(reason, ''),
		       context_snapshot, outcome_snapshot, status, created_at
		FROM workflow_improvement_candidates
		WHERE organization_id = $1::uuid
		  AND workflow_id = $2::uuid AND workflow_version = $3
		ORDER BY created_at, id
	`, organizationID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []workflowapp.ImprovementCandidate
	for rows.Next() {
		var value workflowapp.ImprovementCandidate
		var contextPayload, outcomePayload []byte
		if err := rows.Scan(
			&value.ID, &value.DemonstrationID,
			&value.CorrectionEventID, &value.CorrectedEventID,
			&value.CorrectedAction, &value.Reason,
			&contextPayload, &outcomePayload,
			&value.Status, &value.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(contextPayload, &value.Context); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(outcomePayload, &value.Outcome); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
