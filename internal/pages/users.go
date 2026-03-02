package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strconv"
)

var usersTemplates = append([]string{
	"./template/users.html",
}, commonTemplates...)

type UsersData struct {
	DefaultData
	Users    []models.User
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
}

// UsersHandler renders a paginated list of users.
func UsersHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// pagination
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 48
		limit := pageSize + 1
		offset := (page - 1) * pageSize

		coll, err := app.FindCollectionByNameOrId("users")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "users collection not available")
		}
		recs, err := app.FindRecordsByFilter(coll.Id, "deleted = ''", "-created", limit, offset)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to query users")
		}
		hasNext := len(recs) > pageSize
		if hasNext {
			recs = recs[:pageSize]
		}
		// map users (minimal fields)
		users := make([]models.User, 0, len(recs))
		for _, r := range recs {
			users = append(users, models.User{
				ID:       r.Id,
				Username: r.GetString("username"),
				// avatar is computed via gravatar elsewhere; we can use stored avatar field if present
			})
		}

		d := UsersData{
			Users:    users,
			Page:     page,
			PageSize: pageSize,
			HasPrev:  page > 1,
			HasNext:  hasNext,
		}
		if d.HasPrev {
			d.PrevURL = "/users?p=" + strconv.Itoa(page-1)
		}
		if d.HasNext {
			d.NextURL = "/users?p=" + strconv.Itoa(page+1)
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Users")
		d.Description = i18n.T(d.Language, "Browse users on CreateMod.com")
		d.Slug = "/users"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(usersTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
