package pages

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/storage"
	"createmod/internal/store"
	"io"
	"log/slog"
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

// approvedTempImages filters a temp-image list to those that passed content
// moderation, for display. Pending images aren't servable yet and rejected ones
// were removed, so neither is shown. Takes the (slice, err) result of a store
// call directly for convenience. (#1646)
func approvedTempImages(imgs []store.TempUploadImage, _ error) []store.TempUploadImage {
	out := make([]store.TempUploadImage, 0, len(imgs))
	for _, img := range imgs {
		if img.ModerationStatus == store.TempImageApproved {
			out = append(out, img)
		}
	}
	return out
}

// ModerateTempUploadImageAsync runs the free, violence-tolerant content check on
// a just-uploaded temp image (nudity/hate/harassment/self-harm blocked; violence
// allowed) and records the outcome. Approved images become servable/displayable;
// flagged images are deleted from storage and marked rejected. Runs in its own
// goroutine so the upload response returns immediately. (#1646)
func ModerateTempUploadImageAsync(moderationSvc *moderation.Service, storageSvc *storage.Service, appStore *store.Store, imageID, s3Key string, data []byte, mimeType string) {
	if appStore == nil || imageID == "" {
		return
	}
	// No moderation configured (e.g. dev): approve so images still work.
	if moderationSvc == nil {
		_ = appStore.TempUploadImages.UpdateModerationStatus(context.Background(), imageID, store.TempImageApproved)
		return
	}
	go func() {
		ctx := context.Background()
		res, err := moderationSvc.CheckUserImageContent(data, mimeType)
		if err != nil {
			// Moderation unavailable — fail OPEN (approve) so an OpenAI outage
			// doesn't break uploads; the post-publish image checks remain a
			// backstop for anything published.
			slog.Warn("temp image moderation unavailable, approving", "image_id", imageID, "error", err)
			_ = appStore.TempUploadImages.UpdateModerationStatus(ctx, imageID, store.TempImageApproved)
			return
		}
		if res.Approved {
			_ = appStore.TempUploadImages.UpdateModerationStatus(ctx, imageID, store.TempImageApproved)
			return
		}
		// Rejected: delete the object so it is never served, mark the record.
		slog.Warn("temp image rejected by content moderation, removing", "image_id", imageID, "reason", res.Reason)
		if storageSvc != nil && s3Key != "" {
			if delErr := storageSvc.DeleteRaw(ctx, s3Key); delErr != nil {
				slog.Error("temp image moderation: failed to delete rejected image", "image_id", imageID, "s3_key", s3Key, "error", delErr)
			}
		}
		_ = appStore.TempUploadImages.UpdateModerationStatus(ctx, imageID, store.TempImageRejected)
	}()
}

// HoldSchematicImages marks filenames as held on a schematic — hidden from
// visitors, shown to the owner as "in review" placeholders — WITHOUT changing
// the schematic's moderation state. If the featured image is held it falls back
// to the first still-visible gallery image. The hold is atomic, so it is safe
// against the concurrent gallery/featured moderation goroutines. Writes a
// moderation-log entry and, on the schematic's FIRST hold, emails the author
// (best-effort). (#1646)
func HoldSchematicImages(ctx context.Context, mailService *mailer.Service, appStore *store.Store, schematicID string, filenames []string, reason string) {
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
	// Notify the author only on the FIRST hold (all held images are exactly the
	// ones just added), so the concurrent featured/gallery paths don't both mail.
	uniq := make(map[string]struct{}, len(filenames))
	for _, f := range filenames {
		uniq[f] = struct{}{}
	}
	if mailService != nil && len(store.HeldGallery(updated)) == len(uniq) {
		SendSchematicImageReviewEmail(ctx, mailService, appStore, schematicID)
	}
}

// moderateSchematicImages runs OpenAI image moderation on a schematic's gallery
// images asynchronously. Flagged images are held (hidden) and logged. Only the
// filenames in imagesToCheck are moderated (pass only newly uploaded filenames
// to avoid re-checking existing images on every update).
//
// Images are moderated by their BYTES (downloaded from S3), not a public URL:
// the URL-based path silently passes whenever the OpenAI servers can't fetch the
// image (dev behind an auth gate, private hosts, a wrong BASE_URL).
func moderateSchematicImages(moderationSvc *moderation.Service, storageSvc *storage.Service, mailService *mailer.Service, appStore *store.Store, schematicID string, imagesToCheck []string) {
	if moderationSvc == nil || storageSvc == nil || len(imagesToCheck) == 0 {
		return
	}
	go func() {
		ctx := context.Background()
		var flaggedImages []string
		for _, filename := range imagesToCheck {
			data, mimeType := downloadSchematicImage(ctx, storageSvc, schematicID, filename)
			if data == nil {
				slog.Warn("schematic image unavailable for byte moderation",
					"schematic_id", schematicID, "filename", filename)
				continue
			}
			// Content policy (nudity/hate/harassment/self-harm; violence allowed).
			result, err := moderationSvc.CheckUserImageContent(data, mimeType)
			if err != nil {
				slog.Warn("schematic image moderation unavailable",
					"schematic_id", schematicID, "filename", filename, "error", err)
				continue
			}
			if !result.Approved {
				slog.Warn("schematic image flagged by moderation, will hold",
					"schematic_id", schematicID, "filename", filename, "reason", result.Reason)
				flaggedImages = append(flaggedImages, filename)
				continue
			}
			// Also verify images depict actual Minecraft builds.
			qualResult, qualErr := moderationSvc.CheckImageQualityBytes(data, mimeType)
			if qualErr != nil {
				slog.Warn("schematic image quality check unavailable",
					"schematic_id", schematicID, "filename", filename, "error", qualErr)
				continue
			}
			if !qualResult.Approved {
				slog.Warn("schematic image not a Minecraft build, will hold",
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
		HoldSchematicImages(ctx, mailService, appStore, schematicID, flaggedImages,
			"Held by automated image moderation: "+strings.Join(flaggedImages, ", "))
	}()
}

// maxModerationImageBytes caps how much of an image is read before moderating
// its bytes. Uploaded images are capped well below this.
const maxModerationImageBytes = 12 << 20

// downloadSchematicImage fetches a schematic image's bytes and MIME type for
// byte-based moderation. Returns (nil, "") on any failure.
func downloadSchematicImage(ctx context.Context, storageSvc *storage.Service, schematicID, filename string) ([]byte, string) {
	if storageSvc == nil || schematicID == "" || filename == "" {
		return nil, ""
	}
	reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), schematicID, filename)
	if err != nil {
		return nil, ""
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxModerationImageBytes))
	if err != nil || len(data) == 0 {
		return nil, ""
	}
	return data, imageMimeFromName(filename)
}

// imageMimeFromName maps an image filename to a MIME type for the base64 data
// URI sent to OpenAI. Uploaded images are converted to WebP; other extensions
// are handled for pre-conversion checks.
func imageMimeFromName(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	default:
		return "image/webp"
	}
}
