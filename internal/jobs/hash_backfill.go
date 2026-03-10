package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"time"

	"createmod/internal/storage"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

// HashBackfillArgs are the arguments for the hash backfill job.
type HashBackfillArgs struct{}

func (HashBackfillArgs) Kind() string { return "hash_backfill" }

// HashBackfillWorker backfills validation hashes for schematics that don't
// have an entry in nbt_hashes. It processes schematics one at a time using
// keyset pagination and streaming SHA256 to keep memory usage low.
type HashBackfillWorker struct {
	river.WorkerDefaults[HashBackfillArgs]
	deps Deps
}

const hashBackfillBatchSize = 50

func (w *HashBackfillWorker) Work(ctx context.Context, job *river.Job[HashBackfillArgs]) error {
	if w.deps.Store == nil || w.deps.Storage == nil {
		slog.Warn("hash backfill skipped: missing dependencies")
		return nil
	}

	slog.Info("hash backfill: starting")

	collPrefix := storage.CollectionPrefix("schematics")
	var cursor string
	var total, hashed, skipped, errored int

	for {
		refs, err := w.deps.Store.Schematics.ListMissingHash(ctx, cursor, hashBackfillBatchSize)
		if err != nil {
			slog.Error("hash backfill: failed to list schematics", "error", err)
			return err
		}
		if len(refs) == 0 {
			break
		}

		for _, ref := range refs {
			total++
			cursor = ref.ID

			result := w.hashOne(ctx, collPrefix, ref.ID, ref.SchematicFile)
			switch result {
			case "hashed":
				hashed++
			case "skipped":
				skipped++
			default:
				errored++
			}

			// Gentle rate limiting to avoid hammering S3.
			time.Sleep(50 * time.Millisecond)
		}

		slog.Info("hash backfill: progress", "processed", total, "hashed", hashed, "skipped", skipped, "errors", errored)
	}

	slog.Info("hash backfill: complete", "total", total, "hashed", hashed, "skipped", skipped, "errors", errored)
	return nil
}

// hashOne downloads a single schematic file, computes its SHA256 hash via
// streaming (io.Copy into the hasher), and inserts into nbt_hashes.
func (w *HashBackfillWorker) hashOne(ctx context.Context, collPrefix, schematicID, filename string) string {
	reader, err := w.deps.Storage.Download(ctx, collPrefix, schematicID, filename)
	if err != nil {
		slog.Debug("hash backfill: download failed, skipping", "id", schematicID, "error", err)
		return "skipped"
	}

	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		reader.Close()
		slog.Warn("hash backfill: read failed", "id", schematicID, "error", err)
		return "error"
	}
	reader.Close()

	checksum := hex.EncodeToString(h.Sum(nil))

	id := generateBackfillID()
	if err := w.deps.Store.NBTHashes.Create(ctx, &store.NBTHash{
		ID:          id,
		Hash:        checksum,
		SchematicID: &schematicID,
	}); err != nil {
		slog.Warn("hash backfill: insert failed", "id", schematicID, "hash", checksum, "error", err)
		return "error"
	}

	return "hashed"
}

func generateBackfillID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:15]
}
