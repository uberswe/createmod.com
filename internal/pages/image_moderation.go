package pages

import (
	"context"
	"createmod/internal/moderation"
	"createmod/internal/storage"
	"createmod/internal/store"
	"log/slog"
	"net/url"
	"os"
	"strings"
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

// HoldSchematicImages marks filenames as held on a schematic — hidden from
// visitors, shown to the owner as "in review" placeholders — WITHOUT changing
// the schematic's moderation state. If the featured image is held it falls back
// to the first still-visible gallery image. The hold is atomic, so it is safe
// against the concurrent gallery/featured moderation goroutines. Writes a
// moderation-log entry. (#1646)
func HoldSchematicImages(ctx context.Context, appStore *store.Store, schematicID string, filenames []string, reason string) {
	if appStore == nil || len(filenames) == 0 {
		return
	}
	updated, err := appStore.Schematics.HoldImages(ctx, schematicID, filenames)
	if err != nil || updated == nil {
		slog.Error("moderation: failed to hold images", "schematic_id", schematicID, "error", err)
		return
	}
	slog.Info("moderation: held images", "schematic_id", schematicID, "filenames", filenames, "reason", reason)
	if appStore.ModerationLog != nil {
		note := reason
		if note == "" {
			note = "held images: " + strings.Join(filenames, ", ")
		}
		_ = appStore.ModerationLog.Create(ctx, &store.ModerationLogEntry{
			SchematicID: schematicID,
			ActorType:   "system",
			Action:      "image_hold",
			OldState:    updated.ModerationState,
			NewState:    updated.ModerationState,
			Reason:      note,
		})
	}
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
				continue
			}
			// Also verify images depict actual Minecraft builds
			qualResult, qualErr := moderationSvc.CheckImageQuality(fullURL)
			if qualErr != nil {
				slog.Warn("schematic image quality check unavailable",
					"schematic_id", schematicID, "filename", filename, "error", qualErr)
				continue
			}
			if !qualResult.Approved {
				slog.Warn("schematic image not a Minecraft build, will flag",
					"schematic_id", schematicID, "filename", filename, "reason", qualResult.Reason)
				flaggedImages = append(flaggedImages, filename)
			}
		}

		if len(flaggedImages) == 0 {
			return
		}
		// Per-image hold: hide just the flagged images and keep the schematic
		// published (falling back to a visible featured image if needed), instead
		// of holding the whole schematic for manual review. (#1646)
		HoldSchematicImages(context.Background(), appStore, schematicID, flaggedImages,
			"Held by automated image moderation: "+strings.Join(flaggedImages, ", "))
	}()
}
