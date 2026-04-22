package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var adminSchematicsTemplates = append([]string{
	"./template/admin_schematics.html",
}, commonTemplates...)

var adminSchematicEditTemplates = append([]string{
	"./template/admin_schematic_edit.html",
}, commonTemplates...)

type AdminSchematicItem struct {
	ID               string
	Title            string
	Name             string
	AuthorUsername   string
	ModerationState  string
	ModerationReason string
	Created          time.Time
	FeaturedImage    string
}

type AdminSchematicsData struct {
	DefaultData
	Schematics []AdminSchematicItem
	Filter     string
	Page       int
	TotalPages int
	Total      int64
	PrevPage   int
	NextPage   int
}

type AdminSchematicEditData struct {
	DefaultData
	Schematic           store.Schematic
	AuthorUsername       string
	MinecraftVersions   []models.MinecraftVersion
	CreatemodVersions   []models.CreatemodVersion
	Tags                []models.SchematicTag
	Categories          []models.SchematicCategory
	SchematicCategories []string
	SchematicTags       []string
	CreatemodVersionID  string
	MinecraftVersionID  string
	Success             bool
}

const adminSchematicsPerPage = 20

// AdminSchematicsHandler renders the admin schematic listing page at GET /admin/schematics.
func AdminSchematicsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		filter := e.Request.URL.Query().Get("filter")
		if filter == "" {
			filter = "pending"
		}
		if filter != "all" && filter != "pending" && filter != "published" && filter != "flagged" && filter != "rejected" && filter != "deleted" {
			filter = "pending"
		}

		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * adminSchematicsPerPage

		schematics, err := appStore.Schematics.ListForAdmin(ctx, filter, adminSchematicsPerPage, offset)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list schematics")
		}

		total, err := appStore.Schematics.CountForAdmin(ctx, filter)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to count schematics")
		}

		totalPages := int(total) / adminSchematicsPerPage
		if int(total)%adminSchematicsPerPage != 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		items := make([]AdminSchematicItem, 0, len(schematics))
		for _, s := range schematics {
			username := ""
			if s.AuthorID != "" {
				u, err := appStore.Users.GetUserByID(ctx, s.AuthorID)
				if err == nil && u != nil {
					username = u.Username
				}
			}
			items = append(items, AdminSchematicItem{
				ID:               s.ID,
				Title:            s.Title,
				Name:             s.Name,
				AuthorUsername:   username,
				ModerationState:  s.ModerationState,
				ModerationReason: s.ModerationReason,
				Created:          s.Created,
				FeaturedImage:    s.FeaturedImage,
			})
		}

		d := AdminSchematicsData{
			Schematics: items,
			Filter:     filter,
			Page:       page,
			TotalPages: totalPages,
			Total:      total,
			PrevPage:   page - 1,
			NextPage:   page + 1,
		}
		d.Populate(e)
		d.AdminSection = "schematics"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Schematics"))
		d.Title = i18n.T(d.Language, "Admin: Schematics")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminSchematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminSchematicEditHandler renders the admin schematic edit page at GET /admin/schematics/{id}.
func AdminSchematicEditHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		schem, err := appStore.Schematics.GetByIDAdmin(ctx, id)
		if err != nil || schem == nil {
			return e.String(http.StatusNotFound, "schematic not found")
		}

		authorUsername := ""
		if schem.AuthorID != "" {
			u, err := appStore.Users.GetUserByID(ctx, schem.AuthorID)
			if err == nil && u != nil {
				authorUsername = u.Username
			}
		}

		catIDs, _ := appStore.Schematics.GetCategoryIDs(ctx, id)
		tagIDs, _ := appStore.Schematics.GetTagIDs(ctx, id)

		cmVersionID := ""
		if schem.CreatemodVersionID != nil {
			cmVersionID = *schem.CreatemodVersionID
		}
		mcVersionID := ""
		if schem.MinecraftVersionID != nil {
			mcVersionID = *schem.MinecraftVersionID
		}

		d := AdminSchematicEditData{
			Schematic:           *schem,
			AuthorUsername:       authorUsername,
			MinecraftVersions:   allMinecraftVersionsFromStore(appStore),
			CreatemodVersions:   allCreatemodVersionsFromStore(appStore),
			Tags:                allTagsFromStore(appStore),
			Categories:          allCategoriesFromStoreOnly(appStore, cacheService),
			SchematicCategories: catIDs,
			SchematicTags:       tagIDs,
			CreatemodVersionID:  cmVersionID,
			MinecraftVersionID:  mcVersionID,
			Success:             e.Request.URL.Query().Get("success") == "1",
		}
		d.Populate(e)
		d.AdminSection = "schematics"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Schematics"), "/admin/schematics", i18n.T(d.Language, "Edit"))
		d.Title = fmt.Sprintf("Admin: Edit — %s", schem.Title)
		d.SubCategory = "Admin"

		html, err := registry.LoadFiles(adminSchematicEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminSchematicUpdateHandler handles POST /admin/schematics/{id} to update a schematic as admin.
func AdminSchematicUpdateHandler(cacheService *cache.Service, appStore *store.Store, mailService *mailer.Service, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		schem, err := appStore.Schematics.GetByIDAdmin(ctx, id)
		if err != nil || schem == nil {
			return e.String(http.StatusNotFound, "schematic not found")
		}

		// Track whether state changes from non-public to public
		wasPublic := store.IsPublicState(schem.ModerationState)

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form data")
		}

		// Read form fields
		if title := strings.TrimSpace(e.Request.FormValue("title")); title != "" {
			schem.Title = title
		}
		if content := e.Request.FormValue("content"); content != "" {
			schem.Content = content
		}
		schem.Video = strings.TrimSpace(e.Request.FormValue("video"))

		// Versions
		if cmv := strings.TrimSpace(e.Request.FormValue("createmod_version")); cmv != "" {
			schem.CreatemodVersionID = &cmv
		}
		if mcv := strings.TrimSpace(e.Request.FormValue("minecraft_version")); mcv != "" {
			schem.MinecraftVersionID = &mcv
		}

		// Paid / External URL
		schem.Paid = e.Request.FormValue("paid") == "on"
		schem.ExternalURL = strings.TrimSpace(e.Request.FormValue("external_url"))
		if !schem.Paid {
			schem.ExternalURL = ""
		}

		// Image removal
		if e.Request.FormValue("remove_featured_image") == "true" && schem.FeaturedImage != "" {
			if storageSvc != nil {
				if delErr := storageSvc.Delete(ctx, s3CollectionSchematics, id, schem.FeaturedImage); delErr != nil {
					slog.Warn("admin schematic update: failed to delete featured image from S3", "error", delErr, "id", id)
				}
			}
			schem.FeaturedImage = ""
		}
		if removeGalleryImages := e.Request.Form["remove_gallery_images"]; len(removeGalleryImages) > 0 {
			removeSet := make(map[string]bool, len(removeGalleryImages))
			for _, fn := range removeGalleryImages {
				removeSet[fn] = true
			}
			filtered := make([]string, 0, len(schem.Gallery))
			for _, fn := range schem.Gallery {
				if removeSet[fn] {
					if storageSvc != nil {
						if delErr := storageSvc.Delete(ctx, s3CollectionSchematics, id, fn); delErr != nil {
							slog.Warn("admin schematic update: failed to delete gallery image from S3", "error", delErr, "id", id, "file", fn)
						}
					}
					continue
				}
				filtered = append(filtered, fn)
			}
			schem.Gallery = filtered
		}

		// Moderation controls
		if newState := e.Request.FormValue("moderation_state"); newState != "" {
			schem.ModerationState = newState
		}
		if reason := e.Request.FormValue("moderation_reason"); reason != "" {
			schem.ModerationReason = reason
		}
		schem.Featured = e.Request.FormValue("featured") == "on"

		// Update modified timestamp
		now := time.Now()
		schem.Modified = &now
		schem.Updated = now

		if err := appStore.Schematics.Update(ctx, schem); err != nil {
			slog.Error("admin schematic update: failed to update", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to update schematic")
		}

		// Update categories and tags
		categories := resolveCategoryIDs(ctx, appStore, e.Request.Form["categories"])
		tags := resolveTagIDs(ctx, appStore, e.Request.Form["tags"])
		if len(categories) > 0 {
			if err := appStore.Schematics.SetCategories(ctx, id, categories); err != nil {
				slog.Warn("admin schematic update: failed to set categories", "error", err, "id", id)
			}
		}
		if len(tags) > 0 {
			if err := appStore.Schematics.SetTags(ctx, id, tags); err != nil {
				slog.Warn("admin schematic update: failed to set tags", "error", err, "id", id)
			}
		}

		// Clear cache
		cacheService.DeleteSchematic(cache.SchematicKey(id))
		RefreshIndexCache(cacheService, appStore, []int{7})

		// If moderation just changed from non-public to public, notify the author
		if !wasPublic && store.IsPublicState(schem.ModerationState) && mailService != nil && schem.AuthorID != "" {
			authorID := schem.AuthorID
			emailTitle := schem.Title
			emailID := schem.ID
			emailName := schem.Name
			emailImage := schem.FeaturedImage
			go func() {
				author, err := appStore.Users.GetUserByID(context.Background(), authorID)
				if err != nil || author == nil || author.Email == "" {
					return
				}
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "https://createmod.com"
				}
				var imageURL string
				if emailImage != "" {
					imageURL = fmt.Sprintf("%s/api/files/schematics/%s/%s", baseURL, emailID, url.PathEscape(emailImage))
				}
				schematicURL := fmt.Sprintf("%s/schematics/%s", baseURL, emailName)

				from := mailService.DefaultFrom()
				to := []mail.Address{{Address: author.Email}}
				subject := fmt.Sprintf("Your schematic has been published: %s", emailTitle)
				bodyText := fmt.Sprintf("Your schematic \"%s\" has been reviewed and approved by an admin. It is now live on CreateMod.com!", emailTitle)
				body := mailer.SchematicEmailHTML(emailTitle, imageURL, schematicURL, bodyText)
				msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
				if err := mailService.Send(msg); err != nil {
					slog.Error("admin schematic update: failed to send author notification", "error", err)
				}
			}()
		}

		// Redirect back to edit page with success
		dest := fmt.Sprintf("/admin/schematics/%s?success=1", id)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// AdminSchematicDeleteHandler handles POST /admin/schematics/{id}/delete to soft-delete a schematic.
func AdminSchematicDeleteHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		if err := appStore.Schematics.SoftDelete(context.Background(), id); err != nil {
			slog.Error("admin schematic delete: failed to soft-delete", "error", err, "id", id)
			return e.String(http.StatusInternalServerError, "failed to delete schematic")
		}

		cacheService.DeleteSchematic(cache.SchematicKey(id))
		RefreshIndexCache(cacheService, appStore, []int{7})

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/admin/schematics")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/admin/schematics")
	}
}
