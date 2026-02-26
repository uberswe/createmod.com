package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strings"
)

var userGamificationTemplates = append([]string{
	"./template/user-gamification.html",
}, commonTemplates...)

type UserGamificationData struct {
	DefaultData
	Points                int
	Accessories           string
	AccessoryOptions      []string
	AccessorySelected     map[string]bool
	AccessoryRequirements map[string]int
	AllowedAccessories    map[string]bool
}

func UserGamificationHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserGamificationData{}
		d.Populate(e)
		d.Title = "Gamification"
		d.Description = "Your gamification settings."
		d.Slug = "/settings/gamification"
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

		html, err := registry.LoadFiles(userGamificationTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
