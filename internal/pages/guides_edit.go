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

var guidesEditTemplates = append([]string{
	"./template/guides_edit.html",
}, commonTemplates...)

// GuideEditData holds data for the guide edit form.
type GuideEditData struct {
	DefaultData
	GuideID    string
	GuideTitle string
	Content    string
	VideoURL   string
	WikiURL    string
	Excerpt    string
	BannerURL  string
	Error      string
}

// GuidesEditHandler renders the edit form for an existing guide (owner-only).
func GuidesEditHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides/"+id))
		}

		d := GuideEditData{
			GuideID:    guide.ID,
			GuideTitle: guide.Title,
			Content:    guide.Content,
			VideoURL:   guide.VideoURL,
			WikiURL:    guide.WikiURL,
			Excerpt:    guide.Excerpt,
			BannerURL:  guide.BannerURL,
		}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Guides"), "/guides", d.GuideTitle, "/guides/"+d.GuideID, i18n.T(d.Language, "Edit"))
		d.Title = i18n.T(d.Language, "Edit Guide")
		d.Description = i18n.T(d.Language, "page.guides_edit.description")
		d.Slug = "/guides/" + id + "/edit"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(guidesEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesUpdateHandler handles POST /guides/{id} to update an existing guide (owner-only).
func GuidesUpdateHandler(cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides/"+id))
		}

		// accept up to 4MB multipart form (banner is limited to 2MB below)
		_ = e.Request.ParseMultipartForm(4 << 20)

		title := strings.TrimSpace(e.Request.FormValue("title"))
		content := strings.TrimSpace(e.Request.FormValue("content"))
		video := strings.TrimSpace(e.Request.FormValue("video_url"))
		link := strings.TrimSpace(e.Request.FormValue("external_url"))
		excerpt := strings.TrimSpace(e.Request.FormValue("excerpt"))

		if title != "" {
			guide.Title = title
		}
		guide.Content = content
		if video != "" {
			guide.VideoURL = video
		}
		if link != "" {
			guide.WikiURL = link
		}
		if excerpt != "" {
			guide.Excerpt = excerpt
		} else if content != "" {
			ex := stripHTMLTags(content)
			if len(ex) > 180 {
				ex = ex[:180] + "..."
			}
			guide.Excerpt = ex
		}

		// Process banner image upload
		if file, header, fErr := e.Request.FormFile("banner"); fErr == nil && header != nil {
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
			guide.BannerURL = fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)
		}

		if err := appStore.Guides.Update(ctx, guide); err != nil {
			slog.Warn("guides: failed to update", "error", err)
		}

		dest := "/guides/" + id
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// GuidesDeleteHandler handles POST /guides/{id}/delete to delete a guide (owner-only).
func GuidesDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		if err := appStore.Guides.Delete(ctx, guide.ID); err != nil {
			slog.Warn("guides: failed to delete", "error", err)
		}

		dest := "/guides"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// stripHTMLTags removes HTML tags from a string for generating plain-text excerpts.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
