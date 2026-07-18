package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"createmod/internal/schematic"
	"createmod/internal/storage"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

// SafetyScanArgs scans one schematic's file and stores its transparency
// manifest. Enqueued on publish/edit and by the periodic backfill.
type SafetyScanArgs struct {
	SchematicID string `json:"schematic_id"`
}

func (SafetyScanArgs) Kind() string { return "safety_scan" }

// SafetyScanWorker runs the tier-1 hardening checks plus the tier-2 content
// inspection for one schematic and upserts the schematic_safety row.
type SafetyScanWorker struct {
	river.WorkerDefaults[SafetyScanArgs]
	deps Deps
}

func (w *SafetyScanWorker) Work(ctx context.Context, job *river.Job[SafetyScanArgs]) error {
	if w.deps.Store == nil || w.deps.Storage == nil {
		slog.Warn("safety scan skipped: missing store or storage")
		return nil
	}
	return scanSchematicSafety(ctx, w.deps.Store, w.deps.Storage, job.Args.SchematicID)
}

// scanSchematicSafety loads the schematic's primary file, runs hardening +
// inspection, and persists the result. A failed parse is itself a result
// (file_safe=false), not a job failure.
func scanSchematicSafety(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, schematicID string) error {
	s, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || s == nil {
		slog.Warn("safety scan: schematic not found", "id", schematicID, "error", err)
		return nil
	}
	primary := strings.TrimSpace(s.SchematicFile)
	if primary == "" {
		return nil
	}
	reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, primary)
	if err != nil {
		return fmt.Errorf("safety scan: download %s: %w", schematicID, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, schematic.MaxDecompressedSize))
	if err != nil {
		return fmt.Errorf("safety scan: read %s: %w", schematicID, err)
	}
	sum := sha256.Sum256(data)

	result := &store.SchematicSafety{
		SchematicID:     schematicID,
		Checksum:        hex.EncodeToString(sum[:]),
		PipelineVersion: schematic.InspectorVersion,
	}

	// Tier 1: parse through the hardened readers. A file that fails here is
	// recorded as unsafe rather than erroring the job.
	format, err := schematic.Detect(data)
	if err == nil {
		var model *schematic.Schematic
		model, err = schematic.Read(data, format)
		if err == nil {
			result.FileSafe = true
			manifest := schematic.Inspect(model)
			if mj, jErr := json.Marshal(manifest); jErr == nil {
				result.Manifest = mj
			}
		}
	}
	if err != nil {
		result.FileSafe = false
		failure := map[string]interface{}{
			"inspectorVersion": schematic.InspectorVersion,
			"parseError":       strings.TrimPrefix(err.Error(), "schematic: "),
		}
		result.Manifest, _ = json.Marshal(failure)
	}

	if err := appStore.SchematicSafety.Upsert(ctx, result); err != nil {
		return fmt.Errorf("safety scan: persist %s: %w", schematicID, err)
	}
	return nil
}

// SafetyBackfillArgs sweeps for schematics that have never been scanned or
// were scanned by an older inspector version.
type SafetyBackfillArgs struct{}

func (SafetyBackfillArgs) Kind() string { return "safety_backfill" }

type SafetyBackfillWorker struct {
	river.WorkerDefaults[SafetyBackfillArgs]
	deps Deps
}

const safetyBackfillBatch = 200

func (w *SafetyBackfillWorker) Work(ctx context.Context, job *river.Job[SafetyBackfillArgs]) error {
	if w.deps.Store == nil || w.deps.Storage == nil {
		return nil
	}
	ids, err := w.deps.Store.SchematicSafety.ListNeedingScan(ctx, schematic.InspectorVersion, safetyBackfillBatch)
	if err != nil {
		return fmt.Errorf("safety backfill: list: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	scanned := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		if err := scanSchematicSafety(ctx, w.deps.Store, w.deps.Storage, id); err != nil {
			slog.Warn("safety backfill: scan failed", "id", id, "error", err)
			continue
		}
		scanned++
	}
	slog.Info("safety backfill batch complete", "scanned", scanned, "batch", len(ids))
	return nil
}
