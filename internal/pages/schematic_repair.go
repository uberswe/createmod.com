package pages

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"createmod/internal/nbtparser"
	"createmod/internal/storage"
	"createmod/internal/store"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
)

// RepairSchematics runs background integrity checks on all non-deleted schematics:
//  1. Validates that schematic files are valid NBT; attempts zip/tar extraction if not.
//  2. Generates missing material lists, block counts, dimensions, and mod lists.
//  3. Soft-deletes schematics whose files are missing, unreadable, or contain zero blocks.
//
// Designed to run in a background goroutine at startup.
func RepairSchematics(storageSvc *storage.Service, appStore *store.Store) {
	slog.Info("repair: starting schematic repair job")

	if storageSvc == nil {
		slog.Warn("repair: storage service not configured, skipping")
		return
	}

	ctx := context.Background()

	schematics, err := appStore.Schematics.ListAllForIndex(ctx)
	if err != nil {
		slog.Error("repair: failed to fetch schematics", "error", err)
		return
	}

	// Use the legacy PB collection ID prefix for existing S3 keys.
	collID := storage.CollectionPrefix("schematics")

	var repaired, deleted, skipped int
	for i := range schematics {
		switch repairSchematic(storageSvc, appStore, collID, &schematics[i]) {
		case "repaired":
			repaired++
		case "deleted":
			deleted++
		default:
			skipped++
		}
	}

	slog.Info("repair: schematic repair job complete",
		"total", len(schematics),
		"repaired", repaired,
		"deleted", deleted,
		"skipped", skipped,
	)
}

