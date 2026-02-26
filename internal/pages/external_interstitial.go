package pages

import (
	"createmod/internal/cache"
	"createmod/internal/outurl"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const externalInterstitialTemplate = "./template/external_interstitial.html"

var externalInterstitialTemplates = append([]string{
	externalInterstitialTemplate,
}, commonTemplates...)

type ExternalInterstitialData struct {
	DefaultData
	Target string
}

func ExternalLinkInterstitialHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, outSecret string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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
		d.Title = "You are leaving createmod.com"
		d.Description = "External link warning"
		d.Slug = "/out"
		d.Categories = allCategories(app, cacheService)
		d.Target = payload.URL

		// Record the click asynchronously so the user is never delayed
		go recordOutgoingClick(app, payload)

		html, err := registry.LoadFiles(externalInterstitialTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// recordOutgoingClick increments the click counter for the given (url, source_type, source_id)
// tuple and, for guide sources, increments the guide's view count.
func recordOutgoingClick(app *pocketbase.PocketBase, p outurl.Payload) {
	if p.SourceType == "" {
		return // old token without source context — nothing to record
	}

	// Increment guide views when the source is a guide
	if p.SourceType == "guide" && p.SourceID != "" {
		if coll, err := app.FindCollectionByNameOrId("guides"); err == nil && coll != nil {
			if rec, err := app.FindRecordById(coll.Id, p.SourceID); err == nil && rec != nil {
				current := rec.GetInt("views")
				rec.Set("views", current+1)
				if err := app.Save(rec); err != nil {
					app.Logger().Warn("guides: failed to increment views", "error", err)
				}
			}
		}
	}

	// Upsert outgoing_clicks record
	coll, err := app.FindCollectionByNameOrId("outgoing_clicks")
	if err != nil {
		app.Logger().Warn("outgoing_clicks: collection not found", "error", err)
		return
	}

	// Try to find existing record
	recs, err := app.FindRecordsByFilter(coll.Id,
		"url = {:url} && source_type = {:sourceType} && source_id = {:sourceID}",
		"", 1, 0,
		map[string]any{"url": p.URL, "sourceType": p.SourceType, "sourceID": p.SourceID},
	)
	if err == nil && len(recs) > 0 {
		rec := recs[0]
		rec.Set("clicks", rec.GetInt("clicks")+1)
		if err := app.Save(rec); err != nil {
			app.Logger().Warn("outgoing_clicks: failed to increment", "error", err)
		}
		return
	}

	// Create new record
	rec := core.NewRecord(coll)
	rec.Set("url", p.URL)
	rec.Set("source_type", p.SourceType)
	rec.Set("source_id", p.SourceID)
	rec.Set("clicks", 1)
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("outgoing_clicks: failed to create", "error", err)
	}
}
