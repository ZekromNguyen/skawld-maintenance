package jobs

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
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
