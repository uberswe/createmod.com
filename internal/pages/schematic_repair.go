package pages

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"createmod/internal/nbtparser"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// RepairSchematics runs background integrity checks on all non-deleted schematics:
//  1. Validates that schematic files are valid NBT; attempts zip/tar extraction if not.
//  2. Generates missing material lists, block counts, dimensions, and mod lists.
//  3. Soft-deletes schematics whose files are missing, unreadable, or contain zero blocks.
//
// Designed to run in a background goroutine at startup.
func RepairSchematics(app *pocketbase.PocketBase) {
	app.Logger().Info("repair: starting schematic repair job")

	records, err := app.FindRecordsByFilter("schematics", "deleted = ''", "-created", -1, 0)
	if err != nil {
		app.Logger().Error("repair: failed to fetch schematics", "error", err)
		return
	}

	var repaired, deleted, skipped int
	for _, rec := range records {
		switch repairSchematic(app, rec) {
		case "repaired":
			repaired++
		case "deleted":
			deleted++
		default:
			skipped++
		}
	}

	app.Logger().Info("repair: schematic repair job complete",
		"total", len(records),
		"repaired", repaired,
		"deleted", deleted,
		"skipped", skipped,
	)
}

// repairSchematic checks a single schematic record and returns "repaired", "deleted", or "skipped".
func repairSchematic(app *pocketbase.PocketBase, rec *core.Record) string {
	fname := strings.TrimSpace(rec.GetString("schematic_file"))
	if fname == "" {
		softDeleteSchematic(app, rec, "no schematic_file value")
		return "deleted"
	}

	// Read the file via PocketBase's filesystem (supports S3 and local storage).
	fileKey := rec.BaseFilesPath() + "/" + fname
	fsys, err := app.NewFilesystem()
	if err != nil {
		app.Logger().Warn("repair: could not create filesystem", "id", rec.Id, "error", err)
		return "skipped"
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(fileKey)
	if err != nil {
		softDeleteSchematic(app, rec, "file missing or unreadable")
		return "deleted"
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || len(data) == 0 {
		softDeleteSchematic(app, rec, "file empty or read error")
		return "deleted"
	}

	// --- Phase 1: validate file is NBT (cheap header check) ---
	ok, _ := nbtparser.Validate(data)
	if !ok {
		nbtData, extracted := tryExtractNBT(data)
		if !extracted || len(nbtData) == 0 {
			softDeleteSchematic(app, rec, "invalid NBT and archive extraction failed")
			return "deleted"
		}
		// Replace the bad file with the extracted NBT via PocketBase's file system.
		newFile, fErr := filesystem.NewFileFromBytes(nbtData, fname)
		if fErr != nil {
			app.Logger().Warn("repair: could not create file from bytes", "id", rec.Id, "error", fErr)
			softDeleteSchematic(app, rec, "could not create replacement NBT file")
			return "deleted"
		}
		rec.Set("schematic_file", newFile)
		if sErr := app.Save(rec); sErr != nil {
			app.Logger().Warn("repair: could not save extracted NBT", "id", rec.Id, "error", sErr)
			softDeleteSchematic(app, rec, "could not save extracted NBT")
			return "deleted"
		}
		data = nbtData
		app.Logger().Info("repair: extracted NBT from archive", "id", rec.Id)
	}

	// --- Phase 2: fill missing stats (only parse if something is missing) ---
	hasMaterials := hasMaterialData(rec)
	hasBlockCount := rec.GetInt("block_count") > 0
	hasDimensions := rec.GetInt("dim_x") > 0 || rec.GetInt("dim_y") > 0 || rec.GetInt("dim_z") > 0

	if hasMaterials && hasBlockCount && hasDimensions {
		return "skipped"
	}

	needsSave := false

	// Materials + mods
	if !hasMaterials {
		materials, mErr := nbtparser.ExtractMaterials(data)
		if mErr == nil && len(materials) > 0 {
			if mjson, err := json.Marshal(materials); err == nil {
				rec.Set("materials", string(mjson))
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
					rec.Set("mods", string(mjson))
				}
			}
		}
	}

	// Block count
	if !hasBlockCount {
		blockCount, _, statsOk := nbtparser.ExtractStats(data)
		if statsOk && blockCount > 0 {
			rec.Set("block_count", blockCount)
			needsSave = true
		} else if statsOk && blockCount == 0 {
			softDeleteSchematic(app, rec, "zero blocks")
			return "deleted"
		}
		// If !statsOk the parser failed; don't delete — could be a parser edge case.
	}

	// Dimensions
	if !hasDimensions {
		x, y, z, dimOk := nbtparser.ExtractDimensions(data)
		if dimOk && (x > 0 || y > 0 || z > 0) {
			rec.Set("dim_x", x)
			rec.Set("dim_y", y)
			rec.Set("dim_z", z)
			needsSave = true
		}
	}

	if needsSave {
		if sErr := app.Save(rec); sErr != nil {
			app.Logger().Warn("repair: save failed", "id", rec.Id, "error", sErr)
		} else {
			return "repaired"
		}
	}
	return "skipped"
}

// hasMaterialData returns true if the record already has a non-empty materials JSON array.
func hasMaterialData(rec *core.Record) bool {
	v := strings.TrimSpace(rec.GetString("materials"))
	return v != "" && v != "[]" && v != "null"
}

func softDeleteSchematic(app *pocketbase.PocketBase, rec *core.Record, reason string) {
	app.Logger().Info("repair: soft-deleting schematic", "id", rec.Id, "name", rec.GetString("name"), "reason", reason)
	rec.Set("deleted", time.Now())
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("repair: soft-delete save failed", "id", rec.Id, "error", err)
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
