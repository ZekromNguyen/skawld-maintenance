package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ZekromNguyen/skawld-maintenance/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Migrate(ctx context.Context, databaseURL, command string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	switch command {
	case "up":
		err = goose.UpContext(ctx, db, ".")
	case "status":
		err = goose.StatusContext(ctx, db, ".")
	default:
		return fmt.Errorf("unsupported migration command %q", command)
	}
	if err != nil {
		return fmt.Errorf("migration %s: %w", command, err)
	}
	return nil
}
