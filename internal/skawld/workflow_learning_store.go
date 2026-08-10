package skawld

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkevaluation "github.com/ZekromNguyen/skawld-sdk-go/evaluation"
	sdkworkflow "github.com/ZekromNguyen/skawld-sdk-go/workflow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SDKWorkflowStore struct {
	Pool        *pgxpool.Pool
	Clock       clock.Clock
	SiteID      string
	AssetClass  string
	WorkflowKey string
	Description string
}

func (s SDKWorkflowStore) SaveCandidate(
	ctx context.Context,
	value sdkworkflow.Version,
) (sdkworkflow.Version, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" || principal.ActorID == "" {
		return sdkworkflow.Version{}, sdkcore.NewPermissionError(
			"workflow persistence requires authenticated tenant and actor",
		)
	}
	value.Status = sdkworkflow.VersionCandidate
	value.Workflow.TenantID = principal.TenantID
	if value.SchemaVersion == "" {
		value.SchemaVersion = sdkworkflow.SchemaVersion
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = s.now()
	}
	if value.CreatedBy == "" {
		value.CreatedBy = principal.ActorID
	}
	if err := value.Validate(); err != nil {
		return sdkworkflow.Version{}, err
	}
	if s.SiteID == "" || s.AssetClass == "" || s.WorkflowKey == "" {
		return sdkworkflow.Version{}, sdkcore.NewConfigError(
			"maintenance workflow metadata is required for candidate persistence",
		)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return sdkworkflow.Version{}, fmt.Errorf("marshal workflow candidate: %w", err)
	}
	digest, err := sdkworkflow.Digest(value)
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		INSERT INTO maintenance_workflows (
			id, organization_id, workflow_key, name, description,
			created_by, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7
		)
		ON CONFLICT (organization_id, workflow_key) DO NOTHING
	`, value.Workflow.ID, principal.TenantID, s.WorkflowKey,
		value.Workflow.Name, s.Description, principal.ActorID, value.CreatedAt)
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	if tag.RowsAffected() == 0 {
		var existingID string
		if err := tx.QueryRow(ctx, `
			SELECT id::text
			FROM maintenance_workflows
			WHERE organization_id = $1::uuid AND workflow_key = $2
		`, principal.TenantID, s.WorkflowKey).Scan(&existingID); err != nil {
			return sdkworkflow.Version{}, err
		}
		if existingID != value.Workflow.ID {
			return sdkworkflow.Version{}, &sdkcore.SkawldError{
				Kind:    sdkcore.ErrorConflict,
				Message: "workflow key is already bound to another workflow identity",
			}
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workflow_versions (
			workflow_id, version, organization_id, site_id, asset_class,
			status, sdk_schema_version, sdk_payload, candidate_digest,
			source_demonstration_ids, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2, $3::uuid, $4::uuid, $5,
			'CANDIDATE', $6, $7::jsonb, $8,
			$9::uuid[], $10::uuid, $11, $11
		)
	`, value.Workflow.ID, value.Version, principal.TenantID, s.SiteID,
		s.AssetClass, value.SchemaVersion, payload, digest,
		value.SourceDemonstrationIDs, principal.ActorID, value.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return sdkworkflow.Version{}, &sdkcore.SkawldError{
				Kind: sdkcore.ErrorConflict, Message: "workflow version already exists",
			}
		}
		return sdkworkflow.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sdkworkflow.Version{}, err
	}
	return value, nil
}

