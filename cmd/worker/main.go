package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/ingest"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/jobs"
	s3store "github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore/s3"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	transcriptionpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/transcription/adapter/postgres"
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

	objectStore, err := s3store.New(ctx, cfg.ObjectStore)
	if err != nil {
		logger.Error("worker object storage startup failed", "error", err)
		os.Exit(1)
	}
	_, embeddingProvider, err := skawld.BuildProviders(
		skawld.AIConfig{
			EmbeddingProvider:     cfg.AI.EmbeddingProvider,
			StructuredAPIKey:      cfg.AI.APIKey,
			EmbeddingEndpoint:     cfg.AI.EmbeddingEndpoint,
			EmbeddingModel:        cfg.AI.EmbeddingModel,
			EmbeddingModelVersion: cfg.AI.EmbeddingModelVersion,
		},
		&http.Client{Timeout: 30 * time.Second},
	)
	if err != nil {
		logger.Error("AI provider startup failed", "error", err)
		os.Exit(1)
	}
	processor := ingest.Processor{
		Pool: pool, Objects: objectStore,
		Extractor: ingest.BoundedExtractor{
			PDFToTextBinary: cfg.Documents.PDFToTextBinary,
		},
		Embeddings: embeddingProvider,
		IDs:        id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
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
	transcriptionProcessor := transcriptionpostgres.Processor{
		Pool: pool, Objects: objectStore, Provider: transcriptionProvider,
		IDs: id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	captureProcessor := skawld.CaptureProcessor{
		Pool: pool, Clock: clock.System{},
		Store: skawld.ObservationStore{Pool: pool, Clock: clock.System{}},
	}
	client, err := jobs.NewWithWorkers(pool, cfg.Jobs, logger, jobs.WorkerSet{
		DocumentIngestor: processor, Transcriber: transcriptionProcessor,
		SemanticCapturer: captureProcessor,
	})
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
