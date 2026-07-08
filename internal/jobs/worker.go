// Package jobs provides River-based distributed job processing,
// replacing the goroutine tickers used with PocketBase.
package jobs

import (
	"context"
	"createmod/internal/aidescription"
	"createmod/internal/cache"
	"createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/modmeta"
	"createmod/internal/pointlog"
	"createmod/internal/search"
	"createmod/internal/session"
	"createmod/internal/sitemap"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meilisearch/meilisearch-go"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Deps holds service dependencies shared by all job workers.
type Deps struct {
	Store              *store.Store
	Storage            *storage.Service
	Search             *search.Service
	Cache              *cache.Service
	Sitemap            *sitemap.Service
	AIDesc             *aidescription.Service
	Translation        *translation.Service
	PointLog           *pointlog.Service
	ModMeta            *modmeta.Service
	SessionStore       *session.Store
	Moderation         *moderation.Service
	Mail               *mailer.Service
	MeiliClient        meilisearch.ServiceManager
	TwitchClientID     string
	TwitchClientSecret string
}

// Config holds job worker configuration.
type Config struct {
	Pool *pgxpool.Pool
	Deps Deps
}

// Worker wraps the River client for job processing.
type Worker struct {
	client *river.Client[pgx.Tx]
	pool   *pgxpool.Pool
}

// New creates a new job worker with River.
func New(ctx context.Context, cfg Config) (*Worker, error) {
	workers := river.NewWorkers()

	// Register all job workers with dependencies
	river.AddWorker(workers, &SearchIndexWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TrendingWorker{deps: cfg.Deps})
	river.AddWorker(workers, &AIDescriptionWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TranslationWorker{deps: cfg.Deps})
	river.AddWorker(workers, &PointLogWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ModMetadataWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SitemapWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SessionCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SafetyScanWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SafetyBackfillWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ConvCacheCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TempUploadCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ModerationWorker{deps: cfg.Deps})
	river.AddWorker(workers, &CommentModerationWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SearchCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SearchIndexUpsertWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SearchIndexDeleteWorker{deps: cfg.Deps})
	river.AddWorker(workers, &AnalyticsCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &BadgeRecalculationWorker{deps: cfg.Deps})
	river.AddWorker(workers, &NotificationCleanupWorker{deps: cfg.Deps})
	river.AddWorker(workers, &NotificationEmailDigestWorker{deps: cfg.Deps})
	river.AddWorker(workers, &RedditMetadataWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ReferenceMetadataWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ModpackSyncWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TwitchLiveCheckWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TwitchStreamSearchWorker{deps: cfg.Deps})
	river.AddWorker(workers, &PatreonMembershipWorker{deps: cfg.Deps})
	river.AddWorker(workers, &TrendingNewsletterWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SearchAlertCheckWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SectionUpdateWorker{deps: cfg.Deps})
	river.AddWorker(workers, &ZeroResultAnalysisWorker{deps: cfg.Deps})
	river.AddWorker(workers, &PointBackfillWorker{deps: cfg.Deps})
	river.AddWorker(workers, &SearchStatsFilterWorker{deps: cfg.Deps})
	river.AddWorker(workers, &FeedOGImageWorker{deps: cfg.Deps})
	river.AddWorker(workers, &AdClickRollupWorker{deps: cfg.Deps})

	riverClient, err := river.NewClient(riverpgxv5.New(cfg.Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"search":           {MaxWorkers: 1}, // serialized search index updates
			"ai":               {MaxWorkers: 2}, // AI description + translation
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SearchIndexArgs{}, &river.InsertOpts{
						Queue:      "search",
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return TrendingArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(15*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return SafetyBackfillArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 15 * time.Minute},
					}
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ConvCacheCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(30*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return AIDescriptionArgs{}, &river.InsertOpts{
						Queue:      "ai",
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Minute},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(30*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TranslationArgs{}, &river.InsertOpts{
						Queue:      "ai",
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Minute},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return PointLogArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ModMetadataArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 6 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SessionCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(10*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TempUploadCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 10 * time.Minute},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SitemapArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SearchCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return AnalyticsCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return BadgeRecalculationArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return NotificationCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return NotificationEmailDigestArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return RedditMetadataArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 6 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ReferenceMetadataArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ModpackSyncArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(5*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TwitchLiveCheckArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 5 * time.Minute},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(5*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TwitchStreamSearchArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 5 * time.Minute},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return PatreonMembershipArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 6 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return TrendingNewsletterArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SearchAlertCheckArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 1 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SectionUpdateArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ZeroResultAnalysisArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SearchStatsFilterArgs{}, &river.InsertOpts{
						Queue:      "ai",
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return FeedOGImageArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return AdClickRollupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 24 * time.Hour},
					}
				},
				nil,
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating river client: %w", err)
	}

	return &Worker{
		client: riverClient,
		pool:   cfg.Pool,
	}, nil
}

// Start begins processing jobs. Should be called after server startup.
func (w *Worker) Start(ctx context.Context) error {
	slog.Info("starting River job worker")
	return w.client.Start(ctx)
}

// Stop gracefully shuts down the job worker.
func (w *Worker) Stop(ctx context.Context) error {
	slog.Info("stopping River job worker")
	return w.client.Stop(ctx)
}

// Client returns the underlying River client for direct job insertion.
func (w *Worker) Client() *river.Client[pgx.Tx] {
	return w.client
}

// Insert enqueues a job for background processing.
func (w *Worker) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	_, err := w.client.Insert(ctx, args, opts)
	return err
}

// EnqueueSearchIndexUpsert enqueues an incremental search index update for a schematic.
// Safe to call with a nil Worker (no-ops gracefully).
func (w *Worker) EnqueueSearchIndexUpsert(ctx context.Context, schematicID string) {
	if w == nil || schematicID == "" {
		return
	}
	_, _ = w.client.Insert(ctx, SearchIndexUpsertArgs{SchematicID: schematicID}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Second},
	})
}

// EnqueueSearchIndexDelete enqueues removal of a schematic from the search index.
// Safe to call with a nil Worker (no-ops gracefully).
func (w *Worker) EnqueueSearchIndexDelete(ctx context.Context, schematicID string) {
	if w == nil || schematicID == "" {
		return
	}
	_, _ = w.client.Insert(ctx, SearchIndexDeleteArgs{SchematicID: schematicID}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Second},
	})
}
