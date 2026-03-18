package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/server"
	"net/http"
)

const uploadPublishTemplate = "./template/upload_publish.html"

var uploadPublishTemplates = append([]string{
	uploadPublishTemplate,
	uploadStepsTemplate,
}, commonTemplates...)

// UploadPublishHandler renders the publish form for a previously uploaded temp schematic.
// Requires authentication; redirects to /login if not logged in.
func UploadPublishHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		// Resolve entry from PostgreSQL store
		entry, err := appStore.TempUploads.GetByToken(e.Request.Context(), token)
		if err != nil {
			return e.String(http.StatusNotFound, "invalid or expired token")
		}

		// Load additional files from store
		storeFiles, _ := appStore.TempUploadFiles.ListByToken(e.Request.Context(), token)
		additionalFiles := mapStoreTempUploadFiles(storeFiles)

		d := UploadPublishData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Upload"), "/upload", i18n.T(d.Language, "Publish"))
		d.UploadStep = 3
		d.Title = i18n.T(d.Language, "Publish Schematic")
		d.Description = i18n.T(d.Language, "page.upload.publish.description")
		d.Slug = "/u/" + token + "/publish"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Token = token
		d.Filename = entry.Filename
		d.Size = entry.Size
		d.BlockCount = entry.BlockCount
		d.DimX = entry.DimX
		d.DimY = entry.DimY
		d.DimZ = entry.DimZ
		d.Tags = allTagsFromStore(appStore)
		d.MinecraftVersions = allMinecraftVersionsFromStore(appStore)
		d.CreatemodVersions = allCreatemodVersionsFromStore(appStore)
		d.AdditionalFiles = additionalFiles

		// Check if user has previously approved schematics (trusted user)
		userID := authenticatedUserID(e)
		if userID != "" {
			authorCount, countErr := appStore.Schematics.CountByAuthor(e.Request.Context(), userID)
			if countErr == nil && authorCount > 0 {
				d.TrustedUser = true
			}
		}

		html, err := registry.LoadFiles(uploadPublishTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
