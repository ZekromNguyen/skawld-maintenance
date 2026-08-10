package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRuntimeRolePrivilegesAndTransactionRollback(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var auditUpdate, auditDelete, schemaCreate bool
	var demonstrationEventUpdate, demonstrationEventDelete, reviewUpdate bool
	err = pool.QueryRow(ctx, `
		SELECT
			has_table_privilege(current_user, 'public.audit_events', 'UPDATE'),
			has_table_privilege(current_user, 'public.audit_events', 'DELETE'),
			has_schema_privilege(current_user, 'public', 'CREATE'),
			has_table_privilege(current_user, 'public.demonstration_events', 'UPDATE'),
			has_table_privilege(current_user, 'public.demonstration_events', 'DELETE'),
			has_table_privilege(current_user, 'public.demonstration_reviews', 'UPDATE')
	`).Scan(
		&auditUpdate, &auditDelete, &schemaCreate,
		&demonstrationEventUpdate, &demonstrationEventDelete, &reviewUpdate,
	)
	if err != nil {
		t.Fatal(err)
	}
	if auditUpdate || auditDelete || schemaCreate || demonstrationEventUpdate ||
		demonstrationEventDelete || reviewUpdate {
		t.Fatalf(
			"unsafe runtime privileges: audit update=%t delete=%t schema create=%t demonstration update=%t delete=%t review update=%t",
			auditUpdate, auditDelete, schemaCreate, demonstrationEventUpdate,
			demonstrationEventDelete, reviewUpdate,
		)
	}

	sentinel := errors.New("rollback requested")
	state := []byte("rollback-integration-test")
	_, err = InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
		_, insertErr := tx.Exec(ctx, `
			INSERT INTO auth_flows (
				state_hash, nonce, pkce_verifier, return_to, expires_at, created_at
			) VALUES ($1, 'nonce', 'verifier', '/', $2, $3)
		`, state, time.Now().Add(time.Minute), time.Now())
		if insertErr != nil {
			return struct{}{}, insertErr
		}
		return struct{}{}, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transaction error = %v, want rollback sentinel", err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM auth_flows WHERE state_hash = $1`, state,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back row count = %d, want 0", count)
	}
}
