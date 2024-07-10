package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const loginTemplate = "login.html"

type LoginData struct {
	DefaultData
}

func LoginHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := LoginData{}
		d.Title = "Login"
		err := c.Render(http.StatusOK, loginTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
