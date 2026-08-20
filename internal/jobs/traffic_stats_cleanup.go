package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// trafficStatsRetentionDays is how long raw view/download hit buckets are kept.
const trafficStatsRetentionDays = 30

// TrafficStatsCleanupArgs purges traffic_stats rows older than the retention
// window. Enqueued daily; UniqueOpts makes it a cluster-wide singleton.
type TrafficStatsCleanupArgs struct{}

func (TrafficStatsCleanupArgs) Kind() string { return "traffic_stats_cleanup" }

type TrafficStatsCleanupWorker struct {
	river.WorkerDefaults[TrafficStatsCleanupArgs]
	deps Deps
}

func (w *TrafficStatsCleanupWorker) Work(ctx context.Context, _ *river.Job[TrafficStatsCleanupArgs]) error {
	if w.deps.Store == nil || w.deps.Store.TrafficStats == nil {
		return nil
	}
	// day is 'YYYY-MM-DD' UTC text; anything strictly older than the cutoff day
	// is dropped.
	cutoff := time.Now().UTC().AddDate(0, 0, -trafficStatsRetentionDays).Format("2006-01-02")
	if err := w.deps.Store.TrafficStats.DeleteBefore(ctx, cutoff); err != nil {
		return fmt.Errorf("traffic stats cleanup: %w", err)
	}
	slog.Info("traffic stats cleanup complete", "cutoff", cutoff)
	return nil
}
