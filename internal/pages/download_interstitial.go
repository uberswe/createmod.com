package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"createmod/internal/server"
)

const downloadInterstitialTemplate = "./template/download_interstitial.html"

var downloadInterstitialTemplates = append([]string{
	downloadInterstitialTemplate,
}, commonTemplates...)

type DownloadInterstitialData struct {
	DefaultData
	Name        string
	TokenID     string
	Paid        bool
	ExternalURL string
}

func randomHex(n int) string {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func DownloadInterstitialHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return e.String(http.StatusBadRequest, "missing name")
		}

		// Try to load schematic to determine if it's paid and already published
		paid := false
		external := ""
		if s, err := appStore.Schematics.GetByName(context.Background(), name); err == nil && s != nil && store.IsPublicState(s.ModerationState) && (s.Deleted == nil || s.Deleted.IsZero()) {
			paid = s.Paid
			external = s.ExternalURL
		}

		d := DownloadInterstitialData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Schematics"), "/schematics", i18n.T(d.Language, "Download"))
		d.Slug = "/get/" + name
		d.NoIndex = true
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Name = name

		if paid && external != "" {
			// Paid: do not mint token; route to external interstitial
			d.Paid = true
			d.ExternalURL = external
			d.Title = i18n.T(d.Language, "Preparing External Link")
			d.Description = i18n.T(d.Language, "page.download.external.description")
		} else {
			// Free: generate one-time token and store in PostgreSQL
			token := randomHex(24)
			dt := &store.DownloadToken{
				Token:     token,
				Name:      name,
				ExpiresAt: time.Now().Add(2 * time.Minute),
			}
			if err := appStore.DownloadTokens.Create(context.Background(), dt); err != nil {
				return e.String(http.StatusInternalServerError, "failed to create download token")
			}
			d.Title = i18n.T(d.Language, "Preparing Download")
			d.Description = i18n.T(d.Language, "page.download.file.description")
			d.TokenID = dt.ID
		}

		html, err := registry.LoadFiles(downloadInterstitialTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func DownloadURLHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing id"})
		}

		dt, err := appStore.DownloadTokens.GetByID(e.Request.Context(), id)
		if err != nil || dt == nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		}

		return e.JSON(http.StatusOK, map[string]string{
			"url": "/download/" + dt.Name + "?t=" + dt.Token,
		})
	}
}
