package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// autoReviewBackfillBatch bounds how many stranded schematics one sweep
// re-enqueues, capping the burst of (paid) OpenAI moderation calls.
const autoReviewBackfillBatch = 20

// autoReviewStuckAfter is how long a schematic may sit in auto_review before the
// backfill treats it as stranded. A fresh upload's own moderation job runs well
// within this window, so anything older never got processed (legacy 026-migration
// rows) or had its job discarded.
const autoReviewStuckAfter = 15 * time.Minute

// AutoReviewBackfillArgs re-enqueues the moderation check for schematics stuck
// in auto_review, so they leave that invisible, unqueued state instead of being
// silently lost. (#1646)
type AutoReviewBackfillArgs struct{}

func (AutoReviewBackfillArgs) Kind() string { return "auto_review_backfill" }

// AutoReviewBackfillWorker runs the stranded-auto_review sweep.
type AutoReviewBackfillWorker struct {
	river.WorkerDefaults[AutoReviewBackfillArgs]
	deps Deps
}

func (w *AutoReviewBackfillWorker) Work(ctx context.Context, job *river.Job[AutoReviewBackfillArgs]) error {
	if w.deps.Store == nil {
		return nil
	}
	cutoff := time.Now().Add(-autoReviewStuckAfter)
	stuck, err := w.deps.Store.Schematics.ListStuckAutoReview(ctx, cutoff, autoReviewBackfillBatch)
	if err != nil {
		return fmt.Errorf("auto_review backfill: list: %w", err)
	}
	if len(stuck) == 0 {
		return nil
	}
	client, err := river.ClientFromContextSafely[pgx.Tx](ctx)
	if err != nil {
		return fmt.Errorf("auto_review backfill: no river client: %w", err)
	}

	enqueued := 0
	for _, s := range stuck {
		if ctx.Err() != nil {
			break
		}
		args := ModerationArgs{
			SchematicID: s.ID,
			Title:       s.Title,
			Description: s.Content,
			ImageURL:    s.FeaturedImage,
			Slug:        s.Name,
		}
		// Dedupe against an already-pending moderation job for the same content so
		// overlapping sweeps don't double the OpenAI spend.
		if _, err := client.Insert(ctx, args, &river.InsertOpts{
			UniqueOpts: river.UniqueOpts{ByArgs: true},
		}); err != nil {
			slog.Warn("auto_review backfill: enqueue failed", "schematic_id", s.ID, "error", err)
			continue
		}
		enqueued++
	}
	slog.Info("auto_review backfill batch complete", "enqueued", enqueued, "batch", len(stuck))

	// A full batch means more are probably waiting; drain now rather than waiting
	// for the next periodic sweep.
	if len(stuck) == autoReviewBackfillBatch && ctx.Err() == nil {
		chainBackfill(ctx, AutoReviewBackfillArgs{})
	}
	return nil
}
