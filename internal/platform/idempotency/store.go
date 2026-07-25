package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrKeyConflict = errors.New("idempotency key was already used with a different request")
	ErrInProgress  = errors.New("idempotent operation is still processing")
)

type Record struct {
	Replay         bool
	ResponseStatus int
	ResponseBody   json.RawMessage
}

type Store struct{}

func HashRequest(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal request for idempotency: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return sum[:], nil
}

func (Store) Begin(
	ctx context.Context,
	tx pgx.Tx,
	principalID, scope, key string,
	requestHash []byte,
	now time.Time,
) (Record, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (
			principal_id, scope, idempotency_key, request_hash, status, created_at
		) VALUES ($1::uuid, $2, $3, $4, 'PROCESSING', $5)
		ON CONFLICT DO NOTHING
	`, principalID, scope, key, requestHash, now.UTC())
	if err != nil {
		return Record{}, fmt.Errorf("insert idempotency key: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return Record{}, nil
	}

	var storedHash []byte
	var status string
	var responseStatus *int
	var responseBody []byte
	err = tx.QueryRow(ctx, `
		SELECT request_hash, status, response_status, response_body
		FROM idempotency_keys
		WHERE principal_id = $1::uuid AND scope = $2 AND idempotency_key = $3
		FOR UPDATE
	`, principalID, scope, key).Scan(&storedHash, &status, &responseStatus, &responseBody)
	if err != nil {
		return Record{}, fmt.Errorf("load idempotency key: %w", err)
	}
	if !equalBytes(storedHash, requestHash) {
		return Record{}, ErrKeyConflict
	}
	if status != "COMPLETED" || responseStatus == nil {
		return Record{}, ErrInProgress
	}
	return Record{
		Replay:         true,
		ResponseStatus: *responseStatus,
		ResponseBody:   json.RawMessage(responseBody),
	}, nil
}

func (Store) Complete(
	ctx context.Context,
	tx pgx.Tx,
	principalID, scope, key string,
	status int,
	body []byte,
	now time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = 'COMPLETED', response_status = $4, response_body = $5::jsonb, completed_at = $6
		WHERE principal_id = $1::uuid AND scope = $2 AND idempotency_key = $3
		  AND status = 'PROCESSING'
	`, principalID, scope, key, status, body, now.UTC())
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrInProgress
	}
	return nil
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var different byte
	for index := range left {
		different |= left[index] ^ right[index]
	}
	return different == 0
}
