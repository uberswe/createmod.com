package pages

import (
	"bufio"
	"bytes"
	"createmod/internal/cache"
	"createmod/internal/moderation"
	"encoding/base64"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
	"html/template"
	"image"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
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
func CollectionsEditHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		slug := e.Request.PathValue("slug")
		d := CollectionsEditData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)
		d.Slug = "/collections/" + slug

		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		// Find by slug first, fallback to id
		var rec *core.Record
		if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
			rec = r[0]
		}
		if rec == nil {
			if r, err := app.FindRecordById(coll.Id, slug); err == nil {
				rec = r
			}
		}
		if rec == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		// Author-only
		if rec.GetString("author") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}

		d.TitleText = rec.GetString("title")
		if d.TitleText == "" {
			d.TitleText = rec.GetString("name")
		}
		d.Description = rec.GetString("description")
		d.BannerURL = rec.GetString("banner_url")
		d.Published = rec.GetBool("published")
		d.Title = "Edit collection"

		// Discover associated schematics to power the reorder UI.
		// Preference:
		//  1) If join table exists with optional numeric position, use it (ascending by position if present).
		//  2) Else use the collection's multi-rel field "schematics" as-is.
		ids := make([]string, 0, 64)
		// Start with multi-rel field as fallback.
		if rel := rec.GetStringSlice("schematics"); len(rel) > 0 {
			// copy to avoid mutating underlying slice
			tmp := make([]string, 0, len(rel))
			seen := make(map[string]struct{}, len(rel))
			for _, s := range rel {
				if s == "" {
					continue
				}
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
				tmp = append(tmp, s)
			}
			ids = tmp
		}
		// Try join associations
		type pair struct {
			sid string
			pos int
			idx int
		}
		best := make([]pair, 0, 128)
		for _, jn := range []string{"collections_schematics", "collection_schematics"} {
			if jcoll, jerr := app.FindCollectionByNameOrId(jn); jerr == nil && jcoll != nil {
				// Load links. Sort by -created to get deterministic latest-first; we'll re-sort by position if present.
				if links, _ := app.FindRecordsByFilter(jcoll.Id, "collection = {:c}", "-created", 5000, 0, dbx.Params{"c": rec.Id}); len(links) > 0 {
					best = best[:0]
					seen := make(map[string]struct{}, len(links))
					for i, l := range links {
						sid := l.GetString("schematic")
						if sid == "" {
							continue
						}
						if _, ok := seen[sid]; ok {
							continue
						}
						seen[sid] = struct{}{}
						p := l.GetInt("position")
						best = append(best, pair{sid: sid, pos: p, idx: i})
					}
					// If any position > 0, sort by pos ascending then idx to stabilize.
					anyPos := false
					for _, it := range best {
						if it.pos > 0 {
							anyPos = true
							break
						}
					}
					if anyPos {
						sort.SliceStable(best, func(i, j int) bool {
							if best[i].pos != best[j].pos {
								return best[i].pos < best[j].pos
							}
							return best[i].idx < best[j].idx
						})
					}
					ids = ids[:0]
					for _, it := range best {
						ids = append(ids, it.sid)
					}
					break // prefer the first join table found
				}
			}
		}
		d.SchematicIDs = ids
		d.DescriptionHTML = template.HTML(d.Description)
		d.ReorderSchematics = loadReorderSchematics(app, ids)

		html, err := registry.LoadFiles(collectionsEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// loadReorderSchematics loads lightweight schematic data for the reorder UI.
func loadReorderSchematics(app *pocketbase.PocketBase, ids []string) []ReorderSchematic {
	if len(ids) == 0 {
		return nil
	}
	smColl, err := app.FindCollectionByNameOrId("schematics")
	if err != nil || smColl == nil {
		return nil
	}
	result := make([]ReorderSchematic, 0, len(ids))
	for _, id := range ids {
		r, err := app.FindRecordById(smColl.Id, id)
		if err != nil || r == nil {
			result = append(result, ReorderSchematic{ID: id, Title: id})
			continue
		}
		title := r.GetString("name")
		if title == "" {
			title = id
		}
		featuredImage := r.GetString("featured_image")
		result = append(result, ReorderSchematic{
			ID:            id,
			Title:         title,
			FeaturedImage: featuredImage,
		})
	}
	return result
}

// CollectionsUpdateHandler handles POST updates to a collection (author-only).
// Supports action=save (default), action=publish (with validation + moderation), action=unpublish.
func CollectionsUpdateHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, moderationService *moderation.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		slug := e.Request.PathValue("slug")
		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		var rec *core.Record
		if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
			rec = r[0]
		}
		if rec == nil {
			if r, err := app.FindRecordById(coll.Id, slug); err == nil {
				rec = r
			}
		}
		if rec == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		if rec.GetString("author") != e.Auth.Id {
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
			rec.Set("title", title)
			rec.Set("name", title)
		}
		rec.Set("description", description)

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
				// too wide, crop width
				newW := int(float64(h) * targetRatio)
				x0 := b.Min.X + (w-newW)/2
				crop = image.Rect(x0, b.Min.Y, x0+newW, b.Min.Y+h)
			} else {
				// too tall, crop height
				newH := int(float64(w) / targetRatio)
				y0 := b.Min.Y + (h-newH)/2
				crop = image.Rect(b.Min.X, y0, b.Min.X+w, y0+newH)
			}
			cropped := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(crop)
			// resize to 1600x400
			dst := image.NewRGBA(image.Rect(0, 0, 1600, 400))
			draw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)
			var out bytes.Buffer
			bw := bufio.NewWriter(&out)
			if err := imgconv.Write(bw, dst, &imgconv.FormatOption{Format: imgconv.WEBP, EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)}}); err != nil {
				return e.String(http.StatusInternalServerError, "failed to encode banner image")
			}
			_ = bw.Flush()
			dataURL := "data:image/webp;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
			rec.Set("banner_url", dataURL)
		}

		// renderEditWithError re-renders the edit form with an error message.
		renderEditWithError := func(errMsg string) error {
			d := CollectionsEditData{}
			d.Populate(e)
			d.Categories = allCategories(app, cacheService)
			d.Slug = "/collections/" + slug
			d.TitleText = title
			d.Description = description
			d.DescriptionHTML = template.HTML(description)
			d.BannerURL = rec.GetString("banner_url")
			d.Published = rec.GetBool("published")
			d.Error = errMsg
			d.Title = "Edit collection"
			// Reload schematic IDs for the reorder UI
			ids := rec.GetStringSlice("schematics")
			d.SchematicIDs = ids
			d.ReorderSchematics = loadReorderSchematics(app, ids)
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
					app.Logger().Error("collection publish moderation error", "error", err, "id", rec.Id)
					return renderEditWithError("Content moderation check failed. Please try again later.")
				}
				if !result.Approved {
					return renderEditWithError(fmt.Sprintf("Content did not pass moderation: %s", result.Reason))
				}
			}
			rec.Set("published", true)
		}

		if action == "unpublish" {
			rec.Set("published", false)
		}

		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}
		dest := "/collections/" + slug
		if action == "publish" || action == "unpublish" {
			dest = "/collections/" + slug + "/edit"
		}
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// CollectionsDeleteHandler handles POST delete (soft-delete) for a collection (author-only).
func CollectionsDeleteHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		slug := e.Request.PathValue("slug")
		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		var rec *core.Record
		if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
			rec = r[0]
		}
		if rec == nil {
			if r, err := app.FindRecordById(coll.Id, slug); err == nil {
				rec = r
			}
		}
		if rec == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		if rec.GetString("author") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}
		// Soft delete: set a string timestamp in "deleted" for compatibility with earlier filters
		rec.Set("deleted", time.Now().UTC().Format(time.RFC3339))
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete collection")
		}
		dest := "/collections"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}
