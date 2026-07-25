package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const createOrganizationScope = "identity.organization.create.v1"

type OrganizationStore struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Idempotency idempotency.Store
	Audit       audit.Sink
}

func (s OrganizationStore) Create(
	ctx context.Context,
	principal domain.Principal,
	idempotencyKey string,
	command application.CreateOrganization,
) (application.CreateOrganizationResult, bool, error) {
	requestHash, err := idempotency.HashRequest(command)
	if err != nil {
		return application.CreateOrganizationResult{}, false, err
	}
	if s.Pool == nil || s.IDs == nil || s.Clock == nil {
		return application.CreateOrganizationResult{}, false, fmt.Errorf("organization store dependencies are required")
	}

	type txResult struct {
		Result application.CreateOrganizationResult
		Replay bool
	}
	outcome, err := database.InTx(ctx, s.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (txResult, error) {
		now := s.Clock.Now()
		record, err := s.Idempotency.Begin(
			ctx, tx, principal.ID, createOrganizationScope, idempotencyKey, requestHash, now,
		)
		if err != nil {
			return txResult{}, err
		}
		if record.Replay {
			var replayed application.CreateOrganizationResult
			if err := json.Unmarshal(record.ResponseBody, &replayed); err != nil {
				return txResult{}, fmt.Errorf("decode idempotent response: %w", err)
			}
			return txResult{Result: replayed, Replay: true}, nil
		}

		organization, err := domain.NewOrganization(s.IDs.New(), command.Name, now)
		if err != nil {
			return txResult{}, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO organizations (
				id, name, source_of_truth, version, created_at, updated_at
			) VALUES ($1::uuid, $2, $3, $4, $5, $6)
		`, organization.ID, organization.Name, organization.SourceOfTruth,
			organization.Version, organization.CreatedAt, organization.UpdatedAt)
		if err != nil {
			return txResult{}, fmt.Errorf("insert organization: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO memberships (
				id, principal_id, organization_id, site_id, role, created_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, NULL, 'Administrator', $4)
			ON CONFLICT DO NOTHING
		`, s.IDs.New(), principal.ID, organization.ID, now)
		if err != nil {
			return txResult{}, fmt.Errorf("insert bootstrap administrator membership: %w", err)
		}

		result := application.CreateOrganizationResult{
			ID:            organization.ID,
			Name:          organization.Name,
			SourceOfTruth: string(organization.SourceOfTruth),
			Version:       organization.Version,
		}
		body, err := json.Marshal(result)
		if err != nil {
			return txResult{}, fmt.Errorf("encode organization result: %w", err)
		}
		if err := s.Audit.Append(ctx, tx, audit.Event{
			ID:             s.IDs.New(),
			OrganizationID: organization.ID,
			ActorID:        principal.ID,
			Action:         "organization.created",
			EntityKind:     "organization",
			EntityID:       organization.ID,
			After:          result,
			OccurredAt:     now,
		}); err != nil {
			return txResult{}, err
		}
		if err := s.Idempotency.Complete(
			ctx, tx, principal.ID, createOrganizationScope, idempotencyKey,
			http.StatusCreated, body, now,
		); err != nil {
			return txResult{}, err
		}
		return txResult{Result: result}, nil
	})
	if err != nil {
		if errors.Is(err, idempotency.ErrKeyConflict) {
			return application.CreateOrganizationResult{}, false, err
		}
		return application.CreateOrganizationResult{}, false, err
	}
	return outcome.Result, outcome.Replay, nil
}
