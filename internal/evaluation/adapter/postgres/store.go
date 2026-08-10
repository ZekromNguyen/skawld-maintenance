package postgres

import (
	"context"

	evaluationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func (s Store) Counts(
	ctx context.Context,
	principal identitydomain.Principal,
) (evaluationdomain.Counts, error) {
	var value evaluationdomain.Counts
	err := s.Pool.QueryRow(ctx, `
		WITH scoped_recommendations AS (
			SELECT *
			FROM recommendations
			WHERE organization_id = $1::uuid
			  AND (
			    COALESCE(cardinality($2::uuid[]), 0) = 0
			    OR site_id = ANY($2::uuid[])
			  )
		),
		latest_feedback AS (
			SELECT DISTINCT ON (feedback.recommendation_id)
			       feedback.*
			FROM recommendation_feedback feedback
			JOIN scoped_recommendations recommendation
			  ON recommendation.id = feedback.recommendation_id
			ORDER BY feedback.recommendation_id,
			         feedback.created_at DESC,
			         feedback.id DESC
		)
		SELECT
			count(recommendation.id),
			count(feedback.id),
			count(*) FILTER (WHERE feedback.outcome = 'ACCEPTED'),
			count(*) FILTER (WHERE feedback.outcome = 'CORRECTED'),
			count(*) FILTER (WHERE feedback.outcome = 'REJECTED'),
			count(*) FILTER (WHERE feedback.outcome = 'UNSAFE'),
			count(*) FILTER (WHERE feedback.outcome = 'UNSUPPORTED'),
			count(*) FILTER (
			    WHERE feedback.outcome = 'INCORRECT_NEXT_STEP'
			),
			coalesce(sum(feedback.material_claims), 0),
			coalesce(sum(feedback.supported_claims), 0),
			coalesce(sum(feedback.retrieved_evidence), 0),
			coalesce(sum(feedback.relevant_evidence), 0)
		FROM scoped_recommendations recommendation
		LEFT JOIN latest_feedback feedback
		  ON feedback.recommendation_id = recommendation.id
	`, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.Recommendations,
		&value.Reviewed,
		&value.Accepted,
		&value.Corrected,
		&value.Rejected,
		&value.Unsafe,
		&value.Unsupported,
		&value.IncorrectNextStep,
		&value.MaterialClaims,
		&value.SupportedClaims,
		&value.RetrievedEvidence,
		&value.RelevantEvidence,
	)
	if err != nil {
		return evaluationdomain.Counts{}, err
	}
	err = s.Pool.QueryRow(ctx, `
		SELECT count(*), coalesce(sum(latency_ms), 0),
		       coalesce(sum(tokens_in), 0), coalesce(sum(tokens_out), 0),
		       coalesce(sum(estimated_cost_micros), 0)
		FROM ai_call_records
		WHERE organization_id = $1::uuid
		  AND (
		    COALESCE(cardinality($2::uuid[]), 0) = 0
		    OR site_id = ANY($2::uuid[])
		  )
	`, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.AICalls,
		&value.LatencyTotalMS,
		&value.TokensIn,
		&value.TokensOut,
		&value.EstimatedCostMicros,
	)
	if err != nil {
		return evaluationdomain.Counts{}, err
	}
	err = s.Pool.QueryRow(ctx, `
		SELECT
			count(report.*),
			count(report.*) FILTER (WHERE report.gates_passed)
		FROM workflow_evaluation_reports report
		JOIN workflow_versions version
		  ON version.workflow_id = report.workflow_id
		 AND version.version = report.workflow_version
		WHERE report.organization_id = $1::uuid
		  AND version.organization_id = $1::uuid
		  AND (
		    COALESCE(cardinality($2::uuid[]), 0) = 0
		    OR version.site_id = ANY($2::uuid[])
		  )
	`, principal.OrganizationID, principal.SiteIDs).Scan(
		&value.WorkflowEvaluations,
		&value.WorkflowGatesPassed,
	)
	return value, err
}