// repairSchematic checks a single schematic and returns "repaired", "deleted", or "skipped".
func repairSchematic(storageSvc *storage.Service, appStore *store.Store, collID string, s *store.Schematic) string {
	ctx := context.Background()

	fname := strings.TrimSpace(s.SchematicFile)
	if fname == "" {
		softDeleteRepair(appStore, s.ID, s.Name, "no schematic_file value")
		return "deleted"
	}

	// Check if the file exists in S3 before trying to download.
	// If it doesn't exist (e.g. local dev without file data, or mid-migration),
	// skip the schematic rather than soft-deleting it.
	exists, err := storageSvc.Exists(ctx, collID, s.ID, fname)
	if err != nil {
		slog.Debug("repair: S3 check error, skipping", "id", s.ID, "error", err)
		return "skipped"
	}
	if !exists {
		return "skipped"
	}

	// Read the file via direct S3 access.
	reader, err := storageSvc.Download(ctx, collID, s.ID, fname)
	if err != nil {
		softDeleteRepair(appStore, s.ID, s.Name, "file unreadable")
		return "deleted"
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || len(data) == 0 {
		softDeleteRepair(appStore, s.ID, s.Name, "file empty or read error")
		return "deleted"
	}

	// --- Phase 1: validate file is NBT (cheap header check) ---
	ok, _ := nbtparser.Validate(data)
	if !ok {
		nbtData, extracted := tryExtractNBT(data)
		if !extracted || len(nbtData) == 0 {
			softDeleteRepair(appStore, s.ID, s.Name, "invalid NBT and archive extraction failed")
			return "deleted"
		}
		// Replace the bad file with the extracted NBT via direct S3 upload.
		if uErr := storageSvc.UploadBytes(ctx, collID, s.ID, fname, nbtData, "application/octet-stream"); uErr != nil {
			slog.Warn("repair: could not upload extracted NBT", "id", s.ID, "error", uErr)
			softDeleteRepair(appStore, s.ID, s.Name, "could not upload extracted NBT file")
			return "deleted"
		}
		data = nbtData
		slog.Info("repair: extracted NBT from archive", "id", s.ID)
	}

	// --- Phase 2: fill missing stats (only parse if something is missing) ---
	hasMaterials := hasMaterialDataStore(s)
	hasBlockCount := s.BlockCount > 0
	hasDimensions := s.DimX > 0 || s.DimY > 0 || s.DimZ > 0

	if hasMaterials && hasBlockCount && hasDimensions {
		return "skipped"
	}

	needsSave := false

	// Materials + mods
	if !hasMaterials {
		materials, mErr := nbtparser.ExtractMaterials(data)
		if mErr == nil && len(materials) > 0 {
			if mjson, err := json.Marshal(materials); err == nil {
				s.Materials = mjson
				needsSave = true
			}
			// Derive mod namespaces
			modSet := make(map[string]struct{})
			for _, m := range materials {
				parts := strings.SplitN(m.BlockID, ":", 2)
				if len(parts) == 2 && parts[0] != "minecraft" && parts[0] != "" {
					modSet[parts[0]] = struct{}{}
				}
			}
			if len(modSet) > 0 {
				mods := make([]string, 0, len(modSet))
				for mod := range modSet {
					mods = append(mods, mod)
				}
				if mjson, err := json.Marshal(mods); err == nil {
					s.Mods = mjson
				}
			}
		}
	}

	// Block count
	if !hasBlockCount {
		blockCount, _, statsOk := nbtparser.ExtractStats(data)
		if statsOk && blockCount > 0 {
			s.BlockCount = blockCount
			needsSave = true
		} else if statsOk && blockCount == 0 {
			softDeleteRepair(appStore, s.ID, s.Name, "zero blocks")
			return "deleted"
		}
		// If !statsOk the parser failed; don't delete — could be a parser edge case.
	}

	// Dimensions
	if !hasDimensions {
		x, y, z, dimOk := nbtparser.ExtractDimensions(data)
		if dimOk && (x > 0 || y > 0 || z > 0) {
			s.DimX = x
			s.DimY = y
			s.DimZ = z
			needsSave = true
		}
	}

	if needsSave {
		if sErr := appStore.Schematics.Update(ctx, s); sErr != nil {
			slog.Warn("repair: save failed", "id", s.ID, "error", sErr)
		} else {
			return "repaired"
		}
	}
	return "skipped"
}

// hasMaterialDataStore returns true if the schematic already has a non-empty materials JSON array.
func hasMaterialDataStore(s *store.Schematic) bool {
	v := strings.TrimSpace(string(s.Materials))
	return v != "" && v != "[]" && v != "null"
}

func softDeleteRepair(appStore *store.Store, id, name, reason string) {
	slog.Info("repair: soft-deleting schematic", "id", id, "name", name, "reason", reason)
	ctx := context.Background()
	if err := appStore.Schematics.SoftDelete(ctx, id); err != nil {
		slog.Warn("repair: soft-delete failed", "id", id, "error", err)
	}
}

// ---------------------------------------------------------------------------
// Archive extraction helpers
// ---------------------------------------------------------------------------

const extractLimit = 50 << 20 // 50 MB per extracted file

// tryExtractNBT attempts to find a valid NBT file inside common archive formats.
func tryExtractNBT(data []byte) ([]byte, bool) {
	// Try zip archive
	if nbt, ok := extractFromZip(data); ok {
		return nbt, true
	}
	// Try tar.gz (gzip-compressed tar)
	if nbt, ok := extractFromTarGz(data); ok {
		return nbt, true
	}
	// Try raw tar
	if nbt, ok := extractFromTar(bytes.NewReader(data)); ok {
		return nbt, true
	}
	return nil, false
}

func extractFromZip(data []byte) ([]byte, bool) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, false
	}
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".nbt") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(io.LimitReader(rc, extractLimit))
			rc.Close()
			if err != nil {
				continue
			}
			if ok, _ := nbtparser.Validate(content); ok {
				return content, true
			}
		}
	}
	return nil, false
}

func extractFromTarGz(data []byte) ([]byte, bool) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return nil, false // not gzip
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer gz.Close()
	return extractFromTar(gz)
}

func extractFromTar(r io.Reader) ([]byte, bool) {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if strings.HasSuffix(strings.ToLower(hdr.Name), ".nbt") {
			content, err := io.ReadAll(io.LimitReader(tr, extractLimit))
			if err != nil {
				continue
			}
			if ok, _ := nbtparser.Validate(content); ok {
				return content, true
			}
		}
	}
	return nil, false
}
