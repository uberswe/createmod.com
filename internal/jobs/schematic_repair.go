package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"

	"github.com/riverqueue/river"
)

// SchematicRepairArgs are the arguments for the schematic repair job.
type SchematicRepairArgs struct{}

func (SchematicRepairArgs) Kind() string { return "schematic_repair" }

// SchematicRepairWorker validates and repairs schematics in the background.
type SchematicRepairWorker struct {
	river.WorkerDefaults[SchematicRepairArgs]
	deps Deps
}

func (w *SchematicRepairWorker) Work(ctx context.Context, job *river.Job[SchematicRepairArgs]) error {
	slog.Info("starting schematic repair job")
	if w.deps.Store == nil || w.deps.Storage == nil {
		slog.Warn("schematic repair skipped: missing dependencies")
		return nil
	}

	pages.RepairSchematics(w.deps.Storage, w.deps.Store)
	return nil
}
