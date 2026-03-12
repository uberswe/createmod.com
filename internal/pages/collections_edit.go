package pages

import (
	"bufio"
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/storage"
	"createmod/internal/store"
	"fmt"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"image"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"createmod/internal/server"
	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
)

var collectionsEditTemplates = append([]string{
	"./template/collections_edit.html",
}, commonTemplates...)

// ReorderSchematic holds lightweight data for the reorder UI.
type ReorderSchematic struct {
	ID            string
	Title         string
	FeaturedImage string
}

type CollectionsEditData struct {
	DefaultData
	TitleText         string
	Description       string
	DescriptionHTML   template.HTML
	BannerURL         string
	Error             string
	Published         bool
	SchematicIDs      []string
	ReorderSchematics []ReorderSchematic
}

// CollectionsEditHandler renders the edit form for a collection (author-only).
func CollectionsEditHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		slug := e.Request.PathValue("slug")
		d := CollectionsEditData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/collections/" + slug

		ctx := context.Background()

		// Find by slug first, fallback to id
		coll, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || coll == nil {
			coll, err = appStore.Collections.GetByID(ctx, slug)
		}
		if coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		// Author-only
		if coll.AuthorID == nil || *coll.AuthorID != authenticatedUserID(e) {
			return e.String(http.StatusForbidden, "not allowed")
		}

		d.TitleText = coll.Title
		if d.TitleText == "" {
			d.TitleText = coll.Name
		}
		d.Description = coll.Description
		d.BannerURL = coll.BannerURL
		d.Published = coll.Published
		d.Title = i18n.T(d.Language, "Edit collection")

		// Load associated schematics via the join table (store handles position ordering)
		ids, _ := appStore.Collections.GetSchematicIDs(ctx, coll.ID)
		d.SchematicIDs = ids
		sanitizer := htmlsanitizer.NewHTMLSanitizer()
		sanitizedDesc, sanitizeErr := sanitizer.SanitizeString(d.Description)
		if sanitizeErr != nil {
			sanitizedDesc = template.HTMLEscapeString(d.Description)
		}
		d.DescriptionHTML = template.HTML(sanitizedDesc)
		d.ReorderSchematics = loadReorderSchematicsFromStore(appStore, ids)

		html, err := registry.LoadFiles(collectionsEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// loadReorderSchematicsFromStore loads lightweight schematic data for the reorder UI.
func loadReorderSchematicsFromStore(appStore *store.Store, ids []string) []ReorderSchematic {
	if len(ids) == 0 {
		return nil
	}
	ctx := context.Background()
	schematics, err := appStore.Schematics.ListByIDs(ctx, ids)
	if err != nil {
		return nil
	}
	// Build lookup map
	byID := make(map[string]store.Schematic, len(schematics))
	for _, s := range schematics {
		byID[s.ID] = s
	}
	result := make([]ReorderSchematic, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			title := s.Name
			if title == "" {
				title = id
			}
			result = append(result, ReorderSchematic{
				ID:            id,
				Title:         title,
				FeaturedImage: s.FeaturedImage,
			})
		} else {
			result = append(result, ReorderSchematic{ID: id, Title: id})
		}
	}
	return result
}

