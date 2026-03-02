package pages

import (
	"createmod/internal/cache"
	"createmod/internal/nbtparser"
	"createmod/internal/store"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// APIUploadHandler serves POST /api/schematics/upload as a JSON API for uploading schematics.
// Requires API key authentication. Accepts multipart/form-data with an .nbt file.
// The upload goes through the same pipeline as web uploads — returns a preview token, not a published schematic.
func APIUploadHandler(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		const endpoint = "POST /api/schematics/upload"

		keyID, err := requireAPIKey(app, e)
		if err != nil {
			return nil
		}
		success := true
		defer func() { recordAPIKeyUsage(app, keyID, endpoint, !success) }()

		if ok, retry := rateLimitAllow(cacheService, keyID, 120); !ok {
			success = false
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20)

		// Read file from form (field name "file" or "nbt")
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			file, header, err = e.Request.FormFile("nbt")
			if err != nil {
				success = false
				return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'file' or 'nbt')"})
			}
		}
		if file != nil {
			defer file.Close()
		}

		// Validate filename
		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			success = false
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		if header.Size > maxUploadSize {
			success = false
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Read file data
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			success = false
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			success = false
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Validate NBT
		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			success = false
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": msg})
		}

		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection (skip in dev mode)
		isDev := os.Getenv("DEV") == "true"
		if !isDev {
			dupMsg := "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation."
			if coll, err := app.FindCollectionByNameOrId("temp_uploads"); err == nil && coll != nil {
				recs, err := app.FindRecordsByFilter(coll.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
				if err == nil && len(recs) > 0 {
					success = false
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
			if schemColl, err := app.FindCollectionByNameOrId("schematics"); err == nil && schemColl != nil {
				recs, err := app.FindRecordsByFilter(schemColl.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
				if err == nil && len(recs) > 0 {
					success = false
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}

			tempUploadStore.RLock()
			for _, entry := range tempUploadStore.m {
				if entry.Checksum == checksum {
					tempUploadStore.RUnlock()
					success = false
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
			tempUploadStore.RUnlock()
		}

		// Persist hash to nbt_hashes
		if coll, err := app.FindCollectionByNameOrId("nbt_hashes"); err == nil && coll != nil {
			rec := core.NewRecord(coll)
			rec.Set("checksum", checksum)
			_ = app.Save(rec)
		}

		// Generate token
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			success = false
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to generate preview token"})
		}
		token := hex.EncodeToString(buf)

		// Parse summary
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = fmt.Sprintf("size=%d bytes; nbt=unparsed", n)
		}

		// Extract stats
		blockCount, legacyMaterials, _ := nbtparser.ExtractStats(data)
		parsedMaterials, _ := nbtparser.ExtractMaterials(data)
		dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)

		// Extract mod namespaces
		modSet := make(map[string]struct{})
		for _, m := range parsedMaterials {
			parts := strings.SplitN(m.BlockID, ":", 2)
			if len(parts) == 2 && parts[0] != "minecraft" && parts[0] != "" {
				modSet[parts[0]] = struct{}{}
			}
		}
		mods := make([]string, 0, len(modSet))
		for mod := range modSet {
			mods = append(mods, mod)
		}

		// Store in-memory
		tempUploadStore.Lock()
		tempUploadStore.m[token] = tempUpload{
			Filename:        header.Filename,
			Size:            n,
			Checksum:        checksum,
			UploadedAt:      time.Now(),
			ParsedSummary:   parsed,
			BlockCount:      blockCount,
			Materials:       legacyMaterials,
			ParsedMaterials: parsedMaterials,
			DimX:            dimX,
			DimY:            dimY,
			DimZ:            dimZ,
			Mods:            mods,
			NBTData:         data,
		}
		entry := tempUploadStore.m[token]
		tempUploadStore.Unlock()

		// Persist to PocketBase
		recID, storedName := persistTempUploadPB(app, token, entry)
		if recID != "" {
			tempUploadStore.Lock()
			e2 := tempUploadStore.m[token]
			e2.PBRecordID = recID
			e2.NBTStoredName = storedName
			e2.NBTData = nil
			tempUploadStore.m[token] = e2
			tempUploadStore.Unlock()
		}

		// Build response
		resp := uploadNBTResponse{
			Token:      token,
			URL:        "/u/" + token,
			Checksum:   checksum,
			Filename:   header.Filename,
			Size:       n,
			BlockCount: blockCount,
			Materials:  parsedMaterials,
			Mods:       mods,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ
		if recID != "" && storedName != "" {
			resp.FileURL = "/api/files/temp_uploads/" + recID + "/" + storedName
		}
		if resp.Materials == nil {
			resp.Materials = []nbtparser.Material{}
		}
		if resp.Mods == nil {
			resp.Mods = []string{}
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}
