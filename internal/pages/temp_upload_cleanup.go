package pages

import (
	"time"
)

// housekeeping settings for temporary uploads
const tempUploadTTL = 2 * time.Hour
const tempUploadPurgeInterval = 10 * time.Minute

// init starts a lightweight background goroutine that periodically removes
// expired entries from the in-memory temporary upload store. This is only a
// stop-gap until we persist temp uploads in PocketBase.
func init() {
	go func() {
		t := time.NewTicker(tempUploadPurgeInterval)
		defer t.Stop()
		for range t.C {
			purgeExpiredTempUploads()
		}
	}()
}

func purgeExpiredTempUploads() {
	now := time.Now()
	tempUploadStore.Lock()
	for k, v := range tempUploadStore.m {
		if now.Sub(v.UploadedAt) > tempUploadTTL {
			delete(tempUploadStore.m, k)
		}
	}
	tempUploadStore.Unlock()
}
