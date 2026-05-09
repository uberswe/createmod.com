package jobs

import (
	"context"
	"log/slog"
	"strings"

	"createmod/internal/store"

	"github.com/riverqueue/river"
)

type ZeroResultAnalysisArgs struct{}

func (ZeroResultAnalysisArgs) Kind() string { return "zero_result_analysis" }

type ZeroResultAnalysisWorker struct {
	river.WorkerDefaults[ZeroResultAnalysisArgs]
	deps Deps
}

func (w *ZeroResultAnalysisWorker) Work(ctx context.Context, job *river.Job[ZeroResultAnalysisArgs]) error {
	slog.Info("zero result analysis started")
	if w.deps.Store == nil {
		return nil
	}

	zeroQueries, err := w.deps.Store.SearchTracking.ListTopZeroResultQueries(ctx, 100)
	if err != nil {
		return err
	}

	if len(zeroQueries) == 0 {
		slog.Info("zero result analysis: no zero-result queries")
		return nil
	}

	successfulQueries, err := w.deps.Store.SearchTracking.ListTopSuccessfulQueries(ctx, 500)
	if err != nil {
		return err
	}

	if len(successfulQueries) == 0 {
		slog.Info("zero result analysis: no successful queries to compare")
		return nil
	}

	generated := 0
	for _, zq := range zeroQueries {
		query := strings.ToLower(strings.TrimSpace(zq.Query))
		if len(query) < 2 {
			continue
		}

		bestMatch := ""
		bestDist := 4
		for _, sq := range successfulQueries {
			candidate := strings.ToLower(strings.TrimSpace(sq))
			if candidate == query {
				continue
			}
			d := levenshtein(query, candidate)
			if d < bestDist {
				bestDist = d
				bestMatch = sq
			}
		}

		if bestMatch != "" {
			if err := w.deps.Store.ZeroResults.Upsert(ctx, &store.ZeroResultSuggestion{
				Query:      zq.Query,
				Suggestion: bestMatch,
				Auto:       true,
			}); err != nil {
				slog.Warn("zero result analysis: upsert failed", "query", zq.Query, "err", err)
			} else {
				generated++
			}
		}
	}

	slog.Info("zero result analysis completed", "checked", len(zeroQueries), "suggestions", generated)
	return nil
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
