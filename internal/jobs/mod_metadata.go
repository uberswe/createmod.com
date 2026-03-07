package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// ModMetadataArgs are the arguments for the mod metadata enrichment job.
type ModMetadataArgs struct{}

func (ModMetadataArgs) Kind() string { return "mod_metadata_enrich" }

// ModMetadataWorker fetches mod information from Modrinth and CurseForge.
type ModMetadataWorker struct {
	river.WorkerDefaults[ModMetadataArgs]
	deps Deps
}

func (w *ModMetadataWorker) Work(ctx context.Context, job *river.Job[ModMetadataArgs]) error {
	slog.Info("enriching mod metadata")
	if w.deps.ModMeta == nil {
		slog.Warn("mod metadata enrichment skipped: missing dependencies")
		return nil
	}

	w.deps.ModMeta.EnrichAll()
	slog.Info("mod metadata enrichment complete")
	return nil
}
