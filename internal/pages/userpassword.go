package pages

import (
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"net/http"
	"strings"

	"createmod/internal/server"
)

var userPasswordTemplates = append([]string{
	"./template/user-password.html",
}, commonTemplates...)

type UserPasswordData struct {
	DefaultData
	Success bool
	Error   string
}

func UserPasswordHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserPasswordData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Change Password"))
		d.Title = i18n.T(d.Language, "Change Password")
		d.Description = i18n.T(d.Language, "page.userpassword.description")
		d.Slug = "/settings/password"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(userPasswordTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func UserPasswordPostHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		oldPassword := strings.TrimSpace(e.Request.Form.Get("old_password"))
		newPassword := strings.TrimSpace(e.Request.Form.Get("new_password"))
		confirmPassword := strings.TrimSpace(e.Request.Form.Get("confirm_password"))

		d := UserPasswordData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Change Password"))
		d.Title = i18n.T(d.Language, "Change Password")
		d.Description = i18n.T(d.Language, "page.userpassword.description")
		d.Slug = "/settings/password"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		renderError := func(msg string) error {
			d.Error = msg
			html, err := registry.LoadFiles(userPasswordTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		// Validate inputs
		if oldPassword == "" || newPassword == "" || confirmPassword == "" {
			return renderError("All fields are required.")
		}
		if newPassword != confirmPassword {
			return renderError("New passwords do not match.")
		}
		if len(newPassword) < 8 {
			return renderError("New password must be at least 8 characters.")
		}

		userID := authenticatedUserID(e)

		ctx := e.Request.Context()
		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return renderError("Could not load account.")
		}

		// Verify old password
		matched, _ := auth.CheckPassword(user.PasswordHash, user.OldPassword, oldPassword)
		if !matched {
			return renderError("Current password is incorrect.")
		}

		// Hash new password
		newHash, err := auth.HashPassword(newPassword)
		if err != nil {
			return renderError("Failed to update password. Please try again.")
		}

		if err := appStore.Users.UpdateUserPassword(ctx, userID, newHash); err != nil {
			return renderError("Failed to update password. Please try again.")
		}

		d.Success = true
		html, renderErr := registry.LoadFiles(userPasswordTemplates...).Render(d)
		if renderErr != nil {
			return renderErr
		}
		return e.HTML(http.StatusOK, html)
	}
}