// CollectionsUpdateHandler handles POST updates to a collection (author-only).
// Supports action=save (default), action=publish (with validation + moderation), action=unpublish.
func CollectionsUpdateHandler(registry *server.Registry, cacheService *cache.Service, moderationService *moderation.Service, appStore *store.Store, storageSvc *storage.Service, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		ctx := context.Background()
		slug := e.Request.PathValue("slug")

		// Find collection by slug first, fallback to id
		coll, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || coll == nil {
			coll, err = appStore.Collections.GetByID(ctx, slug)
		}
		if coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		if coll.AuthorID == nil || *coll.AuthorID != authenticatedUserID(e) {
			return e.String(http.StatusForbidden, "not allowed")
		}

		// accept up to 4MB multipart form (banner is limited to 2MB below)
		_ = e.Request.ParseMultipartForm(4 << 20)

		action := strings.TrimSpace(e.Request.FormValue("action"))
		if action == "" {
			action = "save"
		}

		title := e.Request.FormValue("title")
		if title == "" {
			title = e.Request.FormValue("name")
		}
		description := e.Request.FormValue("description")
		if title != "" {
			coll.Title = title
			coll.Name = title
		}
		coll.Description = description

		// If a banner file is provided, process it and set banner_url to a WebP data URL
		if file, header, err := e.Request.FormFile("banner"); err == nil && header != nil {
			defer func() { _ = file.Close() }()
			if header.Size > 2<<20 { // 2MB limit
				return e.String(http.StatusBadRequest, "banner image too large (max 2MB)")
			}
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, file); err != nil {
				return e.String(http.StatusBadRequest, "failed to read banner image")
			}
			img, err := imgconv.Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				return e.String(http.StatusBadRequest, "unsupported or corrupt image (allowed: png, jpg, webp)")
			}
			// center-crop to 4:1
			b := img.Bounds()
			w, h := b.Dx(), b.Dy()
			targetRatio := 4.0
			var crop image.Rectangle
			if float64(w)/float64(h) > targetRatio {
				newW := int(float64(h) * targetRatio)
				x0 := b.Min.X + (w-newW)/2
				crop = image.Rect(x0, b.Min.Y, x0+newW, b.Min.Y+h)
			} else {
				newH := int(float64(w) / targetRatio)
				y0 := b.Min.Y + (h-newH)/2
				crop = image.Rect(b.Min.X, y0, b.Min.X+w, y0+newH)
			}
			cropped := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(crop)
			dst := image.NewRGBA(image.Rect(0, 0, 1600, 400))
			draw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)
			var out bytes.Buffer
			bw := bufio.NewWriter(&out)
			if err := imgconv.Write(bw, dst, &imgconv.FormatOption{Format: imgconv.WEBP, EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)}}); err != nil {
				return e.String(http.StatusInternalServerError, "failed to encode banner image")
			}
			_ = bw.Flush()
			imageID, err := generateImageID()
			if err != nil {
				return e.String(http.StatusInternalServerError, "failed to generate image ID")
			}
			filename := "banner.webp"
			if err := storageSvc.UploadBytes(ctx, "images", imageID, filename, out.Bytes(), "image/webp"); err != nil {
				return e.String(http.StatusInternalServerError, "failed to upload banner")
			}
			coll.BannerURL = fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)
		}

		// renderEditWithError re-renders the edit form with an error message.
		renderEditWithError := func(errMsg string) error {
			d := CollectionsEditData{}
			d.Populate(e)
			d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
			d.Slug = "/collections/" + slug
			d.TitleText = title
			d.Description = description
			errSanitizer := htmlsanitizer.NewHTMLSanitizer()
			errSanitizedDesc, errSanitizeErr := errSanitizer.SanitizeString(description)
			if errSanitizeErr != nil {
				errSanitizedDesc = template.HTMLEscapeString(description)
			}
			d.DescriptionHTML = template.HTML(errSanitizedDesc)
			d.BannerURL = coll.BannerURL
			d.Published = coll.Published
			d.Error = errMsg
			d.Title = i18n.T(d.Language, "Edit collection")
			ids, _ := appStore.Collections.GetSchematicIDs(ctx, coll.ID)
			d.SchematicIDs = ids
			d.ReorderSchematics = loadReorderSchematicsFromStore(appStore, ids)
			html, err := registry.LoadFiles(collectionsEditTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Handle publish action with validation and moderation
		if action == "publish" {
			if len(strings.TrimSpace(title)) < 10 {
				return renderEditWithError("Title must be at least 10 characters to publish.")
			}
			if len(strings.TrimSpace(description)) < 100 {
				return renderEditWithError("Description must be at least 100 characters to publish.")
			}
			if moderationService != nil {
				content := fmt.Sprintf("Title: %s\nDescription: %s", title, description)
				result, err := moderationService.CheckContent(content)
				if err != nil {
					slog.Warn("collection publish moderation unavailable, allowing publish", "error", err, "id", coll.ID)
				} else if !result.Approved {
					return renderEditWithError(fmt.Sprintf("Content did not pass moderation: %s", result.Reason))
				}
			}
			coll.Published = true
		}

		if action == "unpublish" {
			coll.Published = false
		}

		if err := appStore.Collections.Update(ctx, coll); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}

		// Regenerate collage on any save (content may have changed)
		go generateCollectionCollage(storageSvc, appStore, coll.ID)

		// Send admin email on publish
		if action == "publish" && mailService != nil {
			emailTitle := coll.Title
			emailBanner := coll.BannerURL
			if emailBanner == "" {
				emailBanner = coll.CollageURL
			}
			collectionURL := "https://createmod.com/collections/"
			if coll.Slug != "" {
				collectionURL += coll.Slug
			} else {
				collectionURL += coll.ID
			}
			go func() {
				to := adminRecipients(appStore, mailService)
				if len(to) == 0 {
					return
				}
				from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
				subject := fmt.Sprintf("New Collection Published: %s", emailTitle)
				imageURL := ""
				if emailBanner != "" {
					imageURL = "https://createmod.com" + emailBanner
				}
				body := mailer.SchematicEmailHTML(emailTitle, imageURL, collectionURL, "A new collection has been published.")
				if err := mailService.Send(&mailer.Message{From: from, To: to, Subject: subject, HTML: body}); err != nil {
					slog.Error("failed to send collection publish email", "error", err)
				}
			}()
		}

		dest := "/collections/" + slug
		if action == "publish" || action == "unpublish" {
			dest = "/collections/" + slug + "/edit"
		}
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// CollectionsDeleteHandler handles POST delete (soft-delete) for a collection (author-only).
func CollectionsDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		ctx := context.Background()
		slug := e.Request.PathValue("slug")

		coll, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || coll == nil {
			coll, err = appStore.Collections.GetByID(ctx, slug)
		}
		if coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		if coll.AuthorID == nil || *coll.AuthorID != authenticatedUserID(e) {
			return e.String(http.StatusForbidden, "not allowed")
		}
		if err := appStore.Collections.SoftDelete(ctx, coll.ID); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete collection")
		}
		dest := "/collections"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
