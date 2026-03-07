// Package jobs provides River-based distributed job processing,
// replacing the goroutine tickers used with PocketBase.
package jobs

import (
	"context"
	"createmod/internal/aidescription"
	"createmod/internal/cache"
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
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Deps holds service dependencies shared by all job workers.
type Deps struct {
	Store        *store.Store
	Storage      *storage.Service
	Search       *search.Service
	Cache        *cache.Service
	Sitemap      *sitemap.Service
	AIDesc       *aidescription.Service
	Translation  *translation.Service
	PointLog     *pointlog.Service
	ModMeta      *modmeta.Service
	SessionStore *session.Store
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

	riverClient, err := river.NewClient(riverpgxv5.New(cfg.Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"search":           {MaxWorkers: 1}, // serialized search index updates
			"ai":               {MaxWorkers: 2}, // AI description + translation
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(10*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return SearchIndexArgs{}, &river.InsertOpts{
						Queue:      "search",
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(10*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TrendingArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(30*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return AIDescriptionArgs{}, &river.InsertOpts{
						Queue:      "ai",
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(30*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return TranslationArgs{}, &river.InsertOpts{
						Queue:      "ai",
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return PointLogArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ModMetadataArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true},
					}
				},
				nil,
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SessionCleanupArgs{}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true},
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
