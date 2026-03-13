package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/outurl"
	"createmod/internal/store"
	"log/slog"
	"net/http"

	"createmod/internal/server"
)

const externalInterstitialTemplate = "./template/external_interstitial.html"

var externalInterstitialTemplates = append([]string{
	externalInterstitialTemplate,
}, commonTemplates...)

type ExternalInterstitialData struct {
	DefaultData
	Target string
}

func ExternalLinkInterstitialHandler(registry *server.Registry, cacheService *cache.Service, outSecret string, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}
		// Decrypt the token to recover the target URL and optional source context
		payload, err := outurl.DecodePayload(token, outSecret)
		if err != nil {
			return e.String(http.StatusForbidden, "invalid or expired link")
		}

		d := ExternalInterstitialData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "External Link"))
		d.Title = i18n.T(d.Language, "You are leaving createmod.com")
		d.Description = i18n.T(d.Language, "page.external.description")
		d.Slug = "/out"
		d.NoIndex = true
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Target = payload.URL

		// Record the click asynchronously so the user is never delayed
		go recordOutgoingClick(appStore, payload)

		html, err := registry.LoadFiles(externalInterstitialTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// recordOutgoingClick increments the click counter for the given (url, source_type, source_id)
// tuple and, for guide sources, increments the guide's view count.
func recordOutgoingClick(appStore *store.Store, p outurl.Payload) {
	if p.SourceType == "" {
		return // old token without source context — nothing to record
	}

	ctx := context.Background()

	// Increment guide views when the source is a guide
	if p.SourceType == "guide" && p.SourceID != "" {
		if err := appStore.Guides.IncrementViews(ctx, p.SourceID); err != nil {
			slog.Warn("guides: failed to increment views", "error", err)
		}
	}

	// Upsert outgoing_clicks record
	if err := appStore.OutgoingClicks.RecordClick(ctx, p.URL, p.SourceType, p.SourceID, nil); err != nil {
		slog.Warn("outgoing_clicks: failed to record click", "error", err)
	}
}
