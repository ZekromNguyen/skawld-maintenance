package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestFoundationHealthContract(t *testing.T) {
	t.Parallel()
	args := FoundationHealthArgs{IdempotencyKey: "test-1"}
	if got, want := args.Kind(), "foundation.healthcheck"; got != want {
		t.Fatalf("kind = %q, want %q", got, want)
	}
	worker := &FoundationHealthWorker{}
	if got, want := worker.Timeout(&river.Job[FoundationHealthArgs]{}), 5*time.Second; got != want {
		t.Fatalf("timeout = %s, want %s", got, want)
	}
}

func TestJobRetryClassificationAndBounds(t *testing.T) {
	t.Parallel()
	attemptedAt := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	job := &rivertype.JobRow{Attempt: 20, AttemptedAt: &attemptedAt}
	if got, want := boundedRetryAt(
		job, time.Minute, 15*time.Minute,
	), attemptedAt.Add(15*time.Minute); !got.Equal(want) {
		t.Fatalf("retry at = %s, want %s", got, want)
	}
	permanent := Permanent(errors.New("invalid source document"))
	if got := classifyError(permanent); got != "permanent" {
		t.Fatalf("classification = %q", got)
	}
	if got := classifyWorkError(permanent); !errors.Is(got, permanent) {
		t.Fatalf("classified work error = %v", got)
	}
}

func TestDocumentIngestUniquenessIncludesAttachment(t *testing.T) {
	t.Parallel()
	first := DocumentIngestArgs{RevisionID: "revision", AttachmentID: "attachment-1"}
	second := DocumentIngestArgs{RevisionID: "revision", AttachmentID: "attachment-2"}
	if first.Kind() != "knowledge.document_ingest" {
		t.Fatalf("kind = %q", first.Kind())
	}
	if first == second {
		t.Fatal("re-ingestion with a new attachment must have distinct job args")
	}
	worker := &DocumentIngestWorker{}
	if got, want := worker.Timeout(&river.Job[DocumentIngestArgs]{}), 2*time.Minute; got != want {
		t.Fatalf("timeout = %s, want %s", got, want)
	}
}

func TestTranscriptionJobContract(t *testing.T) {
	t.Parallel()
	args := TranscriptionArgs{TranscriptionID: "candidate"}
	if args.Kind() != "evidence.transcription" {
		t.Fatalf("kind = %q", args.Kind())
	}
	worker := &TranscriptionWorker{}
	if got, want := worker.Timeout(&river.Job[TranscriptionArgs]{}), 5*time.Minute; got != want {
		t.Fatalf("timeout = %s, want %s", got, want)
	}
}
