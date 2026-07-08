package jobs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"createmod/internal/schematic"
	"createmod/internal/storage"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

// FingerprintArgs computes the similarity fingerprint for one schematic.
type FingerprintArgs struct {
	SchematicID string `json:"schematic_id"`
}

func (FingerprintArgs) Kind() string { return "fingerprint" }

type FingerprintWorker struct {
	river.WorkerDefaults[FingerprintArgs]
	deps Deps
}

func (w *FingerprintWorker) Work(ctx context.Context, job *river.Job[FingerprintArgs]) error {
	if w.deps.Store == nil || w.deps.Storage == nil {
		return nil
	}
	return computeSchematicFingerprint(ctx, w.deps.Store, w.deps.Storage, job.Args.SchematicID)
}

func computeSchematicFingerprint(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, schematicID string) error {
	s, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || s == nil {
		slog.Warn("fingerprint: schematic not found", "id", schematicID, "error", err)
		return nil
	}
	primary := strings.TrimSpace(s.SchematicFile)
	if primary == "" {
		return nil
	}
	reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, primary)
	if err != nil {
		return fmt.Errorf("fingerprint: download %s: %w", schematicID, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, schematic.MaxDecompressedSize))
	if err != nil {
		return fmt.Errorf("fingerprint: read %s: %w", schematicID, err)
	}

	format, err := schematic.Detect(data)
	if err != nil {
		// Unparseable files still get a row (empty fp, current version) so
		// the backfill doesn't retry them forever.
		return appStore.Fingerprints.Upsert(ctx, &store.SchematicFingerprint{
			SchematicID: schematicID, Version: schematic.FingerprintVersion,
		})
	}
	model, err := schematic.Read(data, format)
	if err != nil {
		return appStore.Fingerprints.Upsert(ctx, &store.SchematicFingerprint{
			SchematicID: schematicID, Version: schematic.FingerprintVersion,
		})
	}
	fp := schematic.ComputeFingerprint(model)
	encoded, err := schematic.EncodeFingerprint(fp)
	if err != nil {
		return fmt.Errorf("fingerprint: encode %s: %w", schematicID, err)
	}
	return appStore.Fingerprints.Upsert(ctx, &store.SchematicFingerprint{
		SchematicID: schematicID,
		FP:          encoded,
		Version:     schematic.FingerprintVersion,
	})
}

// FingerprintBackfillArgs sweeps for schematics missing a current-version
// fingerprint (never computed, older version, or edited since).
type FingerprintBackfillArgs struct{}

func (FingerprintBackfillArgs) Kind() string { return "fingerprint_backfill" }

type FingerprintBackfillWorker struct {
	river.WorkerDefaults[FingerprintBackfillArgs]
	deps Deps
}

const fingerprintBackfillBatch = 200

func (w *FingerprintBackfillWorker) Work(ctx context.Context, job *river.Job[FingerprintBackfillArgs]) error {
	if w.deps.Store == nil || w.deps.Storage == nil {
		return nil
	}
	ids, err := w.deps.Store.Fingerprints.ListNeedingCompute(ctx, schematic.FingerprintVersion, fingerprintBackfillBatch)
	if err != nil {
		return fmt.Errorf("fingerprint backfill: list: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	done := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		if err := computeSchematicFingerprint(ctx, w.deps.Store, w.deps.Storage, id); err != nil {
			slog.Warn("fingerprint backfill: compute failed", "id", id, "error", err)
			continue
		}
		done++
	}
	slog.Info("fingerprint backfill batch complete", "computed", done, "batch", len(ids))
	return nil
}
