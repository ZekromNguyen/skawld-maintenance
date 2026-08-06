package skawld

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkevaluation "github.com/ZekromNguyen/skawld-sdk-go/evaluation"
	sdklearning "github.com/ZekromNguyen/skawld-sdk-go/learning"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	sdkworkflow "github.com/ZekromNguyen/skawld-sdk-go/workflow"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const workflowEvaluationSuite = "maintenance.workflow.v1"

type WorkflowLearningGateway struct {
	Pool  *pgxpool.Pool
	IDs   id.Generator
	Clock clock.Clock
	Audit audit.Sink
}

type learningScope struct {
	WorkflowKey  string
	SiteID       string
	AssetID      string
	AssetClass   string
	Manufacturer string
	Model        string
}

func (g WorkflowLearningGateway) Compile(
	ctx context.Context,
	principal identitydomain.Principal,
	command workflowapp.Compile,
) (workflowapp.Version, error) {
	sdkContext, _, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return workflowapp.Version{}, err
	}
	demonstrations, scope, err := g.loadReviewedDemonstrations(
		sdkContext, principal, command.DemonstrationIDs,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	workflowID, err := g.workflowIdentity(
		ctx, principal.OrganizationID, scope.WorkflowKey,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	catalog, err := maintenanceToolCatalog()
	if err != nil {
		return workflowapp.Version{}, err
	}
	store := SDKWorkflowStore{
		Pool: g.Pool, Clock: g.Clock, SiteID: scope.SiteID,
		AssetClass: scope.AssetClass, WorkflowKey: scope.WorkflowKey,
		Description: command.Description,
	}
	compiler := sdklearning.Compiler{
		Extractor: maintenanceWorkflowExtractor{},
		Tools:     catalog,
		Store:     store,
		Now:       g.now,
	}
	compilation, err := compiler.CompileMultiple(
		sdkContext, workflowID, command.Name, demonstrations,
		sdklearning.MultiDemoOptions{
			MinimumDemonstrations:         2,
			CommonActionThreshold:         2.0 / 3.0,
			MinimumSequenceConsistency:    0.5,
			MinimumEvidenceDemonstrations: 2,
			AllowConflicts:                true,
		},
	)
	if err != nil {
		return workflowapp.Version{}, mapWorkflowError(err)
	}
	analysis, err := json.Marshal(compilation.Analysis)
	if err != nil {
		return workflowapp.Version{}, err
	}
	changes, err := json.Marshal(compilation.Changes)
	if err != nil {
		return workflowapp.Version{}, err
	}
	_, err = g.Pool.Exec(ctx, `
		UPDATE workflow_versions
		SET analysis = $4::jsonb, behavioral_changes = $5::jsonb,
		    updated_at = $6
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid AND status = 'CANDIDATE'
	`, workflowID, compilation.Candidate.Version, principal.OrganizationID,
		analysis, changes, g.now())
	if err != nil {
		return workflowapp.Version{}, err
	}
	if err := g.captureImprovementCandidates(
		ctx, principal, workflowID, compilation.Candidate.Version,
		demonstrations,
	); err != nil {
		return workflowapp.Version{}, err
	}
	if err := g.appendWorkflowAudit(
		ctx, principal, scope.SiteID, workflowID,
		"workflow.candidate.compiled", command.Description,
		map[string]interface{}{
			"version":               compilation.Candidate.Version,
			"demonstration_ids":     command.DemonstrationIDs,
			"sequence_consistency":  compilation.Analysis.SequenceConsistency,
			"conflict_count":        len(compilation.Analysis.Conflicts),
			"tool_catalog_digest":   compilation.Candidate.ToolCatalogDigest,
			"requires_human_review": true,
		},
	); err != nil {
		return workflowapp.Version{}, err
	}
	return g.Get(ctx, principal, workflowID, compilation.Candidate.Version)
}

func (g WorkflowLearningGateway) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
) (workflowapp.Version, error) {
	var result workflowapp.Version
	var sdkPayload, analysis, changes []byte
	var effectiveAt, reviewAt, publishedAt *time.Time
	err := g.Pool.QueryRow(ctx, `
		SELECT w.id::text, w.workflow_key, w.name, w.description,
		       v.version, v.status, v.site_id::text, v.asset_class,
		       v.candidate_digest, v.sdk_payload, v.analysis,
		       v.behavioral_changes, v.source_demonstration_ids::text[],
		       v.prerequisites, v.required_competencies,
		       v.effective_at, v.review_at, v.created_at,
		       v.published_at, coalesce(v.published_by::text, '')
		FROM maintenance_workflows w
		JOIN workflow_versions v ON v.workflow_id = w.id
		WHERE w.id = $1::uuid AND v.version = $2
		  AND w.organization_id = $3::uuid
		  AND (COALESCE(cardinality($4::uuid[]), 0) = 0 OR v.site_id = ANY($4::uuid[]))
	`, workflowID, version, principal.OrganizationID, principal.SiteIDs).Scan(
		&result.WorkflowID, &result.WorkflowKey, &result.Name,
		&result.Description, &result.Version, &result.Status,
		&result.SiteID, &result.AssetClass, &result.CandidateDigest,
		&sdkPayload, &analysis, &changes, &result.SourceDemonstrations,
		&result.Prerequisites, &result.RequiredCompetencies,
		&effectiveAt, &reviewAt, &result.CreatedAt,
		&publishedAt, &result.PublishedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return workflowapp.Version{}, workflowapp.ErrNotFound
	}
	if err != nil {
		return workflowapp.Version{}, err
	}
	result.EffectiveAt, result.ReviewAt = effectiveAt, reviewAt
	result.PublishedAt = publishedAt
	var sdkVersion sdkworkflow.Version
	if err := json.Unmarshal(sdkPayload, &sdkVersion); err != nil {
		return workflowapp.Version{}, err
	}
	result.ToolCatalogDigest = sdkVersion.ToolCatalogDigest
	result.Steps = mapWorkflowSteps(sdkVersion.Steps)
	if sdkVersion.Learning != nil {
		if err := decodeMap(sdkVersion.Learning, &result.Learning); err != nil {
			return workflowapp.Version{}, err
		}
	}
	if err := json.Unmarshal(analysis, &result.Analysis); err != nil {
		return workflowapp.Version{}, err
	}
	if err := json.Unmarshal(changes, &result.BehavioralChanges); err != nil {
		return workflowapp.Version{}, err
	}
	result.Applicability, err = g.loadWorkflowApplicability(
		ctx, principal.OrganizationID, workflowID, version,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	result.Reviews, err = g.loadWorkflowReviews(
		ctx, principal.OrganizationID, workflowID, version,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	result.Evaluations, err = g.loadWorkflowEvaluations(
		ctx, principal.OrganizationID, workflowID, version,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	result.Improvements, err = g.loadWorkflowImprovements(
		ctx, principal.OrganizationID, workflowID, version,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	return result, nil
}

func (g WorkflowLearningGateway) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter workflowapp.ListFilter,
) ([]workflowapp.Version, bool, error) {
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	query := `
		SELECT v.workflow_id::text, v.version
		FROM workflow_versions v
		WHERE v.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR v.site_id = ANY($2::uuid[]))`
	args := []any{principal.OrganizationID, principal.SiteIDs}
	if filter.Cursor != "" {
		parts := strings.SplitN(filter.Cursor, "|", 3)
		if len(parts) != 3 {
			return nil, false, workflowapp.ErrInvalid
		}
		args = append(args, parts[0], parts[1], parts[2])
		query += fmt.Sprintf(
			" AND (v.created_at, v.workflow_id, v.version) < ($%d, $%d::uuid, $%d)",
			len(args)-2, len(args)-1, len(args))
	}
	args = append(args, filter.PageSize+1)
	query += fmt.Sprintf(
		" ORDER BY v.created_at DESC, v.workflow_id DESC, v.version DESC LIMIT $%d",
		len(args))

	rows, err := g.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	type identity struct {
		workflowID string
		version    int
	}
	var identities []identity
	for rows.Next() {
		var value identity
		if err := rows.Scan(&value.workflowID, &value.version); err != nil {
			return nil, false, err
		}
		identities = append(identities, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(identities) > filter.PageSize
	if hasMore {
		identities = identities[:filter.PageSize]
	}
	result := make([]workflowapp.Version, 0, len(identities))
	for _, identity := range identities {
		value, err := g.Get(ctx, principal, identity.workflowID, identity.version)
		if err != nil {
			return nil, false, err
		}
		result = append(result, value)
	}
	return result, hasMore, nil
}

func (g WorkflowLearningGateway) Review(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command workflowapp.Review,
) (workflowapp.Version, error) {
	sdkContext, sdkPrincipal, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return workflowapp.Version{}, err
	}
	store := SDKWorkflowStore{Pool: g.Pool, Clock: g.Clock}
	candidate, exists, err := store.Get(sdkContext, workflowID, version)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if !exists || candidate.Status != sdkworkflow.VersionCandidate {
		return workflowapp.Version{}, workflowapp.ErrConflict
	}
	sdkDecision := sdkworkflow.ReviewRejected
	if command.Decision == "APPROVED" {
		sdkDecision = sdkworkflow.ReviewApproved
	}
	review, err := sdkworkflow.NewReview(
		candidate, sdkDecision, sdkPrincipal, command.Reason, g.now(),
	)
	if err != nil {
		return workflowapp.Version{}, mapWorkflowError(err)
	}
	applicabilityPayload, err := json.Marshal(command.Applicability)
	if err != nil {
		return workflowapp.Version{}, err
	}
	tx, err := g.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return workflowapp.Version{}, err
	}
	defer tx.Rollback(ctx)
	var siteID, assetClass, status string
	err = tx.QueryRow(ctx, `
		SELECT site_id::text, asset_class, status
		FROM workflow_versions
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid
		FOR UPDATE
	`, workflowID, version, principal.OrganizationID).Scan(
		&siteID, &assetClass, &status,
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if status != "CANDIDATE" && status != "REVIEW_REQUIRED" {
		return workflowapp.Version{}, workflowapp.ErrConflict
	}
	if command.Decision == "APPROVED" {
		for _, applicability := range command.Applicability {
			if applicability.SiteID != "" && applicability.SiteID != siteID ||
				applicability.AssetClass != "" &&
					applicability.AssetClass != assetClass {
				return workflowapp.Version{}, workflowapp.ErrInvalid
			}
			if err := g.validateApplicability(
				ctx, tx, principal, applicability,
			); err != nil {
				return workflowapp.Version{}, err
			}
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workflow_reviews (
			id, organization_id, workflow_id, workflow_version,
			candidate_digest, decision, reason, applicability_snapshot,
			prerequisites, required_competencies, effective_at, review_at,
			reviewed_by, reviewed_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4,
			$5, $6, $7, $8::jsonb,
			$9, $10, $11, $12, $13::uuid, $14
		)
	`, review.ID, principal.OrganizationID, workflowID, version,
		review.CandidateDigest, command.Decision, command.Reason,
		applicabilityPayload, command.Prerequisites,
		command.RequiredCompetencies, nullableTime(command.EffectiveAt),
		nullableTime(command.ReviewAt), principal.ID, review.ReviewedAt)
	if err != nil {
		return workflowapp.Version{}, err
	}
	nextStatus := command.Decision
	if command.Decision == "APPROVED" {
		nextStatus = "APPROVED"
	}
	_, err = tx.Exec(ctx, `
		UPDATE workflow_versions
		SET status = $4, prerequisites = $5,
		    required_competencies = $6,
		    effective_at = $7, review_at = $8,
		    approved_by = CASE WHEN $4 = 'APPROVED' THEN $9::uuid ELSE NULL END,
		    approved_at = CASE
		        WHEN $4 = 'APPROVED' THEN $10::timestamptz
		        ELSE NULL::timestamptz
		    END,
		    updated_at = $10
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid
	`, workflowID, version, principal.OrganizationID, nextStatus,
		command.Prerequisites, command.RequiredCompetencies,
		nullableTime(command.EffectiveAt), nullableTime(command.ReviewAt),
		principal.ID, review.ReviewedAt)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if command.Decision == "APPROVED" {
		for _, applicability := range command.Applicability {
			if err := g.insertApplicability(
				ctx, tx, principal, workflowID, version, applicability,
			); err != nil {
				return workflowapp.Version{}, err
			}
		}
	}
	if err := g.Audit.Append(ctx, tx, audit.Event{
		ID: g.newID(), OrganizationID: principal.OrganizationID,
		SiteID: siteID, ActorID: principal.ID,
		Action: "workflow.candidate.reviewed", EntityKind: "workflow_version",
		EntityID:   fmt.Sprintf("%s:%d", workflowID, version),
		WorkflowID: workflowID, Reason: command.Reason,
		After: map[string]interface{}{
			"decision":         command.Decision,
			"candidate_digest": review.CandidateDigest,
			"applicability":    command.Applicability,
		},
		OccurredAt: review.ReviewedAt,
	}); err != nil {
		return workflowapp.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return workflowapp.Version{}, err
	}
	return g.Get(ctx, principal, workflowID, version)
}

func (g WorkflowLearningGateway) Publish(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command workflowapp.Publish,
) (workflowapp.Version, error) {
	sdkContext, sdkPrincipal, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return workflowapp.Version{}, err
	}
	store := SDKWorkflowStore{Pool: g.Pool, Clock: g.Clock}
	candidate, exists, err := store.Get(sdkContext, workflowID, version)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if !exists || candidate.Status != sdkworkflow.VersionCandidate {
		return workflowapp.Version{}, workflowapp.ErrConflict
	}
	catalog, err := maintenanceToolCatalog()
	if err != nil {
		return workflowapp.Version{}, err
	}
	evaluations := SDKEvaluationStore{Pool: g.Pool, Clock: g.Clock}
	suite, err := workflowEvaluation(candidate, catalog)
	if err != nil {
		return workflowapp.Version{}, mapWorkflowError(err)
	}
	report, err := sdkevaluation.NewRunner(sdkevaluation.RunnerOptions{
		Store: evaluations,
		Now:   g.now,
	}).Run(sdkContext, suite, candidate)
	if err != nil {
		return workflowapp.Version{}, mapWorkflowError(err)
	}
	if !report.Gates.Passed {
		return workflowapp.Version{}, workflowapp.ErrPolicy
	}
	publisher, err := sdkevaluation.NewPublisher(
		sdkevaluation.PublisherOptions{
			Workflows:     store,
			Reports:       evaluations,
			Reviews:       SDKWorkflowReviewStore{Pool: g.Pool},
			ToolCatalog:   catalog,
			RequiredSuite: workflowEvaluationSuite,
			MaxReportAge:  24 * time.Hour,
			Now:           g.now,
		},
	)
	if err != nil {
		return workflowapp.Version{}, err
	}
	published, err := publisher.Publish(
		sdkContext, workflowID, version, sdkPrincipal,
	)
	if err != nil {
		return workflowapp.Version{}, mapWorkflowError(err)
	}
	current, err := g.Get(ctx, principal, workflowID, version)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if err := g.appendWorkflowAudit(
		ctx, principal, current.SiteID, workflowID,
		"workflow.version.published", command.Reason,
		map[string]interface{}{
			"version": version, "candidate_digest": current.CandidateDigest,
			"evaluation_report_id": report.ID,
			"tool_catalog_digest":  published.ToolCatalogDigest,
		},
	); err != nil {
		return workflowapp.Version{}, err
	}
	return g.Get(ctx, principal, workflowID, version)
}

func (g WorkflowLearningGateway) ExpandApplicability(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command workflowapp.ExpandApplicability,
) (workflowapp.Version, error) {
	current, err := g.Get(ctx, principal, workflowID, version)
	if err != nil {
		return workflowapp.Version{}, err
	}
	tx, err := g.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return workflowapp.Version{}, err
	}
	defer tx.Rollback(ctx)
	if err := g.validateApplicability(
		ctx, tx, principal, command.Applicability,
	); err != nil {
		return workflowapp.Version{}, err
	}
	if err := g.insertApplicability(
		ctx, tx, principal, workflowID, version, command.Applicability,
	); err != nil {
		return workflowapp.Version{}, err
	}
	if err := g.Audit.Append(ctx, tx, audit.Event{
		ID: g.newID(), OrganizationID: principal.OrganizationID,
		SiteID: current.SiteID, ActorID: principal.ID,
		Action:     "workflow.applicability.expanded",
		EntityKind: "workflow_version",
		EntityID:   fmt.Sprintf("%s:%d", workflowID, version),
		WorkflowID: workflowID, Reason: command.Reason,
		After: command.Applicability, OccurredAt: g.now(),
	}); err != nil {
		return workflowapp.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return workflowapp.Version{}, err
	}
	return g.Get(ctx, principal, workflowID, version)
}

func (g WorkflowLearningGateway) Retire(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	command workflowapp.Retire,
) (workflowapp.Version, error) {
	current, err := g.Get(ctx, principal, workflowID, version)
	if err != nil {
		return workflowapp.Version{}, err
	}
	now := g.now()
	tx, err := g.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return workflowapp.Version{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE workflow_versions
		SET status = 'RETIRED', retired_by = $4::uuid,
		    retired_at = $5, updated_at = $5
		WHERE workflow_id = $1::uuid AND version = $2
		  AND organization_id = $3::uuid AND status = 'PUBLISHED'
	`, workflowID, version, principal.OrganizationID, principal.ID, now)
	if err != nil {
		return workflowapp.Version{}, err
	}
	if tag.RowsAffected() != 1 {
		return workflowapp.Version{}, workflowapp.ErrConflict
	}
	if err := g.Audit.Append(ctx, tx, audit.Event{
		ID: g.newID(), OrganizationID: principal.OrganizationID,
		SiteID: current.SiteID, ActorID: principal.ID,
		Action: "workflow.version.retired", EntityKind: "workflow_version",
		EntityID:   fmt.Sprintf("%s:%d", workflowID, version),
		WorkflowID: workflowID, Reason: command.Reason, OccurredAt: now,
	}); err != nil {
		return workflowapp.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return workflowapp.Version{}, err
	}
	return g.Get(ctx, principal, workflowID, version)
}

func (g WorkflowLearningGateway) Applicable(
	ctx context.Context,
	principal identitydomain.Principal,
	assetID string,
) ([]workflowapp.Version, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT DISTINCT v.workflow_id::text, v.version
		FROM assets a
		JOIN workflow_versions v
		  ON v.organization_id = a.organization_id
		 AND v.status = 'PUBLISHED'
		 AND v.effective_at <= $4
		 AND v.review_at > $4
		JOIN workflow_applicability ap
		  ON ap.workflow_id = v.workflow_id
		 AND ap.workflow_version = v.version
		 AND ap.organization_id = a.organization_id
		 AND ap.validation_status IN ('VALIDATED', 'LIKELY_APPLICABLE')
		 AND (ap.site_id IS NULL OR ap.site_id = a.site_id)
		 AND (ap.asset_id IS NULL OR ap.asset_id = a.id)
		 AND (nullif(ap.asset_class, '') IS NULL OR ap.asset_class = a.asset_class)
		 AND (nullif(ap.manufacturer, '') IS NULL OR ap.manufacturer = a.manufacturer)
		 AND (nullif(ap.model, '') IS NULL OR ap.model = a.model)
		WHERE a.id = $1::uuid AND a.organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR a.site_id = ANY($3::uuid[]))
		ORDER BY v.version DESC
	`, assetID, principal.OrganizationID, principal.SiteIDs, g.now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type identity struct {
		id      string
		version int
	}
	var identities []identity
	for rows.Next() {
		var value identity
		if err := rows.Scan(&value.id, &value.version); err != nil {
			return nil, err
		}
		identities = append(identities, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]workflowapp.Version, 0, len(identities))
	for _, identity := range identities {
		value, err := g.Get(ctx, principal, identity.id, identity.version)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (g WorkflowLearningGateway) loadReviewedDemonstrations(
	ctx context.Context,
	principal identitydomain.Principal,
	ids []string,
) ([]sdkobservation.Demonstration, learningScope, error) {
	store := ObservationStore{Pool: g.Pool, Clock: g.Clock}
	result := make([]sdkobservation.Demonstration, 0, len(ids))
	var scope learningScope
	for _, demonstrationID := range ids {
		var organizationID, siteID, workflowKey, status, reviewStatus string
		var initialContext []byte
		var incomplete int
		err := g.Pool.QueryRow(ctx, `
			SELECT d.organization_id::text, d.site_id::text, d.workflow_key,
			       d.status, d.review_status, d.initial_context,
			       count(c.*) FILTER (WHERE c.status <> 'APPLIED')
			FROM demonstrations d
			LEFT JOIN demonstration_capture_deliveries c
			  ON c.demonstration_id = d.id
			WHERE d.id = $1::uuid AND d.organization_id = $2::uuid
			  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR d.site_id = ANY($3::uuid[]))
			GROUP BY d.id
		`, demonstrationID, principal.OrganizationID, principal.SiteIDs).Scan(
			&organizationID, &siteID, &workflowKey, &status, &reviewStatus,
			&initialContext, &incomplete,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, learningScope{}, workflowapp.ErrNotFound
		}
		if err != nil {
			return nil, learningScope{}, err
		}
		if status != "completed" || reviewStatus != "APPROVED" || incomplete != 0 {
			return nil, learningScope{}, workflowapp.ErrConflict
		}
		demo, exists, err := store.Get(ctx, demonstrationID)
		if err != nil {
			return nil, learningScope{}, err
		}
		if !exists {
			return nil, learningScope{}, workflowapp.ErrNotFound
		}
		if err := g.applyLearningRedactions(ctx, &demo); err != nil {
			return nil, learningScope{}, err
		}
		var initial map[string]interface{}
		if err := json.Unmarshal(initialContext, &initial); err != nil {
			return nil, learningScope{}, err
		}
		current := learningScope{
			WorkflowKey: workflowKey,
			SiteID:      siteID,
			AssetClass:  workflowStringValue(initial["asset_class"]),
			AssetID:     workflowStringValue(initial["asset_id"]),
		}
		if current.AssetClass == "" {
			current.AssetClass = "SHIFT_HANDOVER"
		}
		if current.AssetID != "" {
			_ = g.Pool.QueryRow(ctx, `
				SELECT coalesce(manufacturer, ''), coalesce(model, '')
				FROM assets
				WHERE id = $1::uuid AND organization_id = $2::uuid
			`, current.AssetID, organizationID).Scan(
				&current.Manufacturer, &current.Model,
			)
		}
		if len(result) == 0 {
			scope = current
		} else if scope.WorkflowKey != current.WorkflowKey ||
			scope.SiteID != current.SiteID ||
			scope.AssetClass != current.AssetClass {
			return nil, learningScope{}, workflowapp.ErrInvalid
		}
		result = append(result, demo)
	}
	return result, scope, nil
}

func (g WorkflowLearningGateway) applyLearningRedactions(
	ctx context.Context,
	demonstration *sdkobservation.Demonstration,
) error {
	rows, err := g.Pool.Query(ctx, `
		SELECT event_id::text, json_path, action
		FROM demonstration_event_redactions
		WHERE demonstration_id = $1::uuid
		ORDER BY requested_at, id
	`, demonstration.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type rule struct {
		eventID string
		path    string
		action  string
	}
	var rules []rule
	for rows.Next() {
		var value rule
		if err := rows.Scan(&value.eventID, &value.path, &value.action); err != nil {
			return err
		}
		rules = append(rules, value)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for eventIndex := range demonstration.Trace.Events {
		event := &demonstration.Trace.Events[eventIndex]
		for _, rule := range rules {
			if rule.eventID != event.ID {
				continue
			}
			parts := strings.Split(rule.path, ".")
			if len(parts) < 2 {
				return errors.New("invalid stored demonstration redaction path")
			}
			var target map[string]interface{}
			switch parts[0] {
			case "input":
				target = event.Input
			case "output":
				target = event.Output
			case "context":
				target = event.Context
			case "decision":
				target = event.Decision
			case "result":
				target = event.Result
			default:
				return errors.New("unsupported stored demonstration redaction root")
			}
			applyMapRedaction(target, parts[1:], rule.action)
		}
	}
	return nil
}

func (g WorkflowLearningGateway) workflowIdentity(
	ctx context.Context,
	organizationID, workflowKey string,
) (string, error) {
	var workflowID string
	err := g.Pool.QueryRow(ctx, `
		SELECT id::text
		FROM maintenance_workflows
		WHERE organization_id = $1::uuid AND workflow_key = $2
	`, organizationID, workflowKey).Scan(&workflowID)
	if err == nil {
		return workflowID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return g.newID(), nil
}

func (g WorkflowLearningGateway) captureImprovementCandidates(
	ctx context.Context,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	demonstrations []sdkobservation.Demonstration,
) error {
	for _, demonstration := range demonstrations {
		for _, event := range demonstration.Trace.Events {
			if event.Action != "copilot.recommendation.corrected" ||
				event.CorrectionOf == "" {
				continue
			}
			value, _ := event.Decision["value"].(map[string]interface{})
			correctedAction := workflowStringValue(value["correction"])
			if correctedAction == "" {
				return errors.New("correction event has no corrected action")
			}
			contextPayload, err := json.Marshal(event.Context)
			if err != nil {
				return err
			}
			outcomePayload, err := json.Marshal(
				demonstration.Trace.FinalResult,
			)
			if err != nil {
				return err
			}
			_, err = g.Pool.Exec(ctx, `
				INSERT INTO workflow_improvement_candidates (
					id, organization_id, workflow_id, workflow_version,
					demonstration_id, correction_event_id, corrected_event_id,
					corrected_action, reason, context_snapshot,
					outcome_snapshot, status, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4,
					$5::uuid, $6::uuid, $7::uuid,
					$8, nullif($9, ''), $10::jsonb,
					$11::jsonb, 'OPEN', $12
				)
				ON CONFLICT DO NOTHING
			`, g.newID(), principal.OrganizationID, workflowID, version,
				demonstration.ID, event.ID, event.CorrectionOf,
				correctedAction, workflowStringValue(value["reason"]),
				contextPayload, outcomePayload, g.now())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func workflowEvaluation(
	candidate sdkworkflow.Version,
	catalog interface {
		Describe(context.Context, string) (sdkcore.ToolDescriptor, bool, error)
	},
) (sdkevaluation.Suite, error) {
	fixtures := make(map[string]sdkevaluation.ToolFixture)
	var expectedCalls []sdkevaluation.ExpectedToolCall
	stepStatuses := make(map[string]sdkworkflow.StepStatus)
	for _, step := range candidate.Steps {
		stepStatuses[step.ID] = sdkworkflow.StepCompleted
		if step.Kind != sdkworkflow.StepTool || step.Tool == nil {
			continue
		}
		descriptor, exists, err := catalog.Describe(
			context.Background(), step.Tool.Name,
		)
		if err != nil {
			return sdkevaluation.Suite{}, err
		}
		if !exists {
			return sdkevaluation.Suite{}, fmt.Errorf(
				"evaluation references unknown tool %q", step.Tool.Name,
			)
		}
		arguments := make(map[string]interface{}, len(step.Tool.Arguments))
		for name, value := range step.Tool.Arguments {
			if value.Ref != "" {
				return sdkevaluation.Suite{}, fmt.Errorf(
					"maintenance evaluation does not permit unresolved argument %q",
					name,
				)
			}
			arguments[name] = value.Literal
		}
		fixture := fixtures[step.Tool.Name]
		fixture.Descriptor = descriptor
		fixture.Responses = append(
			fixture.Responses,
			sdkevaluation.ToolResponse{Output: map[string]interface{}{
				"guidance_presented": true,
				"semantic_action":    arguments["semantic_action"],
			}},
		)
		fixtures[step.Tool.Name] = fixture
		expectedCalls = append(expectedCalls, sdkevaluation.ExpectedToolCall{
			Name: step.Tool.Name, Arguments: arguments,
		})
	}
	return sdkevaluation.Suite{
		Name: workflowEvaluationSuite,
		Scenarios: []sdkevaluation.Scenario{{
			ID:    "compiled-guidance-contract",
			Tools: fixtures,
			Expected: sdkevaluation.ExpectedOutcome{
				Status:       sdkworkflow.ExecutionCompleted,
				ToolCalls:    expectedCalls,
				StepStatuses: stepStatuses,
			},
		}},
		Gates: []sdkevaluation.Gate{
			{
				Metric:   sdkevaluation.MetricTaskSuccessRate,
				Operator: sdkevaluation.GateAtLeast,
				Value:    1,
			},
			{
				Metric:   sdkevaluation.MetricUnsafeActionRate,
				Operator: sdkevaluation.GateAtMost,
				Value:    0,
			},
		},
	}, nil
}

func (g WorkflowLearningGateway) validateApplicability(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	value workflowapp.Applicability,
) error {
	if value.SiteID != "" &&
		!principal.CanAccessSite(principal.OrganizationID, value.SiteID) {
		return workflowapp.ErrForbidden
	}
	if value.AssetID != "" {
		var organizationID, siteID, assetClass string
		err := tx.QueryRow(ctx, `
			SELECT organization_id::text, site_id::text, asset_class
			FROM assets
			WHERE id = $1::uuid AND organization_id = $2::uuid
		`, value.AssetID, principal.OrganizationID).Scan(
			&organizationID, &siteID, &assetClass,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return workflowapp.ErrInvalid
		}
		if err != nil {
			return err
		}
		if !principal.CanAccessSite(organizationID, siteID) {
			return workflowapp.ErrForbidden
		}
		if value.SiteID != "" && value.SiteID != siteID ||
			value.AssetClass != "" && value.AssetClass != assetClass {
			return workflowapp.ErrInvalid
		}
	}
	if value.SiteID == "" && value.AssetID == "" &&
		value.AssetClass == "" {
		return workflowapp.ErrInvalid
	}
	return nil
}

func (g WorkflowLearningGateway) insertApplicability(
	ctx context.Context,
	tx pgx.Tx,
	principal identitydomain.Principal,
	workflowID string,
	version int,
	value workflowapp.Applicability,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO workflow_applicability (
			id, organization_id, workflow_id, workflow_version,
			site_id, asset_id, asset_class, manufacturer, model,
			process_service, operating_condition, validation_status,
			approved_by, approved_at, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4,
			nullif($5, '')::uuid, nullif($6, '')::uuid, nullif($7, ''),
			nullif($8, ''), nullif($9, ''), nullif($10, ''),
			nullif($11, ''), $12, $13::uuid, $14, $14
		)
	`, g.newID(), principal.OrganizationID, workflowID, version,
		value.SiteID, value.AssetID, strings.TrimSpace(value.AssetClass),
		strings.TrimSpace(value.Manufacturer), strings.TrimSpace(value.Model),
		strings.TrimSpace(value.ProcessService),
		strings.TrimSpace(value.OperatingCondition),
		value.ValidationStatus, principal.ID, g.now())
	return err
}

func (g WorkflowLearningGateway) appendWorkflowAudit(
	ctx context.Context,
	principal identitydomain.Principal,
	siteID, workflowID, action, reason string,
	attributes map[string]interface{},
) error {
	_, err := database.InTx(
		ctx, g.Pool, pgx.TxOptions{},
		func(tx pgx.Tx) (struct{}, error) {
			err := g.Audit.Append(ctx, tx, audit.Event{
				ID: g.newID(), OrganizationID: principal.OrganizationID,
				SiteID: siteID, ActorID: principal.ID, Action: action,
				EntityKind: "workflow", EntityID: workflowID,
				WorkflowID: workflowID, Reason: reason,
				Attributes: attributes, OccurredAt: g.now(),
			})
			return struct{}{}, err
		},
	)
	return err
}

func applyMapRedaction(
	target map[string]interface{},
	path []string,
	action string,
) {
	if target == nil || len(path) == 0 {
		return
	}
	current := target
	for _, part := range path[:len(path)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return
		}
		current = next
	}
	key := path[len(path)-1]
	if _, exists := current[key]; !exists {
		return
	}
	if action == "DROP" {
		delete(current, key)
	} else {
		current[key] = "[REDACTED]"
	}
}

func mapWorkflowSteps(steps []sdkworkflow.Step) []workflowapp.WorkflowStep {
	result := make([]workflowapp.WorkflowStep, 0, len(steps))
	for _, step := range steps {
		value := workflowapp.WorkflowStep{
			ID: step.ID, Name: step.Name, Kind: string(step.Kind),
		}
		if step.Tool != nil {
			value.ToolName = step.Tool.Name
			value.Arguments = make(map[string]interface{}, len(step.Tool.Arguments))
			for name, argument := range step.Tool.Arguments {
				if argument.Ref != "" {
					value.Arguments[name] = map[string]interface{}{
						"ref": argument.Ref,
					}
				} else {
					value.Arguments[name] = argument.Literal
				}
			}
		}
		for _, evidence := range step.Evidence {
			value.Evidence = append(value.Evidence, workflowapp.Evidence{
				DemonstrationID: evidence.DemonstrationID,
				EventIDs:        evidence.EventIDs,
			})
		}
		result = append(result, value)
	}
	return result
}

func (g WorkflowLearningGateway) now() time.Time {
	if g.Clock == nil {
		return time.Now().UTC()
	}
	return g.Clock.Now().UTC()
}

func (g WorkflowLearningGateway) newID() string {
	if g.IDs == nil {
		return uuid.NewString()
	}
	return g.IDs.New()
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func workflowStringValue(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func decodeMap(value interface{}, destination *map[string]interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, destination)
}

func mapWorkflowError(err error) error {
	var sdkError *sdkcore.SkawldError
	if !errors.As(err, &sdkError) {
		return err
	}
	switch sdkError.Kind {
	case sdkcore.ErrorNotFound:
		return workflowapp.ErrNotFound
	case sdkcore.ErrorConflict:
		return workflowapp.ErrConflict
	case sdkcore.ErrorPermissionDenied:
		return workflowapp.ErrForbidden
	case sdkcore.ErrorValidation, sdkcore.ErrorConfig:
		return fmt.Errorf("%w: %s", workflowapp.ErrInvalid, sdkError.Message)
	case sdkcore.ErrorPolicy:
		return fmt.Errorf("%w: %s", workflowapp.ErrPolicy, sdkError.Message)
	default:
		return err
	}
}
