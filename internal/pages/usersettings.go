package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strings"
	"time"
)

var userSettingsTemplates = append([]string{
	"./template/user-settings.html",
}, commonTemplates...)

type APIKeyItem struct {
	ID      string
	Label   string
	Last8   string
	Created time.Time
}

type UserSettingsData struct {
	DefaultData
	APIKeys               []APIKeyItem
	NewAPIKey             string
	Points                int
	Accessories           string
	AccessoryOptions      []string
	AccessorySelected     map[string]bool
	AccessoryRequirements map[string]int
	AllowedAccessories    map[string]bool
}

func UserSettingsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserSettingsData{}
		d.Populate(e)
		d.Title = "Settings"
		d.Description = "The user settings page."
		d.Slug = "/settings"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		// Load user points and accessories (best-effort)
		if urec, err := app.FindRecordById("_pb_users_auth_", e.Auth.Id); err == nil && urec != nil {
			d.Points = urec.GetInt("points")
			d.Accessories = urec.GetString("accessories")
		}
		// Provide simple predefined accessories and requirements
		d.AccessoryOptions = []string{"goggles", "wrench", "mustache"}
		d.AccessoryRequirements = map[string]int{"goggles": 50, "wrench": 100, "mustache": 200}
		// Which accessories the user currently has selected
		as := map[string]bool{}
		for _, k := range strings.Split(d.Accessories, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				as[k] = true
			}
		}
		d.AccessorySelected = as
		// Determine which accessories are allowed (unlocked) based on points
		allowed := map[string]bool{}
		for _, opt := range d.AccessoryOptions {
			req := d.AccessoryRequirements[opt]
			allowed[opt] = d.Points >= req
		}
		d.AllowedAccessories = allowed

		// Load user's API keys (best-effort)
		if coll, err := app.FindCollectionByNameOrId("api_keys"); err == nil && coll != nil {
			recs, _ := app.FindRecordsByFilter(coll.Id, "user = {:u}", "-created", 200, 0, dbx.Params{"u": e.Auth.Id})
			items := make([]APIKeyItem, 0, len(recs))
			for _, r := range recs {
				items = append(items, APIKeyItem{
					ID:      r.Id,
					Label:   r.GetString("label"),
					Last8:   r.GetString("last8"),
					Created: r.GetDateTime("created").Time(),
				})
			}
			d.APIKeys = items
		}

		// One-time new API key display
		if key, ok := cacheService.GetString("apikey:new:" + e.Auth.Id); ok && key != "" {
			d.NewAPIKey = key
			cacheService.Delete("apikey:new:" + e.Auth.Id)
		}

		html, err := registry.LoadFiles(userSettingsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