func (s SDKWorkflowStore) Publish(
	ctx context.Context,
	workflowID string,
	number int,
	principal sdkcore.Principal,
) (sdkworkflow.Version, error) {
	authenticated, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || authenticated.TenantID != principal.TenantID ||
		authenticated.ActorID != principal.ActorID {
		return sdkworkflow.Version{}, sdkcore.NewPermissionError(
			"workflow publication identity mismatch",
		)
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	defer tx.Rollback(ctx)
	var payload []byte
	var status string
	err = tx.QueryRow(ctx, `
		SELECT sdk_payload, status
		FROM workflow_versions
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid
		FOR UPDATE
	`, workflowID, number, principal.TenantID).Scan(&payload, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return sdkworkflow.Version{}, &sdkcore.SkawldError{
			Kind: sdkcore.ErrorNotFound, Message: "workflow candidate not found",
		}
	}
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	if status != "APPROVED" {
		return sdkworkflow.Version{}, &sdkcore.SkawldError{
			Kind:    sdkcore.ErrorConflict,
			Message: "only maintenance-approved workflow candidates can be published",
		}
	}
	var value sdkworkflow.Version
	if err := json.Unmarshal(payload, &value); err != nil {
		return sdkworkflow.Version{}, err
	}
	value.Status = sdkworkflow.VersionPublished
	value.PublishedAt = s.now()
	value.PublishedBy = principal.ActorID
	publishedPayload, err := json.Marshal(value)
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE workflow_versions
		SET status = 'RETIRED', retired_at = $3,
		    superseded_by_version = $2, updated_at = $3
		WHERE workflow_id = $1::uuid AND status = 'PUBLISHED'
	`, workflowID, number, value.PublishedAt)
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE workflow_versions
		SET status = 'PUBLISHED', sdk_payload = $4::jsonb,
		    published_by = $5::uuid, published_at = $6, updated_at = $6
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid AND status = 'APPROVED'
	`, workflowID, number, principal.TenantID, publishedPayload,
		principal.ActorID, value.PublishedAt)
	if err != nil {
		return sdkworkflow.Version{}, err
	}
	if tag.RowsAffected() != 1 {
		return sdkworkflow.Version{}, &sdkcore.SkawldError{
			Kind: sdkcore.ErrorConflict, Message: "workflow publication raced with another transition",
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return sdkworkflow.Version{}, err
	}
	return value, nil
}

func (s SDKWorkflowStore) Get(
	ctx context.Context,
	workflowID string,
	number int,
) (sdkworkflow.Version, bool, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return sdkworkflow.Version{}, false, sdkcore.NewPermissionError(
			"workflow read requires authenticated tenant",
		)
	}
	var payload []byte
	var status string
	err := s.Pool.QueryRow(ctx, `
		SELECT sdk_payload, status
		FROM workflow_versions
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid
	`, workflowID, number, principal.TenantID).Scan(&payload, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return sdkworkflow.Version{}, false, nil
	}
	if err != nil {
		return sdkworkflow.Version{}, false, err
	}
	var value sdkworkflow.Version
	if err := json.Unmarshal(payload, &value); err != nil {
		return sdkworkflow.Version{}, false, err
	}
	value.Status = sdkStatus(status)
	return value, true, nil
}

func (s SDKWorkflowStore) Published(
	ctx context.Context,
	workflowID string,
) (sdkworkflow.Version, bool, error) {
	versions, err := s.ListVersions(ctx, workflowID)
	if err != nil {
		return sdkworkflow.Version{}, false, err
	}
	for index := len(versions) - 1; index >= 0; index-- {
		if versions[index].Status == sdkworkflow.VersionPublished {
			return versions[index], true, nil
		}
	}
	return sdkworkflow.Version{}, false, nil
}

func (s SDKWorkflowStore) ListVersions(
	ctx context.Context,
	workflowID string,
) ([]sdkworkflow.Version, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return nil, sdkcore.NewPermissionError(
			"workflow list requires authenticated tenant",
		)
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT sdk_payload, status
		FROM workflow_versions
		WHERE workflow_id = $1::uuid AND organization_id = $2::uuid
		ORDER BY version
	`, workflowID, principal.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []sdkworkflow.Version
	for rows.Next() {
		var payload []byte
		var status string
		if err := rows.Scan(&payload, &status); err != nil {
			return nil, err
		}
		var value sdkworkflow.Version
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, err
		}
		value.Status = sdkStatus(status)
		result = append(result, value)
	}
	return result, rows.Err()
}

type SDKWorkflowReviewStore struct {
	Pool *pgxpool.Pool
}

func (s SDKWorkflowReviewStore) Save(
	ctx context.Context,
	review sdkworkflow.Review,
) error {
	if err := review.Validate(); err != nil {
		return err
	}
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID != review.TenantID ||
		principal.ActorID != review.ReviewedBy {
		return sdkcore.NewPermissionError("workflow review identity mismatch")
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO workflow_reviews (
			id, organization_id, workflow_id, workflow_version,
			candidate_digest, decision, reason, reviewed_by, reviewed_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4,
			$5, upper($6), $7, $8::uuid, $9
		)
	`, review.ID, review.TenantID, review.WorkflowID,
		review.WorkflowVersion, review.CandidateDigest,
		string(review.Decision), review.Reason, review.ReviewedBy,
		review.ReviewedAt)
	return err
}

