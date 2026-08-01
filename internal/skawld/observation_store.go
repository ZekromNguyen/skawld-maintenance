package skawld

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type demonstrationStartMetadata struct {
	SiteID      string
	SubjectKind string
	SubjectID   string
	CreatedBy   string
}

type demonstrationStartKey struct{}

func withDemonstrationStart(
	ctx context.Context,
	metadata demonstrationStartMetadata,
) context.Context {
	return context.WithValue(ctx, demonstrationStartKey{}, metadata)
}

// ObservationStore is the only PostgreSQL implementation of the SDK
// observation.Store. SDK persistence remains confined to internal/skawld.
type ObservationStore struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
}

func (s ObservationStore) Create(
	ctx context.Context,
	demo sdkobservation.Demonstration,
) error {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" ||
		demo.Principal.TenantID != principal.TenantID {
		return sdkcore.NewPermissionError("demonstration tenant does not match authenticated context")
	}
	metadata, ok := ctx.Value(demonstrationStartKey{}).(demonstrationStartMetadata)
	if !ok || metadata.SiteID == "" || metadata.SubjectID == "" ||
		metadata.CreatedBy == "" {
		return sdkcore.NewConfigError("demonstration subject metadata is required")
	}
	principalJSON, err := json.Marshal(demo.Principal)
	if err != nil {
		return err
	}
	contextJSON, err := json.Marshal(demo.Trace.InitialContext)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO demonstrations (
			id, organization_id, site_id, subject_kind, subject_id,
			workflow_key, sdk_schema_version, sdk_session_id,
			principal_snapshot, status, initial_context, started_at,
			created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5::uuid,
			$6, $7, $8::uuid, $9::jsonb, $10, $11::jsonb, $12,
			$13::uuid, $14, $14
		)
	`, demo.ID, demo.Principal.TenantID, metadata.SiteID,
		metadata.SubjectKind, metadata.SubjectID, demo.WorkflowKey,
		demo.Trace.SchemaVersion, demo.Trace.SessionID, principalJSON,
		demo.Status, contextJSON, demo.StartedAt.UTC(), metadata.CreatedBy, now)
	if err != nil {
		return mapObservationStoreError(err)
	}
	return nil
}

func (s ObservationStore) Append(
	ctx context.Context,
	demonstrationID string,
	event sdkobservation.Event,
) error {
	return inObservationTx(ctx, s.Pool, func(tx pgx.Tx) error {
		demo, ok, err := loadSDKDemonstration(ctx, tx, demonstrationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return &sdkcore.SkawldError{
				Kind: sdkcore.ErrorNotFound, Message: "demonstration not found",
			}
		}
		if err := authorizeSDKDemonstration(ctx, demo); err != nil {
			return err
		}
		if demo.Status != sdkobservation.DemonstrationRecording {
			return &sdkcore.SkawldError{
				Kind: sdkcore.ErrorConflict, Message: "demonstration is not recording",
			}
		}
		if err := sdkobservation.ValidateAppend(demo.Trace, event); err != nil {
			return err
		}
		principalJSON, err := json.Marshal(event.Principal)
		if err != nil {
			return err
		}
		entity, _ := jsonOrNil(event.Entity)
		input, _ := jsonOrNil(event.Input)
		output, _ := jsonOrNil(event.Output)
		contextValue, _ := jsonOrNil(event.Context)
		decision, _ := jsonOrNil(event.Decision)
		result, _ := jsonOrNil(event.Result)
		provenance := []byte(`{"capture":"sdk_observation_v1"}`)
		domainEventID := ""
		if event.Context != nil {
			if value, ok := event.Context["domain_event_id"].(string); ok {
				domainEventID = value
			}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO demonstration_events (
				id, demonstration_id, ordinal, schema_version, session_id,
				principal_snapshot, occurred_at, source, trust, sensitivity,
				application, action, intent, entity, input_value, output_value,
				context_value, decision_value, result_value, error_value,
				correction_of, approval_id, domain_event_id, provenance, created_at
			) VALUES (
				$1::uuid, $2::uuid, $3, $4, $5::uuid, $6::jsonb, $7, $8, $9, $10,
				nullif($11, ''), $12, nullif($13, ''), $14::jsonb, $15::jsonb,
				$16::jsonb, $17::jsonb, $18::jsonb, $19::jsonb, nullif($20, ''),
				nullif($21, '')::uuid, nullif($22, ''), nullif($23, '')::uuid,
				$24::jsonb, $25
			)
		`, event.ID, demonstrationID, len(demo.Trace.Events)+1,
			event.SchemaVersion, event.SessionID, principalJSON,
			event.Timestamp.UTC(), event.Source, event.Trust, event.Sensitivity,
			event.Application, event.Action, event.Intent, entity, input, output,
			contextValue, decision, result, event.Error, event.CorrectionOf,
			event.ApprovalID, domainEventID, provenance, s.now())
		return mapObservationStoreError(err)
	})
}

