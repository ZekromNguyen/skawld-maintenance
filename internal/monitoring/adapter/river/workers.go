package river

import (
	"context"
	"time"

	monitoringpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/adapter/postgres"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// boundedRetryAt mirrors internal/platform/jobs.boundedRetryAt (unexported
// there): exponential backoff from base up to maximum, capped at 10 attempts.
func boundedRetryAt(job *rivertype.JobRow, base, maximum time.Duration) time.Time {
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
	return time.Now().Add(delay)
}

// SnapshotArgs runs the 5-minute system/infra collection.
type SnapshotArgs struct{}

func (SnapshotArgs) Kind() string { return "monitoring.snapshot" }

type SnapshotWorker struct {
	river.WorkerDefaults[SnapshotArgs]
	Store monitoringpostgres.Store
}

func (w *SnapshotWorker) Work(ctx context.Context, _ *river.Job[SnapshotArgs]) error {
	scopes, err := w.Store.ListScopes(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, scope := range scopes {
		if err := CollectSnapshot(ctx, w.Store, scope.OrganizationID, scope.SiteIDs, now); err != nil {
			return err
		}
	}
	return nil
}

func (*SnapshotWorker) Timeout(*river.Job[SnapshotArgs]) time.Duration {
	return 2 * time.Minute
}

func (*SnapshotWorker) NextRetry(job *river.Job[SnapshotArgs]) time.Time {
	return boundedRetryAt(job.JobRow, 5*time.Second, time.Minute)
}

// DailyRollupArgs runs the daily operational KPI rollup.
type DailyRollupArgs struct{}

func (DailyRollupArgs) Kind() string { return "monitoring.daily_rollup" }

type DailyRollupWorker struct {
	river.WorkerDefaults[DailyRollupArgs]
	Store monitoringpostgres.Store
}

func (w *DailyRollupWorker) Work(ctx context.Context, _ *river.Job[DailyRollupArgs]) error {
	scopes, err := w.Store.ListScopes(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, scope := range scopes {
		if err := CollectDaily(ctx, w.Store, scope.OrganizationID, scope.SiteIDs, now); err != nil {
			return err
		}
	}
	return nil
}

func (*DailyRollupWorker) Timeout(*river.Job[DailyRollupArgs]) time.Duration {
	return 5 * time.Minute
}

func (*DailyRollupWorker) NextRetry(job *river.Job[DailyRollupArgs]) time.Time {
	return boundedRetryAt(job.JobRow, time.Minute, 30*time.Minute)
}
