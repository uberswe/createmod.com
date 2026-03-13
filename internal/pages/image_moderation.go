package pages

import (
	"context"
	"createmod/internal/moderation"
	"createmod/internal/storage"
	"createmod/internal/store"
	"log/slog"
	"net/url"
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

// moderateSchematicImages runs OpenAI image moderation on a schematic's
// featured image and gallery images asynchronously. Flagged images are removed
// from the schematic record and logged. Only the filenames in imagesToCheck
// are moderated (pass only newly uploaded filenames to avoid re-checking
// existing images on every update).
func moderateSchematicImages(moderationSvc *moderation.Service, appStore *store.Store, schematicID string, imagesToCheck []string) {
	if moderationSvc == nil || len(imagesToCheck) == 0 {
		return
	}
	go func() {
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "https://createmod.com"
		}
		s3Prefix := storage.CollectionPrefix("schematics")

		var flaggedImages []string
		for _, filename := range imagesToCheck {
			fullURL := baseURL + "/api/files/" + s3Prefix + "/" + schematicID + "/" + url.PathEscape(filename)
			result, err := moderationSvc.CheckImage(fullURL)
			if err != nil {
				slog.Warn("schematic image moderation unavailable",
					"schematic_id", schematicID, "filename", filename, "error", err)
				continue
			}
			if !result.Approved {
				slog.Warn("schematic image flagged by moderation, will remove",
					"schematic_id", schematicID, "filename", filename, "reason", result.Reason)
				flaggedImages = append(flaggedImages, filename)
			}
		}

		if len(flaggedImages) == 0 {
			return
		}

		flaggedSet := make(map[string]struct{}, len(flaggedImages))
		for _, f := range flaggedImages {
			flaggedSet[f] = struct{}{}
		}

		ctx := context.Background()
		schem, getErr := appStore.Schematics.GetByID(ctx, schematicID)
		if getErr != nil || schem == nil {
			slog.Error("schematic image moderation: failed to load schematic",
				"schematic_id", schematicID, "error", getErr)
			return
		}

		changed := false
		if _, flagged := flaggedSet[schem.FeaturedImage]; flagged {
			slog.Warn("schematic image moderation: removing featured image",
				"schematic_id", schematicID, "filename", schem.FeaturedImage)
			schem.FeaturedImage = ""
			changed = true
		}

		var cleanGallery []string
		for _, g := range schem.Gallery {
			if _, flagged := flaggedSet[g]; flagged {
				slog.Warn("schematic image moderation: removing gallery image",
					"schematic_id", schematicID, "filename", g)
				changed = true
				continue
			}
			cleanGallery = append(cleanGallery, g)
		}
		if cleanGallery == nil {
			cleanGallery = []string{}
		}
		schem.Gallery = cleanGallery

		if changed {
			if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("schematic image moderation: failed to update schematic",
					"schematic_id", schematicID, "error", updateErr)
			}
		}
	}()
}
