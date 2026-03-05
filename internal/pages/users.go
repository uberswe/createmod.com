package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"createmod/internal/server"
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
func UsersHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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

		storeUsers, err := appStore.Users.ListUsers(context.Background(), limit, offset)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to query users")
		}
		hasNext := len(storeUsers) > pageSize
		if hasNext {
			storeUsers = storeUsers[:pageSize]
		}
		// map users (minimal fields)
		users := make([]models.User, 0, len(storeUsers))
		for _, u := range storeUsers {
			users = append(users, models.User{
				ID:       u.ID,
				Username: u.Username,
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
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(usersTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
