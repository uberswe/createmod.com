package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
)

const profileTemplate = "./template/dist/profile.html"

type ProfileData struct {
	Username   string
	Name       string
	Schematics []models.Schematic
	DefaultData
}

func ProfileHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			return editProfile(e, app, registry)
		}
		return showProfile(e, app, cacheService, registry, username)
	}
}

func showProfile(e *core.RequestEvent, app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, username string) error {
	d := ProfileData{}
	d.Populate(e)
	caser := cases.Title(language.English)
	d.Title = "Schematics by " + caser.String(username)
	d.Categories = allCategories(app)

	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	results, err := app.FindRecordsByFilter(
		usersCollection.Id,
		"username = {:username}",
		"-created",
		1,
		0,
		dbx.Params{"username": e.Request.PathValue("username")})

	if err != nil {
		return err
	}

	if len(results) == 1 {
		d.Schematics = findAuthorSchematics(app, cacheService, "", results[0].Id, 1000, "-created")
	}

	html, err := registry.LoadFiles(profileTemplate).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

func editProfile(e *core.RequestEvent, app *pocketbase.PocketBase, registry *template.Registry) error {
	// TODO make this possible as part of #51
	d := ProfileData{}
	d.Populate(e)
	d.Title = "Edit profile coming soon"
	d.Categories = allCategories(app)
	html, err := registry.LoadFiles(profileTemplate).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
