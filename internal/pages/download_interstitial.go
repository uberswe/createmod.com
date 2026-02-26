package pages

import (
	"createmod/internal/cache"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

const downloadInterstitialTemplate = "./template/download_interstitial.html"

var downloadInterstitialTemplates = append([]string{
	downloadInterstitialTemplate,
}, commonTemplates...)

type DownloadInterstitialData struct {
	DefaultData
	Name        string
	Token       string
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

func DownloadInterstitialHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return e.String(http.StatusBadRequest, "missing name")
		}

		// Try to load schematic to determine if it's paid and already published
		paid := false
		external := ""
		if coll, err := app.FindCollectionByNameOrId("schematics"); err == nil && coll != nil {
			recs, err := app.FindRecordsByFilter(coll.Id, "name = {:name} && deleted = '' && moderated = true && (scheduled_at = null || scheduled_at <= {:now})", "-created", 1, 0, dbx.Params{"name": name, "now": time.Now()})
			if err == nil && len(recs) > 0 {
				rec := recs[0]
				paid = rec.GetBool("paid")
				external = rec.GetString("external_url")
			}
		}

		d := DownloadInterstitialData{}
		d.Populate(e)
		d.Slug = "/get/" + name
		d.Categories = allCategories(app, cacheService)
		d.Name = name

		if paid && external != "" {
			// Paid: do not mint token; route to external interstitial
			d.Paid = true
			d.ExternalURL = external
			d.Title = "Preparing External Link"
			d.Description = "You will be redirected to the seller's site shortly."
		} else {
			// Free: generate one-time token and store with TTL
			token := randomHex(24)
			cacheService.SetWithTTL("dl:"+token, name, 2*time.Minute)
			d.Title = "Preparing Download"
			d.Description = "Your download will begin shortly."
			d.Token = token
		}

		html, err := registry.LoadFiles(downloadInterstitialTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
