package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

const (
	riverSchema        = "river"
	QueueFoundation    = "foundation"
	QueueEmbedding     = "embedding"
	QueueReport        = "report"
	QueueVision        = "vision"
	QueueTranscription = "transcription"
)

type FoundationHealthArgs struct {
	IdempotencyKey string `json:"idempotency_key" river:"unique"`
}

func (FoundationHealthArgs) Kind() string { return "foundation.healthcheck" }

type FoundationHealthWorker struct {
	river.WorkerDefaults[FoundationHealthArgs]
	Logger *slog.Logger
}

func (w *FoundationHealthWorker) Work(
	ctx context.Context,
	job *river.Job[FoundationHealthArgs],
) error {
	w.Logger.InfoContext(ctx, "foundation job completed",
		"job_id", job.ID,
		"kind", job.Kind,
	)
	return nil
}

func (*FoundationHealthWorker) Timeout(*river.Job[FoundationHealthArgs]) time.Duration {
	return 5 * time.Second
}

type ErrorHandler struct {
	Logger *slog.Logger
}

func (h ErrorHandler) HandleError(
	ctx context.Context,
	job *rivertype.JobRow,
	err error,
) *river.ErrorHandlerResult {
	h.Logger.ErrorContext(ctx, "job attempt failed",
		"job_id", job.ID,
		"kind", job.Kind,
		"attempt", job.Attempt,
		"error", err,
	)
	if errors.Is(err, context.Canceled) {
		return &river.ErrorHandlerResult{SetCancelled: true}
	}
	return nil
}

func (h ErrorHandler) HandlePanic(
	ctx context.Context,
	job *rivertype.JobRow,
	panicValue any,
	trace string,
) *river.ErrorHandlerResult {
	h.Logger.ErrorContext(ctx, "job panicked",
		"job_id", job.ID,
		"kind", job.Kind,
		"panic", panicValue,
		"trace", trace,
	)
	return &river.ErrorHandlerResult{SetCancelled: true}
}

type Client struct {
	river *river.Client[pgx.Tx]
}

func New(pool *pgxpool.Pool, cfg config.Jobs, logger *slog.Logger) (*Client, error) {
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, &FoundationHealthWorker{Logger: logger}); err != nil {
		return nil, err
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Schema: riverSchema,
		Queues: map[string]river.QueueConfig{
			QueueFoundation:    {MaxWorkers: 1},
			QueueEmbedding:     {MaxWorkers: cfg.EmbeddingConcurrency},
			QueueReport:        {MaxWorkers: cfg.ReportConcurrency},
			QueueVision:        {MaxWorkers: cfg.VisionConcurrency},
			QueueTranscription: {MaxWorkers: cfg.TranscriptionConcurrency},
		},
		Workers:      workers,
		ErrorHandler: ErrorHandler{Logger: logger},
		JobTimeout:   time.Minute,
	})
	if err != nil {
		return nil, err
	}
	return &Client{river: client}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return c.river.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	return c.river.Stop(ctx)
}

func (c *Client) EnqueueFoundationHealth(
	ctx context.Context,
	idempotencyKey string,
) error {
	_, err := c.river.Insert(ctx, FoundationHealthArgs{IdempotencyKey: idempotencyKey}, &river.InsertOpts{
		Queue:       QueueFoundation,
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	})
	return err
}
