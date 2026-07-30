package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// aggregateWindowDays is how many trailing complete days the job re-aggregates
// each run. Covers the 30-day stats window with buffer; re-aggregating (rather
// than tracking a cursor) keeps it self-healing against late data and pruning.
const aggregateWindowDays = 35

// SearchStatsAggregateArgs are the arguments for the daily search-stats
// aggregation job.
type SearchStatsAggregateArgs struct{}

func (SearchStatsAggregateArgs) Kind() string { return "search_stats_aggregate" }

// SearchStatsAggregateWorker rolls raw searches up into per-day, per-term
// counts (search_term_daily) so the site-stats page reads a small pre-computed
// table instead of scanning millions of raw rows on every load. It aggregates
// only COMPLETE UTC days — the current day is never written, so the Daily
// Search Volume / Trending Search Terms charts never show a partial-day dip.
type SearchStatsAggregateWorker struct {
	river.WorkerDefaults[SearchStatsAggregateArgs]
	deps Deps
}

func (w *SearchStatsAggregateWorker) Work(ctx context.Context, job *river.Job[SearchStatsAggregateArgs]) error {
	if w.deps.Store == nil {
		return nil
	}
	st := w.deps.Store.SearchTracking

	// Only aggregate complete days that aren't stored yet. daysBehind is 0 when
	// yesterday is already aggregated (so most runs are a no-op), the gap when
	// the job fell behind, or a large sentinel on an empty table (first
	// backfill). Cap it so the initial backfill is bounded to the stats window.
	daysBehind, err := st.SearchTermDailyDaysBehind(ctx)
	if err != nil {
		slog.Error("search stats aggregate: days-behind check failed", "error", err)
		return err
	}
	if daysBehind <= 0 {
		return nil // yesterday already aggregated; current day is deliberately skipped
	}
	if daysBehind > aggregateWindowDays {
		daysBehind = aggregateWindowDays
	}

	// [start, end) covers the daysBehind trailing complete UTC days. end is
	// today's UTC midnight, so the current (partial) day is never aggregated —
	// that's what keeps the charts from showing a partial-day dip.
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -daysBehind)

	if err := st.AggregateSearchTermDaily(ctx, start, end); err != nil {
		slog.Error("search stats aggregate: failed", "error", err)
		return err
	}
	if _, err := st.PruneSearchTermDaily(ctx); err != nil {
		slog.Warn("search stats aggregate: prune failed", "error", err)
	}

	slog.Info("search stats aggregated",
		"from", start.Format("2006-01-02"), "through", end.AddDate(0, 0, -1).Format("2006-01-02"))
	return nil
}
