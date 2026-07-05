package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// APISchematicDownloadHandler serves GET /api/schematics/{name}/download.
// It increments the download counter and redirects to the schematic's file so
// API clients (e.g. the Brassworks Launcher) can fetch the .nbt directly.
// Auth: API key or HMAC. An optional ?f={fileID} downloads a variation file.
func APISchematicDownloadHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}/download"
		key, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, cacheService)
		if err != nil {
			return nil
		}
		if rejected := applyAPIRateLimit(e, rl, key, isHMAC); rejected {
			return nil
		}
		if !isHMAC {
			defer func() { recordAPIKeyUsageStore(appStore, key.ID, endpoint) }()
		}

		name := e.Request.PathValue("name")
		if strings.TrimSpace(name) == "" {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing schematic name"})
		}

		ctx := context.Background()
		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil || s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "not found"})
		}

		// Variation file download. Validate before counting so a bad file ID
		// can't inflate the download counter.
		if fileID := e.Request.URL.Query().Get("f"); fileID != "" {
			sf, sfErr := appStore.SchematicFiles.GetByID(ctx, fileID)
			if sfErr != nil || sf == nil || sf.SchematicID != s.ID {
				return writeJSON(e, http.StatusNotFound, map[string]string{"error": "variation file not found"})
			}
			countSchematicDownloadStore(appStore, s.ID, e.RealIP(), rl, cacheService)
			return e.Redirect(http.StatusFound, fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, url.PathEscape(sf.Filename)))
		}

		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic file not found"})
		}
		// Count the download (best-effort, IP-deduped) only once we know we can serve it.
		countSchematicDownloadStore(appStore, s.ID, e.RealIP(), rl, cacheService)
		return e.Redirect(http.StatusFound, fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, url.PathEscape(primary)))
	}
}
