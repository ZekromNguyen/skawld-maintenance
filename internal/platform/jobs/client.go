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
	QueueCapture       = "semantic_capture"
)

type FoundationHealthArgs struct {
	IdempotencyKey string `json:"idempotency_key" river:"unique"`
}

type DocumentIngestArgs struct {
	RevisionID   string `json:"revision_id" river:"unique"`
	AttachmentID string `json:"attachment_id" river:"unique"`
}

func (DocumentIngestArgs) Kind() string { return "knowledge.document_ingest" }

type DocumentIngestor interface {
	ProcessDocument(context.Context, string) error
}

type TranscriptionArgs struct {
	TranscriptionID string `json:"transcription_id" river:"unique"`
}

func (TranscriptionArgs) Kind() string { return "evidence.transcription" }

type Transcriber interface {
	ProcessTranscription(context.Context, string) error
}

type SemanticCapturer interface {
	ProcessPending(context.Context, int) error
}

type SemanticCaptureArgs struct{}

func (SemanticCaptureArgs) Kind() string { return "demonstration.semantic_capture" }

type SemanticCaptureWorker struct {
	river.WorkerDefaults[SemanticCaptureArgs]
	Capturer SemanticCapturer
}

func (w *SemanticCaptureWorker) Work(
	ctx context.Context,
	_ *river.Job[SemanticCaptureArgs],
) error {
	return classifyWorkError(w.Capturer.ProcessPending(ctx, 100))
}

func (*SemanticCaptureWorker) Timeout(*river.Job[SemanticCaptureArgs]) time.Duration {
	return 30 * time.Second
}

func (*SemanticCaptureWorker) NextRetry(
	job *river.Job[SemanticCaptureArgs],
) time.Time {
	return boundedRetryAt(job.JobRow, 5*time.Second, time.Minute)
}

type TranscriptionWorker struct {
	river.WorkerDefaults[TranscriptionArgs]
	Transcriber Transcriber
}

func (w *TranscriptionWorker) Work(
	ctx context.Context,
	job *river.Job[TranscriptionArgs],
) error {
	return classifyWorkError(
		w.Transcriber.ProcessTranscription(ctx, job.Args.TranscriptionID),
	)
}

func (*TranscriptionWorker) Timeout(*river.Job[TranscriptionArgs]) time.Duration {
	return 5 * time.Minute
}

func (*TranscriptionWorker) NextRetry(
	job *river.Job[TranscriptionArgs],
) time.Time {
	return boundedRetryAt(job.JobRow, time.Minute, 30*time.Minute)
}

type WorkerSet struct {
	DocumentIngestor DocumentIngestor
	Transcriber      Transcriber
	SemanticCapturer SemanticCapturer
}

type DocumentIngestWorker struct {
	river.WorkerDefaults[DocumentIngestArgs]
	Ingestor DocumentIngestor
}

func (w *DocumentIngestWorker) Work(
	ctx context.Context,
	job *river.Job[DocumentIngestArgs],
) error {
	return classifyWorkError(
		w.Ingestor.ProcessDocument(ctx, job.Args.RevisionID),
	)
}

func (*DocumentIngestWorker) Timeout(*river.Job[DocumentIngestArgs]) time.Duration {
	return 2 * time.Minute
}

func (*DocumentIngestWorker) NextRetry(
	job *river.Job[DocumentIngestArgs],
) time.Time {
	return boundedRetryAt(job.JobRow, 30*time.Second, 15*time.Minute)
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

func (*FoundationHealthWorker) NextRetry(
	job *river.Job[FoundationHealthArgs],
) time.Time {
	return boundedRetryAt(job.JobRow, 5*time.Second, time.Minute)
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
		"classification", classifyError(err),
		"error", err,
	)
	var permanent *PermanentError
	if errors.Is(err, context.Canceled) || errors.As(err, &permanent) {
		return &river.ErrorHandlerResult{SetCancelled: true}
	}
	return nil
}

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	if e == nil || e.Err == nil {
		return "permanent job error"
	}
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

