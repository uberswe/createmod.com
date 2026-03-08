package pages

import (
	"bufio"
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/storage"
	"createmod/internal/store"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"createmod/internal/server"

	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
)

var guidesNewTemplates = append([]string{
	"./template/guides_new.html",
}, commonTemplates...)

// GuidesNewData holds data for the new guide form.
type GuidesNewData struct {
	DefaultData
	Error string
}

// GuidesNewHandler renders a simple Markdown editor form for creating guides (auth required).
func GuidesNewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := GuidesNewData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "New Guide")
		d.Description = i18n.T(d.Language, "page.guides_create.description")
		d.Slug = "/guides/new"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(guidesNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesCreateHandler handles POST /guides to insert a new guide record.
func GuidesCreateHandler(cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		// accept up to 4MB multipart form (banner is limited to 2MB below)
		_ = e.Request.ParseMultipartForm(4 << 20)

		title := strings.TrimSpace(e.Request.FormValue("title"))
		content := strings.TrimSpace(e.Request.FormValue("content"))
		video := strings.TrimSpace(e.Request.FormValue("video_url"))
		link := strings.TrimSpace(e.Request.FormValue("external_url"))
		if title == "" && content == "" {
			dest := "/guides/new?error=missing_title_or_content"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		authorID := authenticatedUserID(e)
		excerpt := ""
		if content != "" {
			excerpt = stripHTMLTags(content)
			if len(excerpt) > 180 {
				excerpt = excerpt[:180] + "..."
			}
		}

		ctx := context.Background()

		// Process banner image upload
		bannerURL := ""
		if file, header, err := e.Request.FormFile("banner"); err == nil && header != nil {
			defer func() { _ = file.Close() }()
			if header.Size > 2<<20 {
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
			bannerURL = fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)
		}

		newGuide := &store.Guide{
			Title:     title,
			Content:   content,
			Excerpt:   excerpt,
			VideoURL:  video,
			WikiURL:   link,
			BannerURL: bannerURL,
			AuthorID:  &authorID,
		}

		if err := appStore.Guides.Create(ctx, newGuide); err != nil {
			slog.Warn("guides: failed to create", "error", err)
			return e.String(http.StatusInternalServerError, "failed to create guide")
		}

		// Award first_guide achievement asynchronously
		go awardFirstGuide(appStore, authorID)

		// Redirect to the newly created guide's detail page
		dest := "/guides/" + newGuide.ID
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
