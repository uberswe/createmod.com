package pages

import (
	"bufio"
	"bytes"
	"createmod/internal/cache"
	"encoding/base64"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
	"image"
	"io"
	"net/http"
	"strings"
)

const collectionsNewTemplate = "./template/collections_new.html"

var collectionsNewTemplates = append([]string{
	collectionsNewTemplate,
}, commonTemplates...)

type CollectionsNewData struct {
	DefaultData
	Error string
}

// CollectionsNewHandler renders the new collection form.
func CollectionsNewHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := CollectionsNewData{}
		d.Populate(e)
		d.Title = "Create collection"
		d.Description = "Create a new collection of schematics"
		d.Slug = "/collections/new"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(collectionsNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// CollectionsCreateHandler handles POST /collections to create a collection record in PB.
func CollectionsCreateHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			// Require login to create a collection
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		// accept up to 4MB multipart form (banner is limited to 2MB below)
		_ = e.Request.ParseMultipartForm(4 << 20)

		title := e.Request.FormValue("title")
		if title == "" {
			title = e.Request.FormValue("name")
		}
		description := e.Request.FormValue("description")
		bannerURL := strings.TrimSpace(e.Request.FormValue("banner_url"))

		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		rec := core.NewRecord(coll)
		if title != "" {
			rec.Set("title", title)
			rec.Set("name", title)
		}
		if description != "" {
			rec.Set("description", description)
		}

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
		} else if bannerURL != "" {
			rec.Set("banner_url", bannerURL)
		}

		rec.Set("author", e.Auth.Id)
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}
		// After create, go back to listing (detail page may not exist yet)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/collections")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/collections")
	}
}
