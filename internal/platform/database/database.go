package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, cfg config.Database, role config.Role) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = cfg.StatementTimeout.String()
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = cfg.LockTimeout.String()
	switch role {
	case config.RoleAPI:
		poolConfig.MaxConns = cfg.APIMaxConns
	case config.RoleWorker:
		poolConfig.MaxConns = cfg.WorkerMaxConns
		poolConfig.ConnConfig.RuntimeParams["search_path"] = "river,public"
	default:
		return nil, fmt.Errorf("unsupported database pool role %q", role)
	}
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func InTx[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	options pgx.TxOptions,
	fn func(pgx.Tx) (T, error),
) (T, error) {
	var zero T
	tx, err := pool.BeginTx(ctx, options)
	if err != nil {
		return zero, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	result, err := fn(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("commit transaction: %w", err)
	}
	return result, nil
}
