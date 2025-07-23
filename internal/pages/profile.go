package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/drexedam/gravatar"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	tmpl "html/template"
	"net/http"
)

var profileTemplates = []string{
	"./template/dist/profile.html",
	"./template/dist/include/schematic_card.html",
}

type ProfileData struct {
	Username      string
	Name          string
	HasSchematics bool
	UserAvatar    tmpl.URL
	Schematics    []models.Schematic
	DefaultData
}

func ProfileHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			return editProfile(e, app, registry, cacheService)
		}
		return showProfile(e, app, cacheService, registry, username)
	}
}

func showProfile(e *core.RequestEvent, app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, username string) error {
	d := ProfileData{}
	d.Populate(e)
	caser := cases.Title(language.English)
	d.Title = "Schematics by " + caser.String(username)
	d.Categories = allCategories(app, cacheService)
	d.Username = caser.String(username)
	d.Description = "Find Create Mod schematics by " + caser.String(username) + " on CreateMod.com"
	d.Slug = "/author/" + username

	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	results, err := app.FindRecordsByFilter(
		usersCollection.Id,
		"username:lower = {:username} && deleted = null && moderated = true",
		"-created",
		1,
		0,
		dbx.Params{"username": e.Request.PathValue("username")})

	if err != nil {
		return err
	}

	if len(results) == 1 {
		d.Schematics = findAuthorSchematics(app, cacheService, "", results[0].Id, 1000, "-created")
		url := gravatar.New(results[0].GetString("email")).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		d.UserAvatar = tmpl.URL(url)
		d.Thumbnail = url
	}
	if len(d.Schematics) > 0 {
		d.HasSchematics = true
	}

	html, err := registry.LoadFiles(profileTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

func editProfile(e *core.RequestEvent, app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) error {
	// TODO make this possible as part of #51
	d := ProfileData{}
	d.Populate(e)
	d.Title = "Edit profile coming soon"
	d.Categories = allCategories(app, cacheService)
	html, err := registry.LoadFiles(profileTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
