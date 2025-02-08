package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
)

const profileTemplate = "profile.html"

type ProfileData struct {
	Username   string
	Name       string
	Schematics []models.Schematic
	DefaultData
}

func ProfileHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		username := c.PathParam("username")
		if username == "" {
			return editProfile(c, app)
		}
		return showProfile(c, app, username)
	}
}

func showProfile(c echo.Context, app *pocketbase.PocketBase, username string) error {
	d := ProfileData{}
	d.Populate(c)
	caser := cases.Title(language.English)
	d.Title = "Profile of " + caser.String(username)
	d.Categories = allCategories(app)
	err := c.Render(http.StatusOK, profileTemplate, d)
	if err != nil {
		return err
	}
	return nil
}

func editProfile(c echo.Context, app *pocketbase.PocketBase) error {
	// TODO make this possible as part of #51
	d := ProfileData{}
	d.Populate(c)
	d.Title = "Editing Profile"
	d.Categories = allCategories(app)
	err := c.Render(http.StatusOK, profileTemplate, d)
	if err != nil {
		return err
	}
	return nil
}
