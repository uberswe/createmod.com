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
func APISchematicDownloadHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, modSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}/download"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, modSecret)
		if err != nil {
			return nil
		}
		if rejected := applyAPIRateLimit(e, rl, keyID, isHMAC); rejected {
			return nil
		}
		if !isHMAC {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
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

		// Count the download (best-effort, IP-deduped).
		countSchematicDownloadStore(appStore, s.ID, e.RealIP(), rl, cacheService)

		// Variation file download.
		if fileID := e.Request.URL.Query().Get("f"); fileID != "" {
			sf, sfErr := appStore.SchematicFiles.GetByID(ctx, fileID)
			if sfErr != nil || sf == nil || sf.SchematicID != s.ID {
				return writeJSON(e, http.StatusNotFound, map[string]string{"error": "variation file not found"})
			}
			return e.Redirect(http.StatusFound, fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, url.PathEscape(sf.Filename)))
		}

		// Paid schematics are only available via the external link on the page.
		if s.Paid {
			return writeJSON(e, http.StatusForbidden, map[string]string{"error": "this schematic is paid; use the external link on the schematic page"})
		}

		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic file not found"})
		}
		return e.Redirect(http.StatusFound, fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, url.PathEscape(primary)))
	}
}
