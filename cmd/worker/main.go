package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/jobs"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(config.RoleWorker)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database, config.RoleWorker)
	if err != nil {
		logger.Error("worker database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	client, err := jobs.New(pool, cfg.Jobs, logger)
	if err != nil {
		logger.Error("worker startup failed", "error", err)
		os.Exit(1)
	}
	if err := client.Start(ctx); err != nil {
		logger.Error("worker start failed", "error", err)
		os.Exit(1)
	}
	logger.Info("worker started")

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Stop(shutdownCtx); err != nil {
		logger.Error("worker shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}