func (s ObservationStore) Complete(
	ctx context.Context,
	demonstrationID string,
	result map[string]interface{},
) (sdkobservation.Demonstration, error) {
	var completed sdkobservation.Demonstration
	err := inObservationTx(ctx, s.Pool, func(tx pgx.Tx) error {
		demo, ok, err := loadSDKDemonstration(ctx, tx, demonstrationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return &sdkcore.SkawldError{
				Kind: sdkcore.ErrorNotFound, Message: "demonstration not found",
			}
		}
		if err := authorizeSDKDemonstration(ctx, demo); err != nil {
			return err
		}
		if demo.Status != sdkobservation.DemonstrationRecording {
			return &sdkcore.SkawldError{
				Kind: sdkcore.ErrorConflict, Message: "demonstration is not recording",
			}
		}
		demo.Status = sdkobservation.DemonstrationCompleted
		demo.CompletedAt = s.now()
		demo.Trace.FinalResult = result
		if err := demo.Trace.Validate(); err != nil {
			return fmt.Errorf("complete demonstration: %w", err)
		}
		finalJSON, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE demonstrations
			SET status = 'completed', final_result = $2::jsonb,
			    completed_at = $3, updated_at = $3
			WHERE id = $1::uuid
		`, demonstrationID, finalJSON, demo.CompletedAt)
		if err != nil {
			return err
		}
		completed = demo
		return nil
	})
	return completed, err
}

func (s ObservationStore) Get(
	ctx context.Context,
	demonstrationID string,
) (sdkobservation.Demonstration, bool, error) {
	demo, ok, err := loadSDKDemonstration(ctx, s.Pool, demonstrationID, false)
	if err != nil || !ok {
		return demo, ok, err
	}
	if err := authorizeSDKDemonstration(ctx, demo); err != nil {
		return sdkobservation.Demonstration{}, false, err
	}
	return demo, true, nil
}

func (s ObservationStore) List(
	ctx context.Context,
	workflowKey string,
) ([]sdkobservation.Demonstration, error) {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return nil, sdkcore.NewPermissionError("authenticated SDK principal is required")
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text
		FROM demonstrations
		WHERE organization_id = $1::uuid
		  AND ($2 = '' OR workflow_key = $2)
		ORDER BY started_at DESC
	`, principal.TenantID, workflowKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]sdkobservation.Demonstration, 0, len(ids))
	for _, id := range ids {
		demo, ok, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, demo)
		}
	}
	return result, nil
}

type observationQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadSDKDemonstration(
	ctx context.Context,
	query observationQuerier,
	id string,
	forUpdate bool,
) (sdkobservation.Demonstration, bool, error) {
	statement := `
		SELECT id::text, workflow_key, principal_snapshot, status, started_at,
		       completed_at, sdk_schema_version, sdk_session_id::text,
		       initial_context, final_result
		FROM demonstrations
		WHERE id = $1::uuid`
	if forUpdate {
		statement += " FOR UPDATE"
	}
	var demo sdkobservation.Demonstration
	var principalJSON, initialJSON, finalJSON []byte
	var completedAt *time.Time
	err := query.QueryRow(ctx, statement, id).Scan(
		&demo.ID, &demo.WorkflowKey, &principalJSON, &demo.Status,
		&demo.StartedAt, &completedAt, &demo.Trace.SchemaVersion,
		&demo.Trace.SessionID, &initialJSON, &finalJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sdkobservation.Demonstration{}, false, nil
	}
	if err != nil {
		return sdkobservation.Demonstration{}, false, err
	}
	if completedAt != nil {
		demo.CompletedAt = completedAt.UTC()
	}
	if err := json.Unmarshal(principalJSON, &demo.Principal); err != nil {
		return sdkobservation.Demonstration{}, false, err
	}
	if len(initialJSON) > 0 {
		if err := json.Unmarshal(initialJSON, &demo.Trace.InitialContext); err != nil {
			return sdkobservation.Demonstration{}, false, err
		}
	}
	if len(finalJSON) > 0 {
		if err := json.Unmarshal(finalJSON, &demo.Trace.FinalResult); err != nil {
			return sdkobservation.Demonstration{}, false, err
		}
	}
	rows, err := query.Query(ctx, `
		SELECT id::text, schema_version, session_id::text, principal_snapshot,
		       occurred_at, source, trust, sensitivity, coalesce(application, ''),
		       action, coalesce(intent, ''), entity, input_value, output_value,
		       context_value, decision_value, result_value,
		       coalesce(error_value, ''), coalesce(correction_of::text, ''),
		       coalesce(approval_id, '')
		FROM demonstration_events
		WHERE demonstration_id = $1::uuid
		ORDER BY ordinal
	`, id)
	if err != nil {
		return sdkobservation.Demonstration{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var event sdkobservation.Event
		var eventPrincipal, entity, input, output, contextValue, decision, result []byte
		if err := rows.Scan(
			&event.ID, &event.SchemaVersion, &event.SessionID, &eventPrincipal,
			&event.Timestamp, &event.Source, &event.Trust, &event.Sensitivity,
			&event.Application, &event.Action, &event.Intent, &entity, &input,
			&output, &contextValue, &decision, &result, &event.Error,
			&event.CorrectionOf, &event.ApprovalID,
		); err != nil {
			return sdkobservation.Demonstration{}, false, err
		}
		if err := json.Unmarshal(eventPrincipal, &event.Principal); err != nil {
			return sdkobservation.Demonstration{}, false, err
		}
		if err := decodeOptional(entity, &event.Entity); err != nil {
			return sdkobservation.Demonstration{}, false, err
		}
		for _, value := range []struct {
			raw    []byte
			target *map[string]interface{}
		}{
			{input, &event.Input},
			{output, &event.Output},
			{contextValue, &event.Context},
			{decision, &event.Decision},
			{result, &event.Result},
		} {
			if len(value.raw) > 0 && string(value.raw) != "null" {
				if err := json.Unmarshal(value.raw, value.target); err != nil {
					return sdkobservation.Demonstration{}, false, err
				}
			}
		}
		demo.Trace.Events = append(demo.Trace.Events, event)
	}
	if err := rows.Err(); err != nil {
		return sdkobservation.Demonstration{}, false, err
	}
	return demo, true, nil
}

func authorizeSDKDemonstration(
	ctx context.Context,
	demo sdkobservation.Demonstration,
) error {
	principal, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" ||
		demo.Principal.TenantID != principal.TenantID {
		return sdkcore.NewPermissionError("demonstration belongs to another tenant")
	}
	return nil
}

func (s ObservationStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func inObservationTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	fn func(pgx.Tx) error,
) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func jsonOrNil(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func decodeOptional(raw []byte, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func mapObservationStoreError(err error) error {
	if err == nil {
		return nil
	}
	// PostgreSQL constraint details remain internal; callers get a stable SDK
	// conflict while retries can still inspect persistence logs.
	if strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") {
		return &sdkcore.SkawldError{
			Kind: sdkcore.ErrorConflict, Message: "observation event already exists",
		}
	}
	return err
}

var _ sdkobservation.Store = ObservationStore{}
