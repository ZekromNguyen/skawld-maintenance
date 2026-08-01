package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Event struct {
	ID               string
	OrganizationID   string
	SiteID           string
	ActorID          string
	Type             string
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	SubjectKind      string
	SubjectID        string
	Payload          any
	OccurredAt       time.Time
}

// Append writes the durable semantic source event in the same transaction as
// the authoritative domain mutation. Demonstration capture is performed later
// by the worker; a provider or SDK failure therefore cannot roll back the
// maintenance transaction.
func Append(ctx context.Context, tx pgx.Tx, event Event) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal domain event payload: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO domain_events (
			id, organization_id, site_id, actor_id, event_type, aggregate_type,
			aggregate_id, aggregate_version, subject_kind, subject_id, payload,
			occurred_at
		) VALUES (
			$1::uuid, $2::uuid, nullif($3, '')::uuid, nullif($4, '')::uuid,
			$5, $6, $7::uuid, $8, nullif($9, ''), nullif($10, '')::uuid,
			$11::jsonb, $12
		)
	`, event.ID, event.OrganizationID, event.SiteID, event.ActorID, event.Type,
		event.AggregateType, event.AggregateID, event.AggregateVersion,
		event.SubjectKind, event.SubjectID, payload, event.OccurredAt.UTC())
	if err != nil {
		return fmt.Errorf("append domain event: %w", err)
	}
	return nil
}
