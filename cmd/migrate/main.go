package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if len(os.Args) != 2 {
		logger.Error("usage: migrate <up|status>")
		os.Exit(2)
	}
	databaseURL := os.Getenv("MIGRATION_DATABASE_URL")
	if databaseURL == "" {
		logger.Error("MIGRATION_DATABASE_URL is required")
		os.Exit(2)
	}
	if err := database.Migrate(context.Background(), databaseURL, os.Args[1]); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	if os.Args[1] == "up" {
		if err := database.MigrateRiver(context.Background(), databaseURL); err != nil {
			logger.Error("River migration failed", "error", err)
			os.Exit(1)
		}
	}
}
