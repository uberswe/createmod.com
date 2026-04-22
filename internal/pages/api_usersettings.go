package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/session"
	"createmod/internal/store"
	"createmod/internal/server"
	"log/slog"
	"net/http"
	"strings"

	"github.com/drexedam/gravatar"
)

// UserUpdateHandler handles PATCH /api/users/{id} to update user profile.
// Replaces PB's PATCH /api/collections/users/records/{id} endpoint.
func UserUpdateHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		targetID := e.Request.PathValue("id")

		// Users can only update their own profile
		if userID == "" || userID != targetID {
			return e.ForbiddenError("", nil)
		}

		if err := e.Request.ParseMultipartForm(1 << 20); err != nil {
			if err2 := e.Request.ParseForm(); err2 != nil {
				return e.BadRequestError("invalid form data", nil)
			}
		}

		ctx := context.Background()
		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return e.NotFoundError("user not found", nil)
		}

		username := strings.TrimSpace(e.Request.FormValue("username"))
		if username != "" {
			user.Username = username
		}

		// Update gravatar
		avatarURL := gravatar.New(user.Email).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		user.Avatar = avatarURL

		if err := appStore.Users.UpdateUser(ctx, user); err != nil {
			slog.Error("failed to update user", "error", err)
			return e.InternalServerError("could not update profile", nil)
		}

		if e.Request.Header.Get("HX-Request") != "" {
			return e.NoContent(http.StatusNoContent)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

// UserDeleteHandler handles DELETE /api/users/{id} to delete a user account.
// Replaces PB's DELETE /api/collections/users/records/{id} endpoint.
func UserDeleteHandler(appStore *store.Store, cacheService *cache.Service, sessStore *session.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		targetID := e.Request.PathValue("id")

		// Users can only delete their own account
		if userID == "" || userID != targetID {
			return e.ForbiddenError("", nil)
		}

		ctx := context.Background()

		// Invalidate schematic caches before cascade delete
		schematics, err := appStore.Schematics.ListByAuthor(ctx, userID, -1, 0)
		if err != nil {
			slog.Error("failed to list user schematics for cache invalidation", "error", err)
		} else {
			for _, schem := range schematics {
				cacheService.DeleteSchematic(cache.SchematicKey(schem.ID))
			}
			if len(schematics) > 0 {
				RefreshIndexCache(cacheService, appStore, []int{7})
			}
		}

		// Cascade soft-delete: user + schematics, collections, comments, ratings, guides
		if err := appStore.Users.CascadeSoftDelete(ctx, userID); err != nil {
			slog.Error("failed to cascade soft-delete user", "error", err)
			return e.InternalServerError("could not delete account", nil)
		}

		// Delete all sessions for this user
		_ = sessStore.DeleteUserSessions(ctx, userID)

		if e.Request.Header.Get("HX-Request") != "" {
			return e.NoContent(http.StatusNoContent)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
