package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
)

func FollowHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		followedID := e.Request.PathValue("id")
		followerID := authenticatedUserID(e)

		if followerID == followedID {
			return &server.APIError{Status: http.StatusBadRequest, Message: "Cannot follow yourself"}
		}

		ctx := e.Request.Context()
		target, err := appStore.Users.GetUserByID(ctx, followedID)
		if err != nil || target == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "User not found"}
		}

		if err := appStore.Follows.Follow(ctx, followerID, followedID); err != nil {
			return err
		}

		lang := preferredLanguageFromRequest(e.Request)
		return e.HTML(http.StatusOK, fmt.Sprintf(
			`<button class="btn btn-sm btn-outline-secondary" hx-delete="/api/users/%s/follow" hx-swap="outerHTML" hx-target="this">%s</button>`,
			followedID, i18n.T(lang, "Unfollow"),
		))
	}
}

func UnfollowHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		followedID := e.Request.PathValue("id")
		followerID := authenticatedUserID(e)

		ctx := e.Request.Context()
		if err := appStore.Follows.Unfollow(ctx, followerID, followedID); err != nil {
			return err
		}

		lang := preferredLanguageFromRequest(e.Request)
		return e.HTML(http.StatusOK, fmt.Sprintf(
			`<button class="btn btn-sm btn-primary" hx-post="/api/users/%s/follow" hx-swap="outerHTML" hx-target="this">%s</button>`,
			followedID, i18n.T(lang, "Follow"),
		))
	}
}
