package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
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
	d.Title = ""
	d.Categories = allCategories(app)
	err := c.Render(http.StatusOK, profileTemplate, d)
	if err != nil {
		return err
	}
	return nil
}

func editProfile(c echo.Context, app *pocketbase.PocketBase) error {
	d := ProfileData{}
	d.Title = ""
	d.Categories = allCategories(app)
	err := c.Render(http.StatusOK, profileTemplate, d)
	if err != nil {
		return err
	}
	return nil
}
