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

	assetpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/asset/adapter/postgres"
	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	attachmentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/adapter/postgres"
	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	copilotpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/adapter/postgres"
	copilotapp "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/application"
	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	evaluationpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/adapter/postgres"
	evaluationapp "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/application"
	executionpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/execution/adapter/postgres"
	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	handoverpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/handover/adapter/postgres"
	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitypostgres "github.com/ZekromNguyen/skawld-maintenance/internal/identity/adapter/postgres"
	identityapp "github.com/ZekromNguyen/skawld-maintenance/internal/identity/application"
	incidentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/incident/adapter/postgres"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	knowledgepostgres "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/adapter/postgres"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/auth"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/httpserver"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/jobs"
	s3store "github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore/s3"
	reportpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/report/adapter/postgres"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	transcriptionpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/transcription/adapter/postgres"
	transcriptionapp "github.com/ZekromNguyen/skawld-maintenance/internal/transcription/application"
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
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
	assetStore := assetpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	incidentStore := incidentpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	executionStore := executionpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	objectStore, err := s3store.New(ctx, cfg.ObjectStore)
	if err != nil {
		logger.Error("object storage startup failed", "error", err)
		os.Exit(1)
	}
	attachmentStore := attachmentpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{}, Objects: objectStore,
	}
	jobClient, err := jobs.New(pool, cfg.Jobs, logger)
	if err != nil {
		logger.Error("job producer startup failed", "error", err)
		os.Exit(1)
	}
	var transcriptionProvider skawld.TranscriptionProvider = skawld.UnavailableTranscriptionProvider{}
	if cfg.Transcription.Endpoint != "" {
		transcriptionProvider, err = skawld.NewHTTPTranscriptionProvider(
			skawld.HTTPTranscriptionConfig{
				Endpoint: cfg.Transcription.Endpoint, APIKey: cfg.Transcription.APIKey,
				Provider: cfg.Transcription.Provider, Model: cfg.Transcription.Model,
				ModelVersion: cfg.Transcription.ModelVersion,
			},
			&http.Client{Timeout: 4 * time.Minute},
		)
		if err != nil {
			logger.Error("transcription provider startup failed", "error", err)
			os.Exit(1)
		}
	}
	embeddingProvider := skawld.DeterministicEmbeddingProvider{Dimensions: 64}
	structuredProvider := skawld.DeterministicProvider{}
	modelRouter := skawld.Router{Providers: map[skawld.Capability]skawld.StructuredProvider{
		skawld.CapabilityRecommendation: structuredProvider,
		skawld.CapabilityReportDraft:    structuredProvider,
		skawld.CapabilityShiftHandover:  structuredProvider,
	}}
	authorityReader := identitypostgres.AuthorityReader{Pool: pool}
	knowledgeStore := knowledgepostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
		Enqueuer: jobClient, Embeddings: embeddingProvider,
	}
	copilotStore := copilotpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	reportStore := reportpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	handoverStore := handoverpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{},
	}
	transcriptionStore := transcriptionpostgres.Store{
		Pool: pool, IDs: idGenerator, Clock: systemClock,
		Idempotency: idempotency.Store{}, Audit: audit.Sink{}, Enqueuer: jobClient,
	}
	knowledgeService := knowledgeapp.Service{
		Store: knowledgeStore, Authorities: authorityReader,
		Now: systemClock.Now,
	}
	openAPI, err := loadOpenAPIDocument()
	if err != nil {
		logger.Warn("openapi document not served", "error", err)
	}
	handler := httpserver.New(httpserver.Dependencies{
		Logger:   logger,
		Database: pool,
		Auth:     authService,
		OpenAPI:  openAPI,
		Organizations: identityapp.OrganizationService{
			Store: organizationStore,
		},
		Sites: identityapp.SiteService{
			Reader: identitypostgres.SiteReader{Pool: pool},
		},
		Assets: assetapp.Service{
			Store:       assetStore,
			Authorities: identitypostgres.AuthorityReader{Pool: pool},
		},
		Incidents:  incidentapp.Service{Store: incidentStore},
		Executions: executionapp.Service{Store: executionStore},
		Attachments: attachmentapp.Service{
			Store: attachmentStore,
		},
		Knowledge: knowledgeService,
		Copilot: copilotapp.Service{
			Store: copilotStore, Search: knowledgeService, Router: modelRouter,
		},
		Reports: reportapp.Service{
			Store: reportStore, Search: knowledgeService, Router: modelRouter,
			Authorities: authorityReader, Now: systemClock.Now,
		},
		Handovers: handoverapp.Service{
			Store: handoverStore, Router: modelRouter,
			Authorities: authorityReader, Now: systemClock.Now,
		},
		Transcriptions: transcriptionapp.Service{
			Store: transcriptionStore, Provider: transcriptionProvider,
		},
		Demonstrations: demonstrationapp.Service{
			Gateway: skawld.DemonstrationGateway{
				Pool: pool, IDs: idGenerator, Clock: systemClock, Audit: audit.Sink{},
			},
			Authorities: authorityReader, Now: systemClock.Now,
		},
		Workflows: workflowapp.Service{
			Gateway: skawld.WorkflowLearningGateway{
				Pool: pool, IDs: idGenerator, Clock: systemClock,
				Audit: audit.Sink{},
			},
			Authorities: authorityReader, Now: systemClock.Now,
		},
		Evaluations: evaluationapp.Service{
			Store: evaluationpostgres.Store{Pool: pool},
			Now:   systemClock.Now,
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

// loadOpenAPIDocument returns the bytes of the OpenAPI contract so the API can
// serve it publicly. The location is explicit when SKAWLD_OPENAPI_PATH is set;
// otherwise it is resolved relative to the working directory (the dev layout).
func loadOpenAPIDocument() ([]byte, error) {
	candidates := []string{os.Getenv("SKAWLD_OPENAPI_PATH"), "api/openapi.yaml"}
	var firstErr error
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		document, err := os.ReadFile(candidate)
		if err == nil {
			return document, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, firstErr
}
