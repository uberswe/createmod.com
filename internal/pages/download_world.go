package pages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"createmod/internal/ratelimit"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// worldCacheKey mirrors the _conv cache scheme: the Updated timestamp keys
// out stale copies after edits; TTL cleanup reclaims them.
func worldCacheKey(schematicID string, updated time.Time) string {
	return fmt.Sprintf("_worlds/v1/%s/%d/flat.zip", schematicID, updated.Unix())
}

// worldGenRateLimitAllow throttles world generation harder than plain
// downloads: it runs in the request path and is CPU/memory heavy.
func worldGenRateLimitAllow(rl ratelimit.Limiter, clientIP string) bool {
	if clientIP == "" || rl == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, _ := rl.Allow(ctx, "worldgen:"+clientIP, 5, 10*time.Minute)
	return ok
}

// serveWorldExport streams the schematic as a ready-to-play world zip,
// generate-once-then-cache.
func serveWorldExport(e *server.RequestEvent, rl ratelimit.Limiter, storageSvc *storage.Service, s *store.Schematic) error {
	if storageSvc == nil {
		return e.String(http.StatusServiceUnavailable, "file storage is not available")
	}
	if ok, reason := schematic.CanExportWorld([3]int{s.DimX, s.DimY, s.DimZ}); !ok {
		return e.String(http.StatusUnprocessableEntity, "too large for world export ("+reason+") — download the schematic instead")
	}
	primary := strings.TrimSpace(s.SchematicFile)
	if primary == "" {
		return e.String(http.StatusNotFound, "schematic has no file")
	}
	worldName := sanitizeFilename(s.Name)
	if worldName == "" {
		worldName = "schematic_world"
	}
	outName := sanitizeContentDispositionFilename(worldName + "_world.zip")
	ctx := e.Request.Context()

	serveZip := func(data []byte, warnings []schematic.Warning) error {
		if len(warnings) > 0 {
			if wj, err := json.Marshal(warnings); err == nil {
				e.Response.Header().Set("X-Conversion-Warnings", string(wj))
			}
		}
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+outName+"\"")
		return e.Blob(http.StatusOK, "application/zip", data)
	}

	key := worldCacheKey(s.ID, s.Updated)
	if exists, _ := storageSvc.ExistsRaw(ctx, key); exists {
		if reader, err := storageSvc.DownloadRaw(ctx, key); err == nil {
			defer reader.Close()
			if data, err := io.ReadAll(reader); err == nil {
				return serveZip(data, nil)
			}
		}
	}

	// Cache miss: generation is the expensive path — rate limit it.
	if !worldGenRateLimitAllow(rl, e.RealIP()) {
		e.Response.Header().Set("Retry-After", "600")
		return e.String(http.StatusTooManyRequests, "world generation limit reached, try again in a few minutes")
	}

	src, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, primary)
	if err != nil {
		return e.String(http.StatusNotFound, "schematic file not found")
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, maxUploadSize+1))
	if err != nil || int64(len(data)) > maxUploadSize {
		return e.String(http.StatusInternalServerError, "failed to read schematic file")
	}
	model, err := schematic.ReadStructureNBT(data)
	if err != nil {
		return e.String(http.StatusUnprocessableEntity, "this schematic cannot be exported: "+convertUserError(err))
	}

	var buf bytes.Buffer
	warnings, err := schematic.WriteWorld(model, worldName, &buf)
	if err != nil {
		return e.String(http.StatusUnprocessableEntity, convertUserError(err))
	}

	out := buf.Bytes()
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		_ = storageSvc.UploadRawBytes(bgCtx, key, out, "application/zip")
	}()
	return serveZip(out, warnings)
}

// isDownloadFormatSlug reports whether slug names a downloadable output:
// any converter format, or the world export.
func isDownloadFormatSlug(slug string) bool {
	if slug == "world" {
		return true
	}
	_, _, ok := convertFormatBySlug(slug)
	return ok
}

// serveGeneratorWorld converts freshly generated NBT into a world zip.
func serveGeneratorWorld(e *server.RequestEvent, nbtData []byte, baseFilename string) error {
	model, err := schematic.ReadStructureNBT(nbtData)
	if err != nil {
		return e.InternalServerError("failed to parse generated schematic", nil)
	}
	worldName := sanitizeFilename(strings.TrimSuffix(filepath.Base(baseFilename), filepath.Ext(baseFilename)))
	if worldName == "" {
		worldName = "generated_world"
	}
	var buf bytes.Buffer
	warnings, err := schematic.WriteWorld(model, worldName, &buf)
	if err != nil {
		return e.BadRequestError(convertUserError(err), nil)
	}
	if len(warnings) > 0 {
		if wj, jErr := json.Marshal(warnings); jErr == nil {
			e.Response.Header().Set("X-Conversion-Warnings", string(wj))
		}
	}
	e.Response.Header().Set("Content-Type", "application/zip")
	e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeContentDispositionFilename(worldName+"_world.zip")))
	e.Response.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	_, writeErr := e.Response.Write(buf.Bytes())
	return writeErr
}
