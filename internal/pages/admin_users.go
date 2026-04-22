package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var adminUsersTemplates = append([]string{
	"./template/admin_users.html",
}, commonTemplates...)

type AdminUserItem struct {
	ID              string
	Username        string
	Email           string
	Avatar          string
	Points          int
	Verified        bool
	IsAdmin         bool
	Deleted         *time.Time
	SchematicsCount int64
	Created         time.Time
}

type AdminUsersData struct {
	DefaultData
	Users      []AdminUserItem
	Filter     string
	Search     string
	Page       int
	TotalPages int
	Total      int64
	PrevPage   int
	NextPage   int
	QueryStr   string
}

const adminUsersPerPage = 30

// AdminUsersHandler renders the admin user listing page at GET /admin/users.
func AdminUsersHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		filter := e.Request.URL.Query().Get("filter")
		switch filter {
		case "all", "active", "deleted", "admin":
		default:
			filter = "active"
		}

		search := strings.TrimSpace(e.Request.URL.Query().Get("q"))

		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * adminUsersPerPage

		users, err := appStore.Users.ListForAdmin(ctx, filter, search, adminUsersPerPage, offset)
		if err != nil {
			slog.Error("admin users: list failed", "error", err)
			return e.String(http.StatusInternalServerError, "failed to list users")
		}

		total, err := appStore.Users.CountForAdmin(ctx, filter, search)
		if err != nil {
			slog.Error("admin users: count failed", "error", err)
			return e.String(http.StatusInternalServerError, "failed to count users")
		}

		totalPages := int(total) / adminUsersPerPage
		if int(total)%adminUsersPerPage != 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		items := make([]AdminUserItem, 0, len(users))
		for _, u := range users {
			count, _ := appStore.Schematics.CountByAuthorAll(ctx, u.ID)
			items = append(items, AdminUserItem{
				ID:              u.ID,
				Username:        u.Username,
				Email:           u.Email,
				Avatar:          u.Avatar,
				Points:          u.Points,
				Verified:        u.Verified,
				IsAdmin:         u.IsAdmin,
				Deleted:         u.Deleted,
				SchematicsCount: count,
				Created:         u.Created,
			})
		}

		q := url.Values{}
		q.Set("filter", filter)
		if search != "" {
			q.Set("q", search)
		}

		d := AdminUsersData{
			Users:      items,
			Filter:     filter,
			Search:     search,
			Page:       page,
			TotalPages: totalPages,
			Total:      total,
			PrevPage:   page - 1,
			NextPage:   page + 1,
			QueryStr:   q.Encode(),
		}
		d.Populate(e)
		d.AdminSection = "users"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Users"))
		d.Title = i18n.T(d.Language, "Admin: Users")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminUsersTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminUserDeleteHandler soft-deletes a user at POST /admin/users/{id}/delete.
func AdminUserDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		// Guard: prevent an admin from deleting themselves.
		if current := session.UserFromContext(e.Request.Context()); current != nil && current.ID == id {
			return e.String(http.StatusBadRequest, "cannot delete your own account from this panel")
		}
		if err := appStore.Users.CascadeSoftDelete(context.Background(), id); err != nil {
			slog.Error("admin users: delete failed", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to delete user")
		}
		return adminUsersRedirect(e)
	}
}

// AdminUserRestoreHandler restores a soft-deleted user at POST /admin/users/{id}/restore.
func AdminUserRestoreHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		if err := appStore.Users.CascadeRestore(context.Background(), id); err != nil {
			slog.Error("admin users: restore failed", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to restore user")
		}
		return adminUsersRedirect(e)
	}
}

func adminUsersRedirect(e *server.RequestEvent) error {
	dest := strings.TrimSpace(e.Request.FormValue("return"))
	if dest == "" || !strings.HasPrefix(dest, "/admin/users") {
		dest = "/admin/users"
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", dest)
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, dest)
}
