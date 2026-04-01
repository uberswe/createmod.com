package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strings"
	"unicode/utf8"
)

const maxModerationMessageLen = 2000
const maxUserMessagesSinceLastMod = 5

func ModerationChatCreateHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		ctx := context.Background()
		userID := authenticatedUserID(e)
		schematicID := e.Request.PathValue("id")
		if schematicID == "" {
			return e.BadRequestError("Missing schematic ID", nil)
		}

		// Get the schematic
		s, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || s == nil {
			return e.NotFoundError("Schematic not found", nil)
		}

		// Check caller is owner or admin
		user, err := appStore.Users.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return e.UnauthorizedError("User not found", nil)
		}
		isAdmin := user.IsAdmin
		isOwner := s.AuthorID == userID
		if !isOwner && !isAdmin {
			return e.ForbiddenError("Not authorized", nil)
		}

		// Parse and validate body
		body := strings.TrimSpace(e.Request.FormValue("body"))
		if body == "" {
			return e.BadRequestError("Message body is required", nil)
		}
		if utf8.RuneCountInString(body) > maxModerationMessageLen {
			return e.BadRequestError("Message too long (max 2000 characters)", nil)
		}

		// Get or create thread
		thread, err := appStore.ModerationChats.GetThreadByContent(ctx, "schematic", schematicID)
		if err != nil {
			// No thread yet, create one
			thread, err = appStore.ModerationChats.CreateThread(ctx, "schematic", schematicID)
			if err != nil {
				return e.InternalServerError("Failed to create thread", err)
			}
		}

		// Spam limit: non-admin users can only post 5 messages since last moderator message
		if !isAdmin {
			count, err := appStore.ModerationChats.CountUserMessagesSinceLastModerator(ctx, thread.ID)
			if err != nil {
				return e.InternalServerError("Failed to check message count", err)
			}
			if count >= maxUserMessagesSinceLastMod {
				return e.BadRequestError("You can send up to 5 messages before a moderator responds. Please wait for a moderator reply.", nil)
			}
		}

		// Create message
		_, err = appStore.ModerationChats.CreateMessage(ctx, thread.ID, userID, isAdmin, body)
		if err != nil {
			return e.InternalServerError("Failed to create message", err)
		}

		// For HTMX requests, trigger a page refresh of the chat section
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Refresh", "true")
			return e.HTML(http.StatusNoContent, "")
		}

		return e.Redirect(http.StatusSeeOther, "/schematics/"+s.Name)
	}
}
