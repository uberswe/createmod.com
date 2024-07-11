package pages

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"net/http"
)

const passwordResetTemplate = "password-reset.html"

type passwordResetData struct {
	DefaultData
}

func PasswordResetHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := passwordResetData{}
		d.Title = "Reset Password"
		err := c.Render(http.StatusOK, passwordResetTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}
