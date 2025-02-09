package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

type DefaultData struct {
	IsAuthenticated bool
	Username        string
	UsernameSlug    string
	Title           string
	SubCategory     string
	Categories      []models.SchematicCategory
	Avatar          string
	HasAvatar       bool
}

func (d *DefaultData) Populate(c echo.Context) {
	user := c.Get(apis.ContextAuthRecordKey)
	if user != nil {
		d.IsAuthenticated = true
		if record, ok := user.(*pbmodels.Record); ok {
			caser := cases.Title(language.English)
			d.Username = caser.String(record.GetString("username"))
			d.UsernameSlug = strings.ToLower(record.GetString("username"))
			d.Avatar = record.GetString("avatar")
			d.HasAvatar = d.Avatar != ""
		}
	}
}

func allCategories(app *pocketbase.PocketBase) []models.SchematicCategory {
	categoriesCollection, err := app.Dao().FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByFilter(categoriesCollection.Id, "1=1", "+name", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}
