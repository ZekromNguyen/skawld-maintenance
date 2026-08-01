package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool        *pgxpool.Pool
	IDs         id.Generator
	Clock       clock.Clock
	Idempotency idempotency.Store
	Audit       audit.Sink
}

type commandResult[T any] struct {
	Value  T
	Replay bool
}

func runCommand[T any](
	ctx context.Context,
	store Store,
	principal identitydomain.Principal,
	scope, key string,
	command any,
	status int,
	fn func(context.Context, pgx.Tx, time.Time) (T, error),
) (T, bool, error) {
	var zero T
	hash, err := idempotency.HashRequest(command)
	if err != nil {
		return zero, false, err
	}
	result, err := database.InTx(ctx, store.Pool, pgx.TxOptions{}, func(tx pgx.Tx) (commandResult[T], error) {
		now := store.Clock.Now()
		record, err := store.Idempotency.Begin(ctx, tx, principal.ID, scope, key, hash, now)
		if err != nil {
			return commandResult[T]{}, err
		}
		if record.Replay {
			var replay T
			if err := json.Unmarshal(record.ResponseBody, &replay); err != nil {
				return commandResult[T]{}, fmt.Errorf("decode idempotent execution response: %w", err)
			}
			return commandResult[T]{Value: replay, Replay: true}, nil
		}
		value, err := fn(ctx, tx, now)
		if err != nil {
			return commandResult[T]{}, err
		}
		body, err := json.Marshal(value)
		if err != nil {
			return commandResult[T]{}, err
		}
		if err := store.Idempotency.Complete(
			ctx, tx, principal.ID, scope, key, status, body, now,
		); err != nil {
			return commandResult[T]{}, err
		}
		return commandResult[T]{Value: value}, nil
	})
	if err != nil {
		return zero, false, err
	}
	return result.Value, result.Replay, nil
}

func appendSyncInbox(
	ctx context.Context,
	tx pgx.Tx,
	organizationID, deviceID, clientEventID, key string,
	payload, result any,
	baseServerVersion *int64,
	serverVersion int64,
	createdAtDevice *time.Time,
	now time.Time,
) error {
	if deviceID == "" || clientEventID == "" || createdAtDevice == nil {
		return nil
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payloadJSON)
	_, err = tx.Exec(ctx, `
		INSERT INTO sync_inbox (
			organization_id, device_id, client_event_id, idempotency_key,
			request_hash, payload_version, base_server_version, created_at_device,
			received_at_server, status, result, server_version
		) VALUES (
			$1::uuid, $2, $3::uuid, $4, $5, 1, $6, $7, $8, 'APPLIED', $9::jsonb, $10
		)
		ON CONFLICT (organization_id, device_id, client_event_id) DO NOTHING
	`, organizationID, deviceID, clientEventID, key, hash[:], baseServerVersion,
		createdAtDevice, now, resultJSON, serverVersion)
	return err
}

func mapStoreError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return executionapp.ErrNotFound
	}
	return err
}

func createdStatus() int {
	return http.StatusCreated
}
