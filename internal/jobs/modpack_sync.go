package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type ModpackSyncArgs struct{}

func (ModpackSyncArgs) Kind() string { return "modpack_sync" }

type ModpackSyncWorker struct {
	river.WorkerDefaults[ModpackSyncArgs]
	deps Deps
}

func (w *ModpackSyncWorker) Work(ctx context.Context, job *river.Job[ModpackSyncArgs]) error {
	slog.Info("modpack sync started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: query Modrinth API for modpacks including Create mod
	// GET https://api.modrinth.com/v2/search?facets=[["categories:create"]]&project_type=modpack
	// For each result, upsert into modpacks table
	slog.Info("modpack sync completed")
	return nil
}
