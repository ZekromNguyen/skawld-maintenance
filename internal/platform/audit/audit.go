package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Event struct {
	ID             string
	OrganizationID string
	SiteID         string
	ActorID        string
	Action         string
	EntityKind     string
	EntityID       string
	Reason         string
	RequestID      string
	ExecutionID    string
	WorkflowID     string
	ApprovalID     string
	AIInvolvement  any
	Before         any
	After          any
	Attributes     map[string]any
	OccurredAt     time.Time
}

type Sink struct{}

func (Sink) Append(ctx context.Context, tx pgx.Tx, event Event) error {
	ai, err := jsonOrNull(event.AIInvolvement)
	if err != nil {
		return fmt.Errorf("marshal AI involvement: %w", err)
	}
	before, err := jsonOrNull(event.Before)
	if err != nil {
		return fmt.Errorf("marshal before value: %w", err)
	}
	after, err := jsonOrNull(event.After)
	if err != nil {
		return fmt.Errorf("marshal after value: %w", err)
	}
	attributes, err := json.Marshal(event.Attributes)
	if err != nil {
		return fmt.Errorf("marshal audit attributes: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, organization_id, site_id, actor_id, action, entity_kind, entity_id,
			reason, request_id, execution_id, workflow_id, approval_id,
			ai_involvement, before_value, after_value, attributes, occurred_at
		) VALUES (
			$1::uuid, nullif($2, '')::uuid, nullif($3, '')::uuid, nullif($4, '')::uuid,
			$5, $6, $7, nullif($8, ''), nullif($9, ''), nullif($10, ''),
			nullif($11, ''), nullif($12, ''), $13::jsonb, $14::jsonb, $15::jsonb,
			$16::jsonb, $17
		)
	`, event.ID, event.OrganizationID, event.SiteID, event.ActorID,
		event.Action, event.EntityKind, event.EntityID, event.Reason, event.RequestID,
		event.ExecutionID, event.WorkflowID, event.ApprovalID, ai, before, after,
		attributes, event.OccurredAt.UTC())
	if err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

func jsonOrNull(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(value)
}
