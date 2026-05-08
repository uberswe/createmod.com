package jobs

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/riverqueue/river"
)

type SearchStatsFilterArgs struct{}

func (SearchStatsFilterArgs) Kind() string { return "search_stats_filter" }

type SearchStatsFilterWorker struct {
	river.WorkerDefaults[SearchStatsFilterArgs]
	deps Deps
}

var suspiciousPattern = regexp.MustCompile(`(?i)(<script|SELECT\s|INSERT\s|DROP\s|DELETE\s|UNION\s|--|;)`)

func (w *SearchStatsFilterWorker) Work(ctx context.Context, job *river.Job[SearchStatsFilterArgs]) error {
	slog.Info("search stats filter started")
	if w.deps.Store == nil {
		return nil
	}

	now := time.Now().UTC()
	since30d := now.AddDate(0, 0, -30)

	raw, err := w.deps.Store.SearchTracking.ListTopSearchesSince(ctx, since30d, 200)
	if err != nil {
		slog.Error("search stats filter: list top searches", "error", err)
		return nil
	}
	if len(raw) == 0 {
		slog.Info("search stats filter: no searches found")
		return nil
	}

	var terms []string
	for _, r := range raw {
		q := strings.TrimSpace(r.Query)
		if q == "" || utf8.RuneCountInString(q) > 100 {
			continue
		}
		if suspiciousPattern.MatchString(q) {
			if err := w.deps.Store.SearchTracking.UpsertSearchTermModeration(ctx, q, false); err != nil {
				slog.Error("search stats filter: upsert heuristic reject", "query", q, "error", err)
			}
			continue
		}
		terms = append(terms, q)
	}

	if len(terms) == 0 {
		slog.Info("search stats filter: all terms filtered by heuristics")
		return nil
	}

	staleThreshold := now.Add(-24 * time.Hour)
	unchecked, err := w.deps.Store.SearchTracking.ListUncheckedSearchTerms(ctx, terms, staleThreshold)
	if err != nil {
		slog.Error("search stats filter: list unchecked", "error", err)
		return nil
	}

	if len(unchecked) > 0 && w.deps.Moderation != nil {
		slog.Info("search stats filter: moderating terms", "count", len(unchecked))
		for _, term := range unchecked {
			result, err := w.deps.Moderation.CheckContent(term)
			if err != nil {
				slog.Error("search stats filter: moderation API error", "term", term, "error", err)
				continue
			}
			isClean := result.Approved
			if err := w.deps.Store.SearchTracking.UpsertSearchTermModeration(ctx, term, isClean); err != nil {
				slog.Error("search stats filter: upsert moderation result", "term", term, "error", err)
			}
		}
	}

	slog.Info("search stats filter completed", "total_terms", len(terms), "newly_checked", len(unchecked))
	return nil
}