func (s SDKWorkflowReviewStore) Get(
	ctx context.Context,
	reviewID string,
) (sdkworkflow.Review, bool, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return sdkworkflow.Review{}, false, sdkcore.NewPermissionError(
			"workflow review read requires authenticated tenant",
		)
	}
	var value sdkworkflow.Review
	var decision string
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, workflow_id::text,
		       workflow_version, candidate_digest, lower(decision),
		       reviewed_at, reviewed_by::text, reason
		FROM workflow_reviews
		WHERE id = $1::uuid AND organization_id = $2::uuid
	`, reviewID, principal.TenantID).Scan(
		&value.ID, &value.TenantID, &value.WorkflowID,
		&value.WorkflowVersion, &value.CandidateDigest, &decision,
		&value.ReviewedAt, &value.ReviewedBy, &value.Reason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sdkworkflow.Review{}, false, nil
	}
	value.Decision = sdkworkflow.ReviewDecision(decision)
	return value, err == nil, err
}

func (s SDKWorkflowReviewStore) List(
	ctx context.Context,
	workflowID string,
	version int,
) ([]sdkworkflow.Review, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return nil, sdkcore.NewPermissionError(
			"workflow review list requires authenticated tenant",
		)
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, organization_id::text, workflow_id::text,
		       workflow_version, candidate_digest, lower(decision),
		       reviewed_at, reviewed_by::text, reason
		FROM workflow_reviews
		WHERE organization_id = $1::uuid
		  AND ($2 = '' OR workflow_id::text = $2)
		  AND ($3 = 0 OR workflow_version = $3)
		ORDER BY reviewed_at, id
	`, principal.TenantID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []sdkworkflow.Review
	for rows.Next() {
		var value sdkworkflow.Review
		var decision string
		if err := rows.Scan(
			&value.ID, &value.TenantID, &value.WorkflowID,
			&value.WorkflowVersion, &value.CandidateDigest, &decision,
			&value.ReviewedAt, &value.ReviewedBy, &value.Reason,
		); err != nil {
			return nil, err
		}
		value.Decision = sdkworkflow.ReviewDecision(decision)
		result = append(result, value)
	}
	return result, rows.Err()
}

type SDKEvaluationStore struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
}

func (s SDKEvaluationStore) Save(
	ctx context.Context,
	report sdkevaluation.Report,
) error {
	if err := report.Validate(); err != nil {
		return err
	}
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID != report.TenantID {
		return sdkcore.NewPermissionError("evaluation report tenant mismatch")
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO workflow_evaluation_reports (
			id, organization_id, workflow_id, workflow_version,
			workflow_digest, suite_name, gates_passed, sdk_payload,
			started_at, completed_at, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4,
			$5, $6, $7, $8::jsonb, $9, $10, $11
		)
	`, report.ID, report.TenantID, report.WorkflowID,
		report.WorkflowVersion, report.WorkflowDigest, report.SuiteName,
		report.Gates.Passed, payload, report.StartedAt, report.CompletedAt,
		s.now())
	return err
}

func (s SDKEvaluationStore) Get(
	ctx context.Context,
	reportID string,
) (sdkevaluation.Report, bool, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return sdkevaluation.Report{}, false, sdkcore.NewPermissionError(
			"evaluation read requires authenticated tenant",
		)
	}
	var payload []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT sdk_payload
		FROM workflow_evaluation_reports
		WHERE id = $1::uuid AND organization_id = $2::uuid
	`, reportID, principal.TenantID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return sdkevaluation.Report{}, false, nil
	}
	if err != nil {
		return sdkevaluation.Report{}, false, err
	}
	var value sdkevaluation.Report
	if err := json.Unmarshal(payload, &value); err != nil {
		return sdkevaluation.Report{}, false, err
	}
	return value, true, nil
}

func (s SDKEvaluationStore) List(
	ctx context.Context,
	workflowID string,
	version int,
) ([]sdkevaluation.Report, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return nil, sdkcore.NewPermissionError(
			"evaluation list requires authenticated tenant",
		)
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT sdk_payload
		FROM workflow_evaluation_reports
		WHERE organization_id = $1::uuid
		  AND ($2 = '' OR workflow_id::text = $2)
		  AND ($3 = 0 OR workflow_version = $3)
		ORDER BY completed_at, id
	`, principal.TenantID, workflowID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []sdkevaluation.Report
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var value sdkevaluation.Report
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CompletedAt.Equal(result[j].CompletedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CompletedAt.Before(result[j].CompletedAt)
	})
	return result, nil
}

func (s SDKWorkflowStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s SDKEvaluationStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func sdkStatus(status string) sdkworkflow.VersionStatus {
	switch status {
	case "PUBLISHED":
		return sdkworkflow.VersionPublished
	case "RETIRED":
		return sdkworkflow.VersionRetired
	default:
		return sdkworkflow.VersionCandidate
	}
}

func isUniqueViolation(err error) bool {
	var postgresError interface{ SQLState() string }
	return errors.As(err, &postgresError) && postgresError.SQLState() == "23505"
}
