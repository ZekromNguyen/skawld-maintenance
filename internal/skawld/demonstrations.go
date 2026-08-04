package skawld

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DemonstrationGateway struct {
	Pool  *pgxpool.Pool
	IDs   id.Generator
	Clock clock.Clock
	Audit audit.Sink
}

type subjectContext struct {
	OrganizationID string
	SiteID         string
	WorkflowKey    string
	Snapshot       map[string]interface{}
}

func (g DemonstrationGateway) Start(
	ctx context.Context,
	principal identitydomain.Principal,
	command demonstrationapp.Start,
) (demonstrationapp.Demonstration, error) {
	subject, err := g.loadSubject(ctx, principal, command)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	sdkContext, sdkPrincipal, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	sdkContext = withDemonstrationStart(sdkContext, demonstrationStartMetadata{
		SiteID: subject.SiteID, SubjectKind: string(command.SubjectKind),
		SubjectID: command.SubjectID, CreatedBy: principal.ID,
	})
	recorder, err := g.recorder()
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	initial := subject.Snapshot
	initial["subject_kind"] = string(command.SubjectKind)
	initial["subject_id"] = command.SubjectID
	initial["capture_policy"] = "maintenance.semantic.v1"
	demo, err := recorder.Start(
		sdkContext, subject.WorkflowKey, sdkPrincipal, initial,
	)
	if err != nil {
		return demonstrationapp.Demonstration{}, mapDemonstrationError(err)
	}
	_, err = recorder.Capture(sdkContext, demo.ID, sdkobservation.Event{
		ID: demo.ID, Principal: sdkPrincipal, Timestamp: demo.StartedAt,
		Source:      sdkobservation.SourceAPI,
		Trust:       sdkobservation.TrustHumanInstruction,
		Sensitivity: sdkobservation.SensitivityInternal,
		Application: "skawld-maintenance",
		Action:      "demonstration.started",
		Intent:      "capture semantic maintenance work",
		Entity: &sdkobservation.Entity{
			Type: strings.ToLower(string(command.SubjectKind)),
			ID:   command.SubjectID,
		},
		Context: map[string]interface{}{
			"subject_kind": string(command.SubjectKind),
			"subject_id":   command.SubjectID,
		},
	})
	if err != nil {
		return demonstrationapp.Demonstration{}, mapDemonstrationError(err)
	}
	if err := g.appendAudit(ctx, principal, demo.ID, subject, "demonstration.started", nil); err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	return g.Get(ctx, principal, demo.ID)
}

