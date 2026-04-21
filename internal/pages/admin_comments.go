package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var adminCommentsTemplates = append([]string{
	"./template/admin_comments.html",
}, commonTemplates...)

type AdminCommentItem struct {
	ID             string
	Content        string
	Approved       bool
	Deleted        *time.Time
	AuthorID       string
	AuthorUsername string
	AuthorAvatar   string
	SchematicID    string
	SchematicName  string
	SchematicTitle string
	SchematicURL   string
	Created        time.Time
}

type AdminCommentsData struct {
	DefaultData
	Comments   []AdminCommentItem
	Filter     string
	Search     string
	Page       int
	TotalPages int
	Total      int64
	PrevPage   int
	NextPage   int
	QueryStr   string // encoded filter+search for pagination links
}

const adminCommentsPerPage = 30

// AdminCommentsHandler renders the admin comment listing page at GET /admin/comments.
func AdminCommentsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		filter := e.Request.URL.Query().Get("filter")
		switch filter {
		case "all", "active", "deleted", "unapproved":
		default:
			filter = "active"
		}

		search := strings.TrimSpace(e.Request.URL.Query().Get("q"))

		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * adminCommentsPerPage

		comments, err := appStore.Comments.ListForAdmin(ctx, filter, search, adminCommentsPerPage, offset)
		if err != nil {
			slog.Error("admin comments: list failed", "error", err)
			return e.String(http.StatusInternalServerError, "failed to list comments")
		}

		total, err := appStore.Comments.CountForAdmin(ctx, filter, search)
		if err != nil {
			slog.Error("admin comments: count failed", "error", err)
			return e.String(http.StatusInternalServerError, "failed to count comments")
		}

		totalPages := int(total) / adminCommentsPerPage
		if int(total)%adminCommentsPerPage != 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		items := make([]AdminCommentItem, 0, len(comments))
		for _, c := range comments {
			authorID := ""
			if c.AuthorID != nil {
				authorID = *c.AuthorID
			}
			schematicID := ""
			if c.SchematicID != nil {
				schematicID = *c.SchematicID
			}
			schURL := ""
			if c.SchematicName != "" {
				schURL = "/schematics/" + c.SchematicName + "#comment-" + c.ID
			}
			items = append(items, AdminCommentItem{
				ID:             c.ID,
				Content:        c.Content,
				Approved:       c.Approved,
				Deleted:        c.Deleted,
				AuthorID:       authorID,
				AuthorUsername: c.AuthorUsername,
				AuthorAvatar:   c.AuthorAvatar,
				SchematicID:    schematicID,
				SchematicName:  c.SchematicName,
				SchematicTitle: c.SchematicTitle,
				SchematicURL:   schURL,
				Created:        c.Created,
			})
		}

		q := url.Values{}
		q.Set("filter", filter)
		if search != "" {
			q.Set("q", search)
		}

		d := AdminCommentsData{
			Comments:   items,
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
		d.AdminSection = "comments"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Comments"))
		d.Title = i18n.T(d.Language, "Admin: Comments")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminCommentsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminCommentDeleteHandler soft-deletes a comment at POST /admin/comments/{id}/delete.
func AdminCommentDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
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
		if err := appStore.Comments.Delete(context.Background(), id); err != nil {
			slog.Error("admin comments: delete failed", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to delete comment")
		}
		return adminCommentsRedirect(e)
	}
}

// AdminCommentRestoreHandler restores a soft-deleted comment at POST /admin/comments/{id}/restore.
func AdminCommentRestoreHandler(appStore *store.Store) func(e *server.RequestEvent) error {
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
		if err := appStore.Comments.Restore(context.Background(), id); err != nil {
			slog.Error("admin comments: restore failed", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to restore comment")
		}
		return adminCommentsRedirect(e)
	}
}

// AdminCommentApproveHandler approves a pending comment at POST /admin/comments/{id}/approve.
func AdminCommentApproveHandler(appStore *store.Store) func(e *server.RequestEvent) error {
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
		if err := appStore.Comments.Approve(context.Background(), id); err != nil {
			slog.Error("admin comments: approve failed", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to approve comment")
		}
		return adminCommentsRedirect(e)
	}
}

func adminCommentsRedirect(e *server.RequestEvent) error {
	dest := strings.TrimSpace(e.Request.FormValue("return"))
	if dest == "" || !strings.HasPrefix(dest, "/admin/comments") {
		dest = "/admin/comments"
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", dest)
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, dest)
}