func classifyWorkError(err error) error {
	var permanent *PermanentError
	if errors.As(err, &permanent) {
		return river.JobCancel(err)
	}
	return err
}

func classifyError(err error) string {
	var permanent *PermanentError
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	case errors.As(err, &permanent):
		return "permanent"
	default:
		return "retryable"
	}
}

func boundedRetryAt(
	job *rivertype.JobRow,
	base, maximum time.Duration,
) time.Time {
	if job == nil {
		return time.Time{}
	}
	attempt := job.Attempt
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 10 {
		attempt = 10
	}
	delay := base * time.Duration(1<<(attempt-1))
	if delay > maximum {
		delay = maximum
	}
	start := time.Now().UTC()
	if job.AttemptedAt != nil {
		start = job.AttemptedAt.UTC()
	}
	return start.Add(delay)
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

func New(
	pool *pgxpool.Pool,
	cfg config.Jobs,
	logger *slog.Logger,
	documentIngestors ...DocumentIngestor,
) (*Client, error) {
	set := WorkerSet{}
	if len(documentIngestors) > 0 {
		set.DocumentIngestor = documentIngestors[0]
	}
	return NewWithWorkers(pool, cfg, logger, set)
}

func NewWithWorkers(
	pool *pgxpool.Pool,
	cfg config.Jobs,
	logger *slog.Logger,
	set WorkerSet,
) (*Client, error) {
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, &FoundationHealthWorker{Logger: logger}); err != nil {
		return nil, err
	}
	if set.DocumentIngestor != nil {
		if err := river.AddWorkerSafely(
			workers,
			&DocumentIngestWorker{Ingestor: set.DocumentIngestor},
		); err != nil {
			return nil, err
		}
	}
	if set.Transcriber != nil {
		if err := river.AddWorkerSafely(
			workers,
			&TranscriptionWorker{Transcriber: set.Transcriber},
		); err != nil {
			return nil, err
		}
	}
	var periodicJobs []*river.PeriodicJob
	if set.SemanticCapturer != nil {
		if err := river.AddWorkerSafely(
			workers,
			&SemanticCaptureWorker{Capturer: set.SemanticCapturer},
		); err != nil {
			return nil, err
		}
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(2*time.Second),
			func() (river.JobArgs, *river.InsertOpts) {
				return SemanticCaptureArgs{}, &river.InsertOpts{
					Queue: QueueCapture, MaxAttempts: 3,
					UniqueOpts: river.UniqueOpts{ByPeriod: 2 * time.Second},
				}
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Schema: riverSchema,
		Queues: map[string]river.QueueConfig{
			QueueFoundation:    {MaxWorkers: 1},
			QueueEmbedding:     {MaxWorkers: cfg.EmbeddingConcurrency},
			QueueReport:        {MaxWorkers: cfg.ReportConcurrency},
			QueueVision:        {MaxWorkers: cfg.VisionConcurrency},
			QueueTranscription: {MaxWorkers: cfg.TranscriptionConcurrency},
			QueueCapture:       {MaxWorkers: 1},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
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

func (c *Client) EnqueueDocumentIngestionTx(
	ctx context.Context,
	tx pgx.Tx,
	revisionID, attachmentID string,
) error {
	_, err := c.river.InsertTx(
		ctx,
		tx,
		DocumentIngestArgs{RevisionID: revisionID, AttachmentID: attachmentID},
		&river.InsertOpts{
			Queue:       QueueEmbedding,
			MaxAttempts: 5,
			UniqueOpts: river.UniqueOpts{
				ByArgs: true,
			},
		},
	)
	return err
}

func (c *Client) EnqueueTranscriptionTx(
	ctx context.Context,
	tx pgx.Tx,
	transcriptionID string,
) error {
	_, err := c.river.InsertTx(
		ctx, tx, TranscriptionArgs{TranscriptionID: transcriptionID},
		&river.InsertOpts{
			Queue: QueueTranscription, MaxAttempts: 4,
			UniqueOpts: river.UniqueOpts{ByArgs: true},
		},
	)
	return err
}
