package pages

import (
	"context"
	"createmod/internal/store"
	"log/slog"
	"time"
)

// housekeeping settings for temporary uploads
const tempUploadTTL = 2 * time.Hour
const tempUploadPurgeInterval = 10 * time.Minute

// StartTempUploadCleanup launches a background goroutine that periodically
// removes expired temp uploads from PostgreSQL. Call this once at server startup.
func StartTempUploadCleanup(appStore *store.Store) {
	go func() {
		t := time.NewTicker(tempUploadPurgeInterval)
		defer t.Stop()
		for range t.C {
			cutoff := time.Now().Add(-tempUploadTTL)
			n, err := appStore.TempUploads.DeleteExpired(context.Background(), cutoff)
			if err != nil {
				slog.Error("failed to purge expired temp uploads", "error", err)
			} else if n > 0 {
				slog.Info("purged expired temp uploads", "count", n)
			}
		}
	}()
}
