package pages

import (
	"context"
	"createmod/internal/moderation"
	"createmod/internal/store"
	"log/slog"
	"os"
)

// moderateCollectionBanner runs the OpenAI image moderation check on a
// collection's banner asynchronously. If the image is flagged, the banner URL
// is cleared and the change is logged.
func moderateCollectionBanner(moderationSvc *moderation.Service, appStore *store.Store, collectionID, bannerURL string) {
	if moderationSvc == nil || bannerURL == "" {
		return
	}
	go func() {
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "https://createmod.com"
		}
		fullURL := baseURL + bannerURL

		result, err := moderationSvc.CheckImage(fullURL)
		if err != nil {
			slog.Warn("collection image moderation unavailable",
				"collection_id", collectionID, "error", err)
			return
		}
		if !result.Approved {
			slog.Warn("collection banner flagged by moderation, removing",
				"collection_id", collectionID, "reason", result.Reason, "url", bannerURL)
			ctx := context.Background()
			coll, getErr := appStore.Collections.GetByID(ctx, collectionID)
			if getErr != nil || coll == nil {
				slog.Error("collection image moderation: failed to load collection",
					"collection_id", collectionID, "error", getErr)
				return
			}
			coll.BannerURL = ""
			if updateErr := appStore.Collections.Update(ctx, coll); updateErr != nil {
				slog.Error("collection image moderation: failed to clear banner",
					"collection_id", collectionID, "error", updateErr)
			}
		}
	}()
}

// moderateGuideBanner runs the OpenAI image moderation check on a guide's
// banner asynchronously. If the image is flagged, the banner URL is cleared
// and the change is logged.
func moderateGuideBanner(moderationSvc *moderation.Service, appStore *store.Store, guideID, bannerURL string) {
	if moderationSvc == nil || bannerURL == "" {
		return
	}
	go func() {
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "https://createmod.com"
		}
		fullURL := baseURL + bannerURL

		result, err := moderationSvc.CheckImage(fullURL)
		if err != nil {
			slog.Warn("guide image moderation unavailable",
				"guide_id", guideID, "error", err)
			return
		}
		if !result.Approved {
			slog.Warn("guide banner flagged by moderation, removing",
				"guide_id", guideID, "reason", result.Reason, "url", bannerURL)
			ctx := context.Background()
			guide, getErr := appStore.Guides.GetByID(ctx, guideID)
			if getErr != nil || guide == nil {
				slog.Error("guide image moderation: failed to load guide",
					"guide_id", guideID, "error", getErr)
				return
			}
			guide.BannerURL = ""
			if updateErr := appStore.Guides.Update(ctx, guide); updateErr != nil {
				slog.Error("guide image moderation: failed to clear banner",
					"guide_id", guideID, "error", updateErr)
			}
		}
	}()
}
