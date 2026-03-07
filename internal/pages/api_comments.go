package pages

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/sym01/htmlsanitizer"
)

// CommentCreateHandler handles POST /api/comments to create a new comment.
// Replaces PB's POST /api/collections/comments/records endpoint.
func CommentCreateHandler(appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("", nil)
		}

		if err := e.Request.ParseMultipartForm(2 << 20); err != nil {
			if err2 := e.Request.ParseForm(); err2 != nil {
				return e.BadRequestError("invalid form data", nil)
			}
		}

		schematicID := e.Request.FormValue("schematic")
		content := e.Request.FormValue("content")
		parentID := e.Request.FormValue("parent")

		if schematicID == "" || content == "" {
			return e.BadRequestError("schematic and content are required", nil)
		}

		ctx := context.Background()

		// Validate schematic exists
		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return e.BadRequestError("invalid schematic", nil)
		}

		// Validate parent comment if replying
		replyToUser := ""
		if parentID != "" {
			comments, err := appStore.Comments.ListBySchematic(ctx, schematicID)
			if err != nil {
				return e.BadRequestError("could not validate parent comment", nil)
			}
			found := false
			for _, c := range comments {
				if c.ID == parentID && c.AuthorID != nil {
					replyToUser = *c.AuthorID
					found = true
					break
				}
			}
			if !found {
				return e.BadRequestError("invalid parent comment", nil)
			}
		}

		// Sanitize content
		sanitizer := htmlsanitizer.NewHTMLSanitizer()
		sanitizedContent, err := sanitizer.SanitizeString(content)
		if err != nil {
			return e.BadRequestError("invalid content", nil)
		}

		now := time.Now()
		comment := &store.Comment{
			AuthorID:    &userID,
			SchematicID: &schematicID,
			Content:     sanitizedContent,
			Published:   &now,
			Approved:    true,
			Type:        "comment",
		}
		if parentID != "" {
			comment.ParentID = &parentID
		}

		if err := appStore.Comments.Create(ctx, comment); err != nil {
			slog.Error("failed to create comment", "error", err)
			return e.InternalServerError("could not save comment", nil)
		}

		// Award first_comment achievement
		go func() {
			ctx := context.Background()
			count, err := appStore.Comments.CountByUser(ctx, userID)
			if err != nil || count != 1 {
				return
			}
			ach, err := appStore.Achievements.GetByKey(ctx, "first_comment")
			if err != nil || ach == nil {
				return
			}
			has, _ := appStore.Achievements.HasAchievement(ctx, userID, ach.ID)
			if has {
				return
			}
			_ = appStore.Achievements.Award(ctx, userID, ach.ID)
			if u, err := appStore.Users.GetUserByID(ctx, userID); err == nil {
				_ = appStore.Users.UpdateUserPoints(ctx, userID, u.Points+10)
			}
			_ = appStore.Achievements.CreatePointLog(ctx, &store.PointLogEntry{
				UserID:      userID,
				Points:      10,
				Reason:      "first_comment",
				Description: "Posted your first comment",
				EarnedAt:    time.Now(),
			})
		}()

		// Send email notification
		go func() {
			if mailService == nil {
				return
			}
			ctx := context.Background()

			var recipientEmail, subject, body string
			if replyToUser == "" {
				// Notify schematic author
				u, err := appStore.Users.GetUserByID(ctx, schem.AuthorID)
				if err != nil || u == nil {
					return
				}
				recipientEmail = u.Email
				subject = fmt.Sprintf("New comment on %s", schem.Title)
				body = fmt.Sprintf("<p>A new comment has been posted on your CreateMod.com schematic: <a href=\"https://www.createmod.com/schematics/%s\">https://www.createmod.com/schematics/%s</a></p>", schem.Name, schem.Name)
			} else {
				// Notify parent comment author
				u, err := appStore.Users.GetUserByID(ctx, replyToUser)
				if err != nil || u == nil {
					return
				}
				recipientEmail = u.Email
				subject = fmt.Sprintf("New reply on %s", schem.Title)
				body = fmt.Sprintf("<p>A new reply has been posted to your comment on CreateMod.com: <a href=\"https://www.createmod.com/schematics/%s\">https://www.createmod.com/schematics/%s</a></p>", schem.Name, schem.Name)
			}

			from := mail.Address{Address: mailService.SenderAddress, Name: mailService.SenderName}
			to := []mail.Address{{Address: recipientEmail}}
			msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
			if err := mailService.Send(msg); err != nil {
				slog.Error("failed to send comment notification", "error", err)
			}
		}()

		// Return 204 for HTMX
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Trigger", "commentCreated")
			return e.NoContent(http.StatusNoContent)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

// CommentDeleteHandler handles DELETE /api/comments/{id} to delete a comment.
// Only the comment author or an admin may delete.
func CommentDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("", nil)
		}

		commentID := e.Request.PathValue("id")
		if commentID == "" {
			return e.BadRequestError("comment id is required", nil)
		}

		ctx := context.Background()

		comment, err := appStore.Comments.GetByID(ctx, commentID)
		if err != nil || comment == nil {
			return e.NotFoundError("comment not found", nil)
		}

		// Only the author or an admin may delete
		isAuthor := comment.AuthorID != nil && *comment.AuthorID == userID
		user, _ := appStore.Users.GetUserByID(ctx, userID)
		isAdmin := user != nil && user.IsAdmin

		if !isAuthor && !isAdmin {
			return e.ForbiddenError("you are not allowed to delete this comment", nil)
		}

		if err := appStore.Comments.Delete(ctx, commentID); err != nil {
			slog.Error("failed to delete comment", "error", err)
			return e.InternalServerError("could not delete comment", nil)
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Trigger", "commentDeleted")
			return e.NoContent(http.StatusNoContent)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
