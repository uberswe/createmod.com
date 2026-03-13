package pages

import (
	"bufio"
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
	"fmt"
	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
	"image"
	"io"
	"net/http"
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
func CollectionsNewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := CollectionsNewData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Collections"), "/collections", i18n.T(d.Language, "Create collection"))
		d.Title = i18n.T(d.Language, "Create collection")
		d.Description = i18n.T(d.Language, "Create a new collection of schematics")
		d.Slug = "/collections/new"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(collectionsNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// CollectionsCreateHandler handles POST /collections to create a new collection.
func CollectionsCreateHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		// accept up to 4MB multipart form (banner is limited to 2MB below)
		_ = e.Request.ParseMultipartForm(4 << 20)

		title := e.Request.FormValue("title")
		if title == "" {
			title = e.Request.FormValue("name")
		}
		description := e.Request.FormValue("description")

		authorID := authenticatedUserID(e)
		ctx := context.Background()

		newColl := &store.Collection{
			Title:       title,
			Name:        title,
			Description: description,
			AuthorID:    &authorID,
		}

		// If a banner file is provided, process it and upload to S3
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
			newColl.BannerURL = fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)
		}

		if err := appStore.Collections.Create(ctx, newColl); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}
		// Award first_collection achievement asynchronously
		go awardFirstCollection(appStore, authorID)

		// After create, go back to listing (detail page may not exist yet)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/collections"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/collections"))
	}
}
