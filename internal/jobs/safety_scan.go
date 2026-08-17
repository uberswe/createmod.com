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

// fileScan is the tier-1 + tier-2 result for one stored file.
type fileScan struct {
	safe     bool
	manifest *schematic.Manifest
	checksum string
	parseErr string
}

// scanStoredFile downloads one schematic file and runs the hardened readers
// (tier 1) plus content inspection (tier 2). A parse failure is a result
// (safe=false), not an error; only a download/read failure returns an error so
// the job retries. This is the same gauntlet for every format.
func scanStoredFile(ctx context.Context, storageSvc *storage.Service, schematicID, filename string) (fileScan, error) {
	reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), schematicID, filename)
	if err != nil {
		return fileScan{}, fmt.Errorf("safety scan: download %s/%s: %w", schematicID, filename, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, schematic.MaxDecompressedSize))
	if err != nil {
		return fileScan{}, fmt.Errorf("safety scan: read %s/%s: %w", schematicID, filename, err)
	}
	sum := sha256.Sum256(data)
	fs := fileScan{checksum: hex.EncodeToString(sum[:])}

	format, derr := schematic.Detect(data)
	if derr == nil {
		var model *schematic.Schematic
		if model, derr = schematic.Read(data, format); derr == nil {
			fs.safe = true
			fs.manifest = schematic.Inspect(model)
			return fs, nil
		}
	}
	fs.parseErr = strings.TrimPrefix(derr.Error(), "schematic: ")
	return fs, nil
}

// scanSchematicSafety runs the safety gauntlet over every downloadable file for
// a schematic — the normalized .nbt and, when the upload was converted, the
// preserved original — and persists the combined result. The schematic is only
// file_safe when BOTH pass hardening; the manifest reports the union of their
// findings, so an original that hides something the .nbt dropped (or is itself
// malformed) can never be presented as safe. A failed parse is a result
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

	prim, err := scanStoredFile(ctx, storageSvc, s.ID, primary)
	if err != nil {
		return err
	}

	fileSafe := prim.safe
	manifest := prim.manifest
	parseErr := prim.parseErr
	checksum := prim.checksum

	// The preserved original is downloadable too, so it must clear the same
	// checks; formats with no writer (e.g. Aeronautics .excraft) are only ever
	// served as this original.
	if orig := strings.TrimSpace(s.OriginalFile); orig != "" {
		os, oerr := scanStoredFile(ctx, storageSvc, s.ID, orig)
		if oerr != nil {
			// Can't verify a file we serve — retry rather than claim safe.
			return oerr
		}
		checksum = prim.checksum + ":" + os.checksum
		fileSafe = prim.safe && os.safe
		if os.safe {
			manifest = mergeManifests(manifest, os.manifest)
		} else if parseErr == "" {
			parseErr = "original upload (" + orig + "): " + os.parseErr
		}
	}

	result := &store.SchematicSafety{
		SchematicID:     schematicID,
		Checksum:        checksum,
		PipelineVersion: schematic.InspectorVersion,
		FileSafe:        fileSafe,
	}
	if fileSafe && manifest != nil {
		result.Manifest, _ = json.Marshal(manifest)
	} else {
		result.Manifest, _ = json.Marshal(map[string]interface{}{
			"inspectorVersion": schematic.InspectorVersion,
			"parseError":       parseErr,
		})
	}

	if err := appStore.SchematicSafety.Upsert(ctx, result); err != nil {
		return fmt.Errorf("safety scan: persist %s: %w", schematicID, err)
	}
	return nil
}

// mergeManifests combines two file manifests into the worst case: counts are
// the per-type maximum (the files describe the same build, so summing would
// double-count), findings are the deduplicated union capped at MaxFindings, and
// mod namespaces are unioned. Either argument may be nil.
func mergeManifests(a, b *schematic.Manifest) *schematic.Manifest {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := &schematic.Manifest{
		InspectorVersion:  schematic.InspectorVersion,
		Counts:            map[schematic.FindingType]int{},
		FindingsTruncated: a.FindingsTruncated || b.FindingsTruncated,
	}
	for t, n := range a.Counts {
		out.Counts[t] = n
	}
	for t, n := range b.Counts {
		if n > out.Counts[t] {
			out.Counts[t] = n
		}
	}
	seen := map[schematic.Finding]bool{}
	for _, src := range [][]schematic.Finding{a.Findings, b.Findings} {
		for _, f := range src {
			if seen[f] {
				continue
			}
			seen[f] = true
			if len(out.Findings) >= schematic.MaxFindings {
				out.FindingsTruncated = true
				break
			}
			out.Findings = append(out.Findings, f)
		}
	}
	nsSeen := map[string]bool{}
	for _, ns := range append(append([]string{}, a.ModNamespaces...), b.ModNamespaces...) {
		if !nsSeen[ns] {
			nsSeen[ns] = true
			out.ModNamespaces = append(out.ModNamespaces, ns)
		}
	}
	if len(out.Counts) == 0 {
		out.Counts = nil
	}
	return out
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
	// A full batch means more work is probably waiting — drain it now
	// rather than one batch per periodic sweep.
	if len(ids) == safetyBackfillBatch && ctx.Err() == nil {
		chainBackfill(ctx, SafetyBackfillArgs{})
	}
	return nil
}
