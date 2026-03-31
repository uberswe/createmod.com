package pages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/ratelimit"
	"createmod/internal/store"

	"createmod/internal/server"
)

// dailyDownloadLimit is the maximum number of schematic downloads per IP per day.
const dailyDownloadLimit = 100

// downloadRateLimitAllow checks whether the given IP has exceeded the daily download limit.
// Returns (allowed, retryAfterSeconds).
func downloadRateLimitAllow(rl ratelimit.Limiter, clientIP string) (bool, int) {
	if clientIP == "" || rl == nil {
		return true, 0
	}
	now := time.Now()
	dayKey := "dldaily:" + clientIP + ":" + now.Format("20060102")
	// TTL: time remaining until end of the current UTC day
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttl := time.Until(endOfDay)
	if ttl <= 0 {
		ttl = time.Second
	}
	ok, _ := rl.Allow(context.Background(), dayKey, dailyDownloadLimit, ttl)
	if !ok {
		ra := int(ttl.Seconds())
		if ra < 1 {
			ra = 1
		}
		return false, ra
	}
	return true, 0
}

// DownloadHandler redirects to the schematic file and increments a download counter.
// Requires a valid one-time token (?t=) issued by the interstitial page.
func DownloadHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return e.String(http.StatusBadRequest, "missing name")
		}

		// Global per-IP daily download rate limit
		if ok, retry := downloadRateLimitAllow(rl, e.RealIP()); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return e.String(http.StatusTooManyRequests, "Download limit reached. Please try again tomorrow.")
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

		// Block download for rejected or deleted schematics
		if s.ModerationState == store.ModerationRejected || s.ModerationState == store.ModerationDeleted {
			return e.String(http.StatusForbidden, "This schematic has been blocked and cannot be downloaded.")
		}

		// Increment download counter (best-effort, IP-deduped)
		countSchematicDownloadStore(appStore, s.ID, e.RealIP(), rl, cacheService)

		// Single file redirect
		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" {
			return e.String(http.StatusNotFound, "schematic file not found")
		}
		fileURL := fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, url.PathEscape(primary))
		return e.Redirect(http.StatusFound, fileURL)
	}
}

// countSchematicDownloadStore increments download counters via the PostgreSQL store.
// clientIP and rl are used for IP-based deduplication.
func countSchematicDownloadStore(appStore *store.Store, schematicID string, clientIP string, rl ratelimit.Limiter, cacheService *cache.Service) {
	// IP-based dedup: skip if same IP already downloaded this schematic recently
	if clientIP != "" && rl != nil {
		ipKey := fmt.Sprintf("dlip:%s:%s", clientIP, schematicID)
		if rl.Check(context.Background(), ipKey) {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		rl.Mark(context.Background(), ipKey, 1*time.Hour)
	}

	if err := appStore.ViewRatings.RecordDownload(context.Background(), schematicID, nil); err != nil {
		return
	}
	// Update cache with new total
	if cacheService != nil {
		if total, err := appStore.ViewRatings.GetDownloadCount(context.Background(), schematicID); err == nil {
			cacheService.SetInt(cache.DownloadKey(schematicID), total)
		}
	}
}
