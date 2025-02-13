package pages

import (
	"createmod/internal/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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

func (d *DefaultData) Populate(e *core.RequestEvent) {
	user := e.Auth
	if user != nil {
		d.IsAuthenticated = true
		caser := cases.Title(language.English)
		d.Username = caser.String(user.GetString("username"))
		d.UsernameSlug = strings.ToLower(user.GetString("username"))
		d.Avatar = user.GetString("avatar")
		d.HasAvatar = d.Avatar != ""

	}
}

func allCategories(app *pocketbase.PocketBase) []models.SchematicCategory {
	categoriesCollection, err := app.FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(categoriesCollection.Id, "1=1", "+name", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}
