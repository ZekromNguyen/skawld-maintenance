package skawld

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkobservation "github.com/ZekromNguyen/skawld-sdk-go/observation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CaptureProcessor struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
	Store ObservationStore
}

type captureDelivery struct {
	DemonstrationID string
	DomainEventID   string
	Attempts        int
}

type sourceEvent struct {
	OrganizationID   string
	ActorID          string
	Type             string
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	SubjectKind      string
	SubjectID        string
	Payload          map[string]interface{}
	OccurredAt       time.Time
}

// ProcessPending is the periodic River worker entrypoint. Individual mapping
// failures are persisted on the delivery and retried independently; they do
// not fail or replay the authoritative maintenance command.
func (p CaptureProcessor) ProcessPending(ctx context.Context, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	for index := 0; index < limit; index++ {
		delivery, ok, err := p.claim(ctx, "")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		p.processClaimed(ctx, delivery)
	}
	return nil
}

func (p CaptureProcessor) ProcessDemonstration(
	ctx context.Context,
	demonstrationID string,
	limit int,
) error {
	if limit <= 0 || limit > 1_000 {
		limit = 500
	}
	for index := 0; index < limit; index++ {
		delivery, ok, err := p.claim(ctx, demonstrationID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		p.processClaimed(ctx, delivery)
	}
	return nil
}

func (p CaptureProcessor) processClaimed(
	ctx context.Context,
	delivery captureDelivery,
) {
	err := p.capture(ctx, delivery)
	if err == nil {
		_ = p.finish(ctx, delivery, "APPLIED", "", time.Time{})
		return
	}
	delay := time.Duration(1<<min(delivery.Attempts, 8)) * time.Second
	_ = p.finish(
		ctx, delivery, "FAILED", truncateError(err),
		p.now().Add(delay),
	)
}

func (p CaptureProcessor) claim(
	ctx context.Context,
	demonstrationID string,
) (captureDelivery, bool, error) {
	var value captureDelivery
	err := p.Pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT c.demonstration_id, c.domain_event_id
			FROM demonstration_capture_deliveries c
			JOIN demonstrations d ON d.id = c.demonstration_id
			JOIN domain_events e ON e.id = c.domain_event_id
			WHERE d.status = 'recording'
			  AND ($1 = '' OR c.demonstration_id::text = $1)
			  AND (
			    (c.status IN ('PENDING', 'FAILED') AND (
			      $1 <> '' OR c.next_attempt_at <= $2
			    ))
			    OR
			    (c.status = 'PROCESSING' AND c.updated_at < $2 - interval '5 minutes')
			  )
			ORDER BY e.occurred_at, c.created_at, c.domain_event_id
			FOR UPDATE OF c SKIP LOCKED
			LIMIT 1
		)
		UPDATE demonstration_capture_deliveries c
		SET status = 'PROCESSING', attempts = c.attempts + 1,
		    last_error = NULL, updated_at = $2
		FROM candidate
		WHERE c.demonstration_id = candidate.demonstration_id
		  AND c.domain_event_id = candidate.domain_event_id
		RETURNING c.demonstration_id::text, c.domain_event_id::text, c.attempts
	`, demonstrationID, p.now()).Scan(
		&value.DemonstrationID, &value.DomainEventID, &value.Attempts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return captureDelivery{}, false, nil
	}
	if err != nil {
		return captureDelivery{}, false, err
	}
	return value, true, nil
}

func (p CaptureProcessor) capture(
	ctx context.Context,
	delivery captureDelivery,
) error {
	var alreadyApplied bool
	if err := p.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM demonstration_events
			WHERE demonstration_id = $1::uuid AND domain_event_id = $2::uuid
		)
	`, delivery.DemonstrationID, delivery.DomainEventID).Scan(&alreadyApplied); err != nil {
		return err
	}
	if alreadyApplied {
		return nil
	}
	source, err := p.loadSource(ctx, delivery.DomainEventID)
	if err != nil {
		return err
	}
	demo, ok, err := loadSDKDemonstration(
		ctx, p.Pool, delivery.DemonstrationID, false,
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("capture target demonstration was not found")
	}
	actor, err := p.actorPrincipal(ctx, source.OrganizationID, source.ActorID)
	if err != nil {
		return err
	}
	event, err := p.mapEvent(ctx, demo, source, actor, delivery.DomainEventID)
	if err != nil {
		return err
	}
	authContext := sdkcore.WithPrincipal(ctx, demo.Principal)
	recorder, err := p.recorder()
	if err != nil {
		return err
	}
	_, err = recorder.Capture(authContext, demo.ID, event)
	if err != nil {
		var sdkErr *sdkcore.SkawldError
		if errors.As(err, &sdkErr) && sdkErr.Kind == sdkcore.ErrorConflict {
			var exists bool
			checkErr := p.Pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM demonstration_events
					WHERE demonstration_id = $1::uuid AND id = $2::uuid
				)
			`, demo.ID, delivery.DomainEventID).Scan(&exists)
			if checkErr == nil && exists {
				return nil
			}
		}
	}
	return err
}

func (p CaptureProcessor) loadSource(
	ctx context.Context,
	domainEventID string,
) (sourceEvent, error) {
	var value sourceEvent
	var payload []byte
	err := p.Pool.QueryRow(ctx, `
		SELECT organization_id::text, coalesce(actor_id::text, ''),
		       event_type, aggregate_type, aggregate_id::text,
		       aggregate_version, coalesce(subject_kind, ''),
		       coalesce(subject_id::text, ''), payload, occurred_at
		FROM domain_events
		WHERE id = $1::uuid
	`, domainEventID).Scan(
		&value.OrganizationID, &value.ActorID, &value.Type,
		&value.AggregateType, &value.AggregateID, &value.AggregateVersion,
		&value.SubjectKind, &value.SubjectID, &payload, &value.OccurredAt,
	)
	if err != nil {
		return sourceEvent{}, err
	}
	if err := json.Unmarshal(payload, &value.Payload); err != nil {
		return sourceEvent{}, fmt.Errorf("decode semantic source payload: %w", err)
	}
	return value, nil
}

func (p CaptureProcessor) actorPrincipal(
	ctx context.Context,
	organizationID, actorID string,
) (sdkcore.Principal, error) {
	if actorID == "" {
		return sdkcore.Principal{}, errors.New("semantic source event has no actor")
	}
	rows, err := p.Pool.Query(ctx, `
		SELECT DISTINCT role
		FROM memberships
		WHERE principal_id = $1::uuid AND organization_id = $2::uuid
	`, actorID, organizationID)
	if err != nil {
		return sdkcore.Principal{}, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return sdkcore.Principal{}, err
		}
		if mapped := RoleName(identitydomain.Role(role)); mapped != "" {
			roles = append(roles, mapped)
		}
	}
	if err := rows.Err(); err != nil {
		return sdkcore.Principal{}, err
	}
	sort.Strings(roles)
	return sdkcore.Principal{
		TenantID: organizationID, ActorID: actorID, Roles: roles,
	}, nil
}

func (p CaptureProcessor) mapEvent(
	ctx context.Context,
	demo sdkobservation.Demonstration,
	source sourceEvent,
	actor sdkcore.Principal,
	domainEventID string,
) (sdkobservation.Event, error) {
	timestamp := source.OccurredAt.UTC()
	if count := len(demo.Trace.Events); count > 0 {
		previous := demo.Trace.Events[count-1].Timestamp
		if timestamp.Before(previous) {
			timestamp = previous
		}
	}
	event := sdkobservation.Event{
		ID: domainEventID, Principal: actor, Timestamp: timestamp,
		Source:      sdkobservation.SourceDatabase,
		Trust:       sdkobservation.TrustApplicationEvent,
		Sensitivity: sdkobservation.SensitivityInternal,
		Application: "skawld-maintenance", Action: source.Type,
		Entity: &sdkobservation.Entity{
			Type: source.AggregateType, ID: source.AggregateID,
		},
		Output: source.Payload,
		Context: map[string]interface{}{
			"domain_event_id":    domainEventID,
			"source_occurred_at": source.OccurredAt.UTC().Format(time.RFC3339Nano),
			"aggregate_version":  source.AggregateVersion,
			"subject_kind":       source.SubjectKind,
			"subject_id":         source.SubjectID,
		},
	}
	switch source.Type {
	case "copilot.recommendation.generated":
		event.Intent = "review evidence-backed maintenance recommendation"
	case "copilot.recommendation.reviewed":
		recommendationID, _ := source.Payload["recommendation_id"].(string)
		if recommendationID == "" {
			return sdkobservation.Event{}, errors.New("recommendation feedback has no recommendation ID")
		}
		var correctionOf string
		err := p.Pool.QueryRow(ctx, `
			SELECT id::text
			FROM demonstration_events
			WHERE demonstration_id = $1::uuid
			  AND action = 'copilot.recommendation.generated'
			  AND entity->>'id' = $2
			ORDER BY ordinal DESC LIMIT 1
		`, demo.ID, recommendationID).Scan(&correctionOf)
		if errors.Is(err, pgx.ErrNoRows) {
			return sdkobservation.Event{}, errors.New("recommendation correction has no captured proposal event")
		}
		if err != nil {
			return sdkobservation.Event{}, err
		}
		event.Context["proposal_event_id"] = correctionOf
		event.Decision = source.Payload
		feedback, _ := source.Payload["value"].(map[string]interface{})
		outcome, _ := feedback["outcome"].(string)
		if outcome == "CORRECTED" {
			event.CorrectionOf = correctionOf
			event.Action = "copilot.recommendation.corrected"
			event.Intent = "record human correction to AI proposal"
		} else {
			event.Intent = "record human review of AI proposal"
		}
	case "decision.recorded":
		event.Decision = source.Payload
		event.Intent = "record technician reasoning"
	case "measurement.recorded":
		event.Intent = "record verified engineering evidence"
	case "observation.recorded":
		event.Intent = "record technician observation"
	case "maintenance.action.recorded":
		event.Intent = "record performed maintenance action"
	case "execution.completed":
		event.Result = source.Payload
		event.Intent = "record maintenance outcome"
	case "handover.edited", "handover.submitted", "handover.accepted",
		"handover.acknowledged":
		event.Intent = "preserve shift handover work and human acceptance"
	}
	return event, nil
}

func (p CaptureProcessor) finish(
	ctx context.Context,
	delivery captureDelivery,
	status, lastError string,
	nextAttempt time.Time,
) error {
	if nextAttempt.IsZero() {
		nextAttempt = p.now()
	}
	_, err := p.Pool.Exec(ctx, `
		UPDATE demonstration_capture_deliveries
		SET status = $3, last_error = nullif($4, ''),
		    next_attempt_at = $5, updated_at = $6
		WHERE demonstration_id = $1::uuid AND domain_event_id = $2::uuid
	`, delivery.DemonstrationID, delivery.DomainEventID, status,
		lastError, nextAttempt, p.now())
	return err
}

func (p CaptureProcessor) recorder() (*sdkobservation.Recorder, error) {
	redactor, err := sdkobservation.NewRedactor(sdkobservation.RedactorOptions{
		Rules: map[string]sdkobservation.RedactionAction{
			"input.access_token":    sdkobservation.RedactDrop,
			"input.password":        sdkobservation.RedactDrop,
			"input.authorization":   sdkobservation.RedactDrop,
			"output.access_token":   sdkobservation.RedactDrop,
			"output.password":       sdkobservation.RedactDrop,
			"output.authorization":  sdkobservation.RedactDrop,
			"context.authorization": sdkobservation.RedactDrop,
		},
	})
	if err != nil {
		return nil, err
	}
	store := p.Store
	if store.Pool == nil {
		store = ObservationStore{Pool: p.Pool, Clock: p.Clock}
	}
	return sdkobservation.NewRecorderWithOptions(sdkobservation.RecorderOptions{
		Store: store, Sanitizer: redactor,
	})
}

func (p CaptureProcessor) now() time.Time {
	if p.Clock == nil {
		return time.Now().UTC()
	}
	return p.Clock.Now().UTC()
}

func truncateError(err error) string {
	value := strings.TrimSpace(err.Error())
	if len(value) > 1_000 {
		return value[:1_000]
	}
	return value
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
