package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	identitypostgres "github.com/ZekromNguyen/skawld-maintenance/internal/identity/adapter/postgres"
	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/auth"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/httpserver"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(config.RoleAPI)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database, config.RoleAPI)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	systemClock := clock.System{}
	idGenerator := id.UUID{}
	authRepository := auth.Repository{Pool: pool, IDs: idGenerator, Clock: systemClock}
	authService, err := auth.New(ctx, cfg.Auth, authRepository, systemClock, logger)
	if err != nil {
		logger.Error("OIDC startup failed", "error", err)
		os.Exit(1)
	}
	organizationStore := identitypostgres.OrganizationStore{
		Pool:        pool,
		IDs:         idGenerator,
		Clock:       systemClock,
		Idempotency: idempotency.Store{},
		Audit:       audit.Sink{},
	}
	handler := httpserver.New(httpserver.Dependencies{
		Logger:   logger,
		Database: pool,
		Auth:     authService,
		Organizations: application.OrganizationService{
			Store: organizationStore,
		},
		Sites: application.SiteService{
			Reader: identitypostgres.SiteReader{Pool: pool},
		},
	})
	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("API listening", "address", cfg.HTTP.Address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("API shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("API stopped")
}
