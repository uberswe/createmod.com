package pages

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/store"

	"createmod/internal/server"
)

// DownloadHandler redirects to the schematic file and increments a download counter.
// Requires a valid one-time token (?t=) issued by the interstitial page.
func DownloadHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return e.String(http.StatusBadRequest, "missing name")
		}
		// validate one-time token
		token := e.Request.URL.Query().Get("t")
		if token == "" {
			return e.String(http.StatusForbidden, "missing download token; please open the download page again")
		}

		// Atomically consume the token from PostgreSQL
		dt, err := appStore.DownloadTokens.Consume(context.Background(), token)
		if err != nil || dt == nil || dt.Name != name {
			return e.String(http.StatusForbidden, "invalid or expired download token; please open the download page again")
		}

		s, err := appStore.Schematics.GetByName(context.Background(), name)
		if err != nil || s == nil || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return e.String(http.StatusNotFound, "schematic not found")
		}

		// Block site download for paid schematics
		if s.Paid {
			return e.String(http.StatusForbidden, "This schematic is paid; please use the external link on the schematic page.")
		}

		// Block download for blacklisted schematics
		if s.Blacklisted {
			return e.String(http.StatusForbidden, "This schematic has been blacklisted and cannot be downloaded.")
		}

		// Increment download counter (best-effort, IP-deduped)
		countSchematicDownloadStore(appStore, s.ID, e.RealIP(), cacheService)

		// Single file redirect
		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" {
			return e.String(http.StatusNotFound, "schematic file not found")
		}
		base := "schematics/" + s.ID
		fileURL := fmt.Sprintf("/api/files/%s/%s", base, primary)
		return e.Redirect(http.StatusFound, fileURL)
	}
}

// countSchematicDownloadStore increments download counters via the PostgreSQL store.
// clientIP and cacheService are used for IP-based rate limiting.
func countSchematicDownloadStore(appStore *store.Store, schematicID string, clientIP string, cacheService *cache.Service) {
	// IP-based rate limiting: skip if same IP already downloaded this schematic recently
	if clientIP != "" && cacheService != nil {
		ipKey := fmt.Sprintf("dlip:%s:%s", clientIP, schematicID)
		if _, already := cacheService.Get(ipKey); already {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
	}

	if err := appStore.ViewRatings.RecordDownload(context.Background(), schematicID, nil); err != nil {
		return
	}
	// Update cache with new total
	if total, err := appStore.ViewRatings.GetDownloadCount(context.Background(), schematicID); err == nil {
		cacheService.SetInt(cache.DownloadKey(schematicID), total)
	}
}