func (g DemonstrationGateway) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID string,
) (demonstrationapp.Demonstration, error) {
	sdkContext, _, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	store := ObservationStore{Pool: g.Pool, Clock: g.Clock}
	demo, ok, err := store.Get(sdkContext, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, mapDemonstrationError(err)
	}
	if !ok {
		return demonstrationapp.Demonstration{}, demonstrationapp.ErrNotFound
	}
	var result demonstrationapp.Demonstration
	var completedAt *time.Time
	err = g.Pool.QueryRow(ctx, `
		SELECT organization_id::text, site_id::text, subject_kind,
		       subject_id::text, review_status, created_by::text, completed_at
		FROM demonstrations
		WHERE id = $1::uuid AND organization_id = $2::uuid
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
	`, demonstrationID, principal.OrganizationID, principal.SiteIDs).Scan(
		&result.OrganizationID, &result.SiteID, &result.SubjectKind,
		&result.SubjectID, &result.ReviewStatus, &result.CreatedBy, &completedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return demonstrationapp.Demonstration{}, demonstrationapp.ErrNotFound
	}
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	result.ID, result.WorkflowKey = demo.ID, demo.WorkflowKey
	result.SchemaVersion, result.SessionID = demo.Trace.SchemaVersion, demo.Trace.SessionID
	result.Status = string(demo.Status)
	result.InitialContext, result.FinalResult = demo.Trace.InitialContext, demo.Trace.FinalResult
	result.StartedAt, result.CompletedAt = demo.StartedAt, completedAt
	eventMetadata, err := g.loadEventMetadata(ctx, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	redactions, err := g.loadRedactions(ctx, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	for index, event := range demo.Trace.Events {
		mapped := mapObservationEvent(event)
		if metadata, exists := eventMetadata[event.ID]; exists {
			mapped.Ordinal = metadata.Ordinal
			mapped.DomainEventID = metadata.DomainEventID
			mapped.Provenance = metadata.Provenance
		} else {
			mapped.Ordinal = index + 1
		}
		mapped.Redactions = redactions[event.ID]
		applyReviewRedactions(&mapped)
		result.Events = append(result.Events, mapped)
	}
	result.Reviews, err = g.loadReviews(ctx, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	result.Capture, err = g.captureHealth(ctx, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	return result, nil
}

func (g DemonstrationGateway) List(
	ctx context.Context,
	principal identitydomain.Principal,
	siteID string,
) ([]demonstrationapp.Demonstration, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text
		FROM demonstrations
		WHERE organization_id = $1::uuid
		  AND ($2 = '' OR site_id::text = $2)
		  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
		ORDER BY started_at DESC
		LIMIT 200
	`, principal.OrganizationID, siteID, principal.SiteIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		ids = append(ids, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	output := make([]demonstrationapp.Demonstration, 0, len(ids))
	for _, value := range ids {
		demo, err := g.Get(ctx, principal, value)
		if err != nil {
			return nil, err
		}
		output = append(output, demo)
	}
	return output, nil
}

func (g DemonstrationGateway) Complete(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID string,
	command demonstrationapp.Complete,
) (demonstrationapp.Demonstration, error) {
	processor := CaptureProcessor{
		Pool: g.Pool, Clock: g.Clock,
		Store: ObservationStore{Pool: g.Pool, Clock: g.Clock},
	}
	if err := processor.ProcessDemonstration(ctx, demonstrationID, 500); err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	health, err := g.captureHealth(ctx, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	if health.Pending > 0 || health.Processing > 0 || health.Failed > 0 {
		return demonstrationapp.Demonstration{}, demonstrationapp.ErrCapturePending
	}
	sdkContext, _, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	recorder, err := g.recorder()
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	if _, err := recorder.Complete(sdkContext, demonstrationID, map[string]interface{}{
		"outcome": command.Outcome,
	}); err != nil {
		return demonstrationapp.Demonstration{}, mapDemonstrationError(err)
	}
	demo, err := g.Get(ctx, principal, demonstrationID)
	if err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	subject := subjectContext{
		OrganizationID: demo.OrganizationID, SiteID: demo.SiteID,
	}
	if err := g.appendAudit(
		ctx, principal, demo.ID, subject, "demonstration.completed",
		map[string]any{"outcome": command.Outcome, "event_count": len(demo.Events)},
	); err != nil {
		return demonstrationapp.Demonstration{}, err
	}
	return demo, nil
}

func (g DemonstrationGateway) RecordEvidenceView(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID string,
	command demonstrationapp.RecordEvidenceView,
) (demonstrationapp.Event, error) {
	if err := g.validateEvidence(ctx, principal, command.EvidenceID); err != nil {
		return demonstrationapp.Event{}, err
	}
	processor := CaptureProcessor{
		Pool: g.Pool, Clock: g.Clock,
		Store: ObservationStore{Pool: g.Pool, Clock: g.Clock},
	}
	if err := processor.ProcessDemonstration(ctx, demonstrationID, 500); err != nil {
		return demonstrationapp.Event{}, err
	}
	sdkContext, sdkPrincipal, err := AuthenticatedContext(ctx, principal)
	if err != nil {
		return demonstrationapp.Event{}, err
	}
	recorder, err := g.recorder()
	if err != nil {
		return demonstrationapp.Event{}, err
	}
	event, err := recorder.Capture(sdkContext, demonstrationID, sdkobservation.Event{
		Principal: sdkPrincipal, Source: sdkobservation.SourceBrowser,
		Trust:       sdkobservation.TrustHumanInstruction,
		Sensitivity: sdkobservation.SensitivityInternal,
		Application: "skawld-maintenance", Action: "evidence.viewed",
		Intent:  command.Intent,
		Entity:  &sdkobservation.Entity{Type: "evidence", ID: command.EvidenceID},
		Context: map[string]interface{}{"evidence_id": command.EvidenceID},
	})
	if err != nil {
		return demonstrationapp.Event{}, mapDemonstrationError(err)
	}
	return mapObservationEvent(event), nil
}

func (g DemonstrationGateway) Review(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID string,
	command demonstrationapp.Review,
) (demonstrationapp.ReviewRecord, error) {
	value := demonstrationapp.ReviewRecord{
		ID: g.IDs.New(), Decision: command.Decision, Reason: command.Reason,
		ReviewedBy: principal.ID, ReviewedAt: g.now(),
	}
	_, err := database.InTx(ctx, g.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		tag, err := tx.Exec(ctx, `
			UPDATE demonstrations
			SET review_status = $3, updated_at = $4
			WHERE id = $1::uuid AND organization_id = $2::uuid
			  AND status = 'completed'
		`, demonstrationID, principal.OrganizationID, command.Decision, value.ReviewedAt)
		if err != nil {
			return struct{}{}, err
		}
		if tag.RowsAffected() != 1 {
			return struct{}{}, demonstrationapp.ErrConflict
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO demonstration_reviews (
				id, demonstration_id, decision, reason, reviewed_by, reviewed_at
			) VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, $6)
		`, value.ID, demonstrationID, value.Decision, value.Reason,
			value.ReviewedBy, value.ReviewedAt)
		if err != nil {
			return struct{}{}, err
		}
		if err := g.Audit.Append(ctx, tx, audit.Event{
			ID: g.IDs.New(), OrganizationID: principal.OrganizationID,
			ActorID: principal.ID, Action: "demonstration.reviewed",
			EntityKind: "demonstration", EntityID: demonstrationID,
			Reason: command.Reason, After: value, OccurredAt: value.ReviewedAt,
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return value, err
}

func (g DemonstrationGateway) RedactEvent(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID, eventID string,
	command demonstrationapp.RedactEvent,
) (demonstrationapp.Redaction, error) {
	value := demonstrationapp.Redaction{
		ID: g.IDs.New(), EventID: eventID, JSONPath: command.JSONPath,
		Action: command.Action, Reason: command.Reason,
		RequestedBy: principal.ID, RequestedAt: g.now(),
	}
	_, err := database.InTx(ctx, g.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		var exists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM demonstration_events e
				JOIN demonstrations d ON d.id = e.demonstration_id
				WHERE e.id = $1::uuid AND e.demonstration_id = $2::uuid
				  AND d.organization_id = $3::uuid
			)
		`, eventID, demonstrationID, principal.OrganizationID).Scan(&exists)
		if err != nil {
			return struct{}{}, err
		}
		if !exists {
			return struct{}{}, demonstrationapp.ErrNotFound
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO demonstration_event_redactions (
				id, demonstration_id, event_id, json_path, action, reason,
				requested_by, requested_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7::uuid, $8)
			ON CONFLICT (event_id, json_path) DO UPDATE
			SET action = EXCLUDED.action, reason = EXCLUDED.reason,
			    requested_by = EXCLUDED.requested_by,
			    requested_at = EXCLUDED.requested_at
			RETURNING id::text
		`, value.ID, demonstrationID, eventID, value.JSONPath, value.Action,
			value.Reason, value.RequestedBy, value.RequestedAt).Scan(&value.ID)
		if err != nil {
			return struct{}{}, err
		}
		if err := g.Audit.Append(ctx, tx, audit.Event{
			ID: g.IDs.New(), OrganizationID: principal.OrganizationID,
			ActorID: principal.ID, Action: "demonstration.event.redacted",
			EntityKind: "demonstration_event", EntityID: eventID,
			Reason: command.Reason,
			After: map[string]string{
				"json_path": command.JSONPath, "action": command.Action,
			},
			OccurredAt: value.RequestedAt,
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return value, err
}

func (g DemonstrationGateway) recorder() (*sdkobservation.Recorder, error) {
	redactor, err := sdkobservation.NewRedactor(sdkobservation.RedactorOptions{
		Rules: map[string]sdkobservation.RedactionAction{
			"initial_context.access_token": sdkobservation.RedactDrop,
			"initial_context.password":     sdkobservation.RedactDrop,
			"input.access_token":           sdkobservation.RedactDrop,
			"input.password":               sdkobservation.RedactDrop,
			"input.authorization":          sdkobservation.RedactDrop,
			"output.secret":                sdkobservation.RedactMask,
			"context.authorization":        sdkobservation.RedactDrop,
		},
	})
	if err != nil {
		return nil, err
	}
	return sdkobservation.NewRecorderWithOptions(sdkobservation.RecorderOptions{
		Store:     ObservationStore{Pool: g.Pool, Clock: g.Clock},
		Sanitizer: redactor,
	})
}

func (g DemonstrationGateway) loadSubject(
	ctx context.Context,
	principal identitydomain.Principal,
	command demonstrationapp.Start,
) (subjectContext, error) {
	var result subjectContext
	var snapshot []byte
	switch command.SubjectKind {
	case demonstrationapp.SubjectExecution:
		var assetClass string
		err := g.Pool.QueryRow(ctx, `
			SELECT e.organization_id::text, e.site_id::text, a.asset_class,
			       jsonb_build_object(
			         'execution_id', e.id,
			         'incident_id', e.incident_id,
			         'asset_id', e.asset_id,
			         'asset_tag', a.tag,
			         'asset_class', a.asset_class,
			         'purpose', e.purpose,
			         'state', e.state,
			         'workflow_id', e.workflow_id,
			         'workflow_version', e.workflow_version
			       )
			FROM maintenance_executions e
			JOIN assets a ON a.id = e.asset_id
			WHERE e.id = $1::uuid AND e.organization_id = $2::uuid
			  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR e.site_id = ANY($3::uuid[]))
		`, command.SubjectID, principal.OrganizationID, principal.SiteIDs).Scan(
			&result.OrganizationID, &result.SiteID, &assetClass, &snapshot,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return subjectContext{}, demonstrationapp.ErrNotFound
		}
		if err != nil {
			return subjectContext{}, err
		}
		result.WorkflowKey = "maintenance.execution." + workflowSegment(assetClass)
	case demonstrationapp.SubjectHandover:
		err := g.Pool.QueryRow(ctx, `
			SELECT h.organization_id::text, h.site_id::text,
			       jsonb_build_object(
			         'handover_id', h.id,
			         'shift_start', h.shift_start,
			         'shift_end', h.shift_end,
			         'state', h.state,
			         'version', h.version,
			         'content', h.structured_content,
			         'evidence', h.evidence_snapshot
			       )
			FROM shift_handovers h
			WHERE h.id = $1::uuid AND h.organization_id = $2::uuid
			  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR h.site_id = ANY($3::uuid[]))
		`, command.SubjectID, principal.OrganizationID, principal.SiteIDs).Scan(
			&result.OrganizationID, &result.SiteID, &snapshot,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return subjectContext{}, demonstrationapp.ErrNotFound
		}
		if err != nil {
			return subjectContext{}, err
		}
		result.WorkflowKey = "maintenance.shift_handover"
	default:
		return subjectContext{}, demonstrationapp.ErrInvalid
	}
	if !principal.CanAccessSite(result.OrganizationID, result.SiteID) {
		return subjectContext{}, demonstrationapp.ErrForbidden
	}
	if err := json.Unmarshal(snapshot, &result.Snapshot); err != nil {
		return subjectContext{}, err
	}
	return result, nil
}

func workflowSegment(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func (g DemonstrationGateway) validateEvidence(
	ctx context.Context,
	principal identitydomain.Principal,
	evidenceID string,
) error {
	parts := strings.SplitN(evidenceID, ":", 2)
	if len(parts) != 2 {
		return demonstrationapp.ErrInvalid
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return demonstrationapp.ErrInvalid
	}
	var exists bool
	switch strings.ToLower(parts[0]) {
	case "document_chunk":
		err := g.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM document_chunks c
				JOIN document_revisions r ON r.id = c.revision_id
				JOIN documents d ON d.id = r.document_id
				WHERE c.id = $1::uuid AND d.organization_id = $2::uuid
				  AND r.approval_status = 'APPROVED'
				  AND r.ingestion_state = 'READY'
				  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR d.site_id = ANY($3::uuid[]))
			)
		`, parts[1], principal.OrganizationID, principal.SiteIDs).Scan(&exists)
		if err != nil {
			return err
		}
	case "historical_incident", "incident":
		err := g.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM incidents
				WHERE id = $1::uuid AND organization_id = $2::uuid
				  AND (COALESCE(cardinality($3::uuid[]), 0) = 0 OR site_id = ANY($3::uuid[]))
			)
		`, parts[1], principal.OrganizationID, principal.SiteIDs).Scan(&exists)
		if err != nil {
			return err
		}
	default:
		return demonstrationapp.ErrInvalid
	}
	if !exists {
		return demonstrationapp.ErrNotFound
	}
	return nil
}

type eventMetadata struct {
	Ordinal       int
	DomainEventID string
	Provenance    map[string]any
}

func (g DemonstrationGateway) loadEventMetadata(
	ctx context.Context,
	demonstrationID string,
) (map[string]eventMetadata, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, ordinal, coalesce(domain_event_id::text, ''), provenance
		FROM demonstration_events
		WHERE demonstration_id = $1::uuid
	`, demonstrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]eventMetadata)
	for rows.Next() {
		var eventID string
		var value eventMetadata
		var provenance []byte
		if err := rows.Scan(
			&eventID, &value.Ordinal, &value.DomainEventID, &provenance,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(provenance, &value.Provenance)
		result[eventID] = value
	}
	return result, rows.Err()
}

func (g DemonstrationGateway) loadRedactions(
	ctx context.Context,
	demonstrationID string,
) (map[string][]demonstrationapp.Redaction, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, event_id::text, json_path, action, reason,
		       requested_by::text, requested_at
		FROM demonstration_event_redactions
		WHERE demonstration_id = $1::uuid
		ORDER BY requested_at
	`, demonstrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]demonstrationapp.Redaction)
	for rows.Next() {
		var value demonstrationapp.Redaction
		if err := rows.Scan(
			&value.ID, &value.EventID, &value.JSONPath, &value.Action,
			&value.Reason, &value.RequestedBy, &value.RequestedAt,
		); err != nil {
			return nil, err
		}
		result[value.EventID] = append(result[value.EventID], value)
	}
	return result, rows.Err()
}

func (g DemonstrationGateway) loadReviews(
	ctx context.Context,
	demonstrationID string,
) ([]demonstrationapp.ReviewRecord, error) {
	rows, err := g.Pool.Query(ctx, `
		SELECT id::text, decision, reason, reviewed_by::text, reviewed_at
		FROM demonstration_reviews
		WHERE demonstration_id = $1::uuid
		ORDER BY reviewed_at
	`, demonstrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []demonstrationapp.ReviewRecord
	for rows.Next() {
		var value demonstrationapp.ReviewRecord
		if err := rows.Scan(
			&value.ID, &value.Decision, &value.Reason,
			&value.ReviewedBy, &value.ReviewedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (g DemonstrationGateway) captureHealth(
	ctx context.Context,
	demonstrationID string,
) (demonstrationapp.CaptureHealth, error) {
	var result demonstrationapp.CaptureHealth
	var updatedAt *time.Time
	err := g.Pool.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE status = 'PENDING'),
		  count(*) FILTER (WHERE status = 'PROCESSING'),
		  count(*) FILTER (WHERE status = 'FAILED'),
		  count(*) FILTER (WHERE status = 'APPLIED'),
		  coalesce((
		    SELECT last_error
		    FROM demonstration_capture_deliveries
		    WHERE demonstration_id = $1::uuid AND last_error IS NOT NULL
		    ORDER BY updated_at DESC LIMIT 1
		  ), ''),
		  max(updated_at)
		FROM demonstration_capture_deliveries
		WHERE demonstration_id = $1::uuid
	`, demonstrationID).Scan(
		&result.Pending, &result.Processing, &result.Failed, &result.Applied,
		&result.LastError, &updatedAt,
	)
	if updatedAt != nil {
		result.UpdatedAt = updatedAt.UTC()
	}
	return result, err
}

func (g DemonstrationGateway) appendAudit(
	ctx context.Context,
	principal identitydomain.Principal,
	demonstrationID string,
	subject subjectContext,
	action string,
	after any,
) error {
	_, err := database.InTx(ctx, g.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		err := g.Audit.Append(ctx, tx, audit.Event{
			ID: g.IDs.New(), OrganizationID: subject.OrganizationID,
			SiteID: subject.SiteID, ActorID: principal.ID, Action: action,
			EntityKind: "demonstration", EntityID: demonstrationID,
			After: after, OccurredAt: g.now(),
		})
		return struct{}{}, err
	})
	return err
}

func (g DemonstrationGateway) now() time.Time {
	if g.Clock == nil {
		return time.Now().UTC()
	}
	return g.Clock.Now().UTC()
}

func mapObservationEvent(event sdkobservation.Event) demonstrationapp.Event {
	roles := append([]string(nil), event.Principal.Roles...)
	return demonstrationapp.Event{
		ID: event.ID, SchemaVersion: event.SchemaVersion,
		Timestamp: event.Timestamp, ActorID: event.Principal.ActorID, Roles: roles,
		Source: string(event.Source), Trust: string(event.Trust),
		Sensitivity: string(event.Sensitivity), Application: event.Application,
		Action: event.Action, Intent: event.Intent, Input: event.Input,
		Output: event.Output, Context: event.Context, Decision: event.Decision,
		Result: event.Result, Error: event.Error, CorrectionOf: event.CorrectionOf,
		ApprovalID: event.ApprovalID,
		Entity: func() map[string]any {
			if event.Entity == nil {
				return nil
			}
			return map[string]any{"type": event.Entity.Type, "id": event.Entity.ID}
		}(),
		Provenance: map[string]any{"capture": "sdk_observation_v1"},
	}
}

func applyReviewRedactions(event *demonstrationapp.Event) {
	for _, redaction := range event.Redactions {
		parts := strings.Split(redaction.JSONPath, ".")
		if len(parts) < 2 {
			continue
		}
		var target map[string]any
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
		}
		redactMap(target, parts[1:], redaction.Action)
	}
}

func redactMap(current map[string]any, path []string, action string) {
	if current == nil || len(path) == 0 {
		return
	}
	if len(path) == 1 {
		if action == "DROP" {
			delete(current, path[0])
		} else if _, ok := current[path[0]]; ok {
			current[path[0]] = "[REDACTED]"
		}
		return
	}
	child, ok := current[path[0]].(map[string]any)
	if ok {
		redactMap(child, path[1:], action)
	}
}

func mapDemonstrationError(err error) error {
	var sdkErr *sdkcore.SkawldError
	if errors.As(err, &sdkErr) {
		switch sdkErr.Kind {
		case sdkcore.ErrorPermissionDenied:
			return demonstrationapp.ErrForbidden
		case sdkcore.ErrorNotFound:
			return demonstrationapp.ErrNotFound
		case sdkcore.ErrorConflict:
			return demonstrationapp.ErrConflict
		case sdkcore.ErrorValidation:
			return demonstrationapp.ErrInvalid
		}
	}
	return err
}

var _ demonstrationapp.Gateway = DemonstrationGateway{}
