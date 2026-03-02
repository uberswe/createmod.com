package router

import (
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/i18n"
	"createmod/internal/moderation"
	"createmod/internal/modmeta"
	"createmod/internal/outurl"
	"createmod/internal/pages"
	"createmod/internal/promotion"
	"createmod/internal/search"
	"createmod/internal/session"
	"createmod/internal/store"
	"createmod/internal/translation"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/template"
	html "html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// computeAssetVersion hashes local CSS files and returns a short hex string.
// Called once at startup so templates can append ?v=<hash> for cache-busting.
func computeAssetVersion() string {
	h := sha256.New()
	for _, path := range []string{"./template/static/style.css", "./template/static/app.css"} {
		data, err := os.ReadFile(path)
		if err == nil {
			h.Write(data)
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:8]
}

func Register(app *pocketbase.PocketBase, e *router.Router[*core.RequestEvent], searchService *search.Service, cacheService *cache.Service, discordService *discord.Service, moderationService *moderation.Service, translationService *translation.Service, modMetaService *modmeta.Service, appStore *store.Store, sessionStore *session.Store, discordOAuth *auth.OAuthProvider, githubOAuth *auth.OAuthProvider) {
	promotionService := promotion.New()
	registry := template.NewRegistry()

	assetVer := computeAssetVersion()

	// Derive a stable HMAC key for signing outgoing redirect URLs.
	outSecret := deriveOutSecret(app)

	funcMap := html.FuncMap{
		"ToLower":   strings.ToLower,
		"mod":       func(i, j int) bool { return i%j == 0 },
		"HumanDate": func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 MST") },
		"DateOnly":  func(t time.Time) string { return t.UTC().Format("2006-01-02") },
		"printf":    fmt.Sprintf,
		"T":         func(lang string, key string) string { return i18n.T(lang, key) },
		"AssetVer":  func() string { return assetVer },
		"SignedOutURL": func(rawURL string, args ...string) string {
			if len(args) == 2 {
				return outurl.BuildPathWithSource(rawURL, outSecret, args[0], args[1])
			}
			return outurl.BuildPath(rawURL, outSecret)
		},
		"tagSelected": func(selected []string, key string) bool {
			for _, s := range selected {
				if s == key {
					return true
				}
			}
			return false
		},
		"LangURL": func(lang string, path string) string {
			return pages.PrefixedPath(lang, path)
		},
		"Hreflangs": func(barePath string) []pages.HreflangEntry {
			return pages.AllHreflangs()
		},
		"LangFlag": func(code string) string {
			switch code {
			case "en":
				return "\U0001F1EC\U0001F1E7"
			case "pt-BR":
				return "\U0001F1E7\U0001F1F7"
			case "pt-PT":
				return "\U0001F1F5\U0001F1F9"
			case "es":
				return "\U0001F1EA\U0001F1F8"
			case "de":
				return "\U0001F1E9\U0001F1EA"
			case "pl":
				return "\U0001F1F5\U0001F1F1"
			case "ru":
				return "\U0001F1F7\U0001F1FA"
			case "zh-Hans":
				return "\U0001F1E8\U0001F1F3"
			default:
				return "\U0001F310"
			}
		},
	}

	registry.AddFuncs(funcMap)

	e.BindFunc(legacyFileCompat())
	e.BindFunc(legacySearchCompat())
	e.BindFunc(legacyCategoryCompat())
	e.BindFunc(legacyTagCompat())
	e.BindFunc(cookieAuth(app, sessionStore))
	// Frontend routes
	e.GET("/sitemaps/{path...}", apis.Static(os.DirFS("./template/dist/sitemaps"), false))
	e.GET("/assets/x/{path...}", apis.Static(os.DirFS("./template/static"), false))
	// Serve unbundled source assets directly (no npm build)
	e.GET("/robots.txt", func(e *core.RequestEvent) error {
		return e.String(200, "User-agent: *\nDisallow: /_/\nAllow: /\nSitemap: https://createmod.com/sitemaps/sitemap.xml")
	})
	e.GET("/ads.txt", func(e *core.RequestEvent) error {
		s, ok := cacheService.GetString("ads.txt")
		if ok {
			return e.String(200, s)
		}
		s, err := getContent("https://api.nitropay.com/v1/ads-2143.txt")
		if err != nil || s == "" {
			return e.String(500, "Could not determine content")
		}
		cacheService.SetString("ads.txt", s)
		return e.String(200, s)
	})
	// Index
	e.GET("/", pages.IndexHandler(app, cacheService, registry, appStore))
	// Removed the about page, not relevant anymore
	e.GET("/upload", pages.UploadHandler(app, registry, cacheService, appStore))
	e.POST("/upload/nbt", pages.UploadNBTHandler(app, registry, cacheService, appStore))
	// Private preview URL for temporary uploads
	e.GET("/u/{token}", pages.UploadPreviewHandler(app, registry, cacheService, appStore))
	// Make public endpoint for temporary uploads
	e.GET("/u/{token}/download", pages.UploadDownloadHandler(app, appStore))
	e.POST("/u/{token}/add-file", pages.UploadAddFileHandler(app, appStore))
	e.DELETE("/u/{token}/files/{fileId}", pages.UploadDeleteFileHandler(app, appStore))
	e.GET("/u/{token}/files/{fileId}/download", pages.UploadFileDownloadHandler(app, appStore))
	e.POST("/u/{token}/make-public", pages.UploadMakePublicHandler(app, registry, cacheService, appStore))
	// Publish form for temporary uploads (requires auth)
	e.GET("/u/{token}/publish", pages.UploadPublishHandler(app, registry, cacheService, appStore))
	// Upload moderation pending confirmation page
	e.GET("/upload/pending", pages.UploadPendingHandler(app, registry, cacheService, appStore))
	e.GET("/contact", pages.ContactHandler(app, registry, cacheService, appStore))
	e.GET("/blacklist-request", func(e *core.RequestEvent) error {
		return e.Redirect(http.StatusMovedPermanently, pages.LangRedirectURL(e, "/settings/blacklist"))
	})
	// Redirect legacy single guide page to the guides listing
	e.GET("/guide", func(e *core.RequestEvent) error {
		return e.Redirect(http.StatusMovedPermanently, pages.LangRedirectURL(e, "/guides"))
	})
	e.GET("/rules", pages.RulesHandler(app, registry, cacheService, appStore))
	e.GET("/explore", pages.ExploreHandler(app, cacheService, registry, appStore))
	e.GET("/terms-of-service", pages.TermsOfServiceHandler(app, registry, cacheService, appStore))
	e.GET("/privacy-policy", pages.PrivacyPolicyHandler(app, registry, cacheService, appStore))
	e.GET("/settings", pages.UserSettingsHandler(app, registry, cacheService, appStore))
	e.GET("/settings/password", pages.UserPasswordHandler(app, registry, cacheService, appStore))
	e.POST("/settings/password", pages.UserPasswordPostHandler(app, registry, cacheService, appStore))
	e.GET("/settings/points", pages.UserPointsHandler(app, registry, cacheService, appStore))
	e.GET("/settings/gamification", func(e *core.RequestEvent) error {
		return e.Redirect(http.StatusMovedPermanently, pages.LangRedirectURL(e, "/settings/points"))
	})
	e.GET("/settings/api-keys", pages.UserAPIKeysHandler(app, registry, cacheService, appStore))
	e.GET("/settings/statistics", pages.UserStatsHandler(app, registry, cacheService, appStore))
	e.GET("/settings/blacklist", pages.BlacklistRequestHandler(app, registry, cacheService, appStore))
	// API Docs
	e.GET("/api", pages.APIDocsHandler(app, registry, cacheService, appStore))
	// Public JSON API (beta)
	e.GET("/api/schematics", pages.APISchematicsListHandler(app, searchService, cacheService, appStore))
	e.GET("/api/schematics/{name}", pages.APISchematicDetailHandler(app, cacheService, appStore))
	e.POST("/api/schematics/upload", pages.APIUploadHandler(app, cacheService, appStore))
	// Reports
	e.POST("/reports", pages.ReportSubmitHandler(app, appStore))
	// Admin
	e.GET("/admin/reports", pages.AdminReportsHandler(app, registry, cacheService, appStore))
	e.POST("/admin/reports/{id}/resolve", pages.AdminReportResolveHandler(app, appStore))
	// Auth
	e.GET("/login", pages.LoginHandler(app, registry, appStore))
	// Handle login form submissions
	e.POST("/login", pages.LoginPostHandler(app, appStore, sessionStore))
	e.GET("/register", pages.RegisterHandler(app, registry, appStore))
	e.POST("/register", pages.RegisterPostHandler(app, appStore, sessionStore))
	e.GET("/reset-password", pages.PasswordResetHandler(app, registry, appStore))
	e.POST("/reset-password", pages.PasswordResetPostHandler(app, registry, appStore))
	e.GET("/reset-password/{token}", pages.PasswordResetConfirmHandler(app, registry, appStore))
	e.POST("/reset-password/{token}", pages.PasswordResetConfirmPostHandler(app, registry, appStore, sessionStore))
	// OAuth routes
	e.GET("/auth/discord", pages.OAuthRedirectHandler(discordOAuth))
	e.GET("/auth/discord/callback", pages.OAuthCallbackHandler(app, discordOAuth, appStore, sessionStore))
	e.GET("/auth/github", pages.OAuthRedirectHandler(githubOAuth))
	e.GET("/auth/github/callback", pages.OAuthCallbackHandler(app, githubOAuth, appStore, sessionStore))
	e.GET("/logout", func(e *core.RequestEvent) error {
		secure := e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")

		// Delete PostgreSQL session
		if cookie, err := e.Request.Cookie(auth.CookieName); err == nil {
			_ = sessionStore.Delete(e.Request.Context(), cookie.Value)
		}

		auth.ClearAuthCookie(e.Response, secure)
		// Clear server-side auth for this request context
		e.Auth = nil
		if e.Request.Header.Get("HX-Request") != "" {
			// HTMX request: instruct client to navigate
			e.Response.Header().Set("HX-Redirect", pages.LangRedirectURL(e, "/"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusFound, pages.LangRedirectURL(e, "/"))
	})
	// News
	e.GET("/news", pages.NewsHandler(app, registry, cacheService, appStore))
	e.GET("/news/{slug}", pages.NewsPostHandler(app, registry, cacheService, appStore))
	// Users listing
	e.GET("/users", pages.UsersHandler(app, registry, cacheService, appStore))
	// Videos listing
	e.GET("/videos", pages.VideosHandler(app, registry, cacheService, appStore))
	// Guides listing
	e.GET("/guides", pages.GuidesHandler(app, registry, cacheService, outSecret, appStore))
	// Guide create
	e.GET("/guides/new", pages.GuidesNewHandler(app, registry, cacheService, appStore))
	e.POST("/guides", pages.GuidesCreateHandler(app, cacheService, appStore))
	// Guide detail/edit/update/delete
	e.GET("/guides/{id}", pages.GuidesShowHandler(app, registry, cacheService, translationService, appStore))
	e.GET("/guides/{id}/edit", pages.GuidesEditHandler(app, registry, cacheService, appStore))
	e.POST("/guides/{id}", pages.GuidesUpdateHandler(app, cacheService, appStore))
	e.POST("/guides/{id}/delete", pages.GuidesDeleteHandler(app, appStore))
	// Collections listing
	// Mods
	e.GET("/mods", pages.ModsHandler(app, cacheService, registry, modMetaService, appStore))
	e.GET("/mods/{slug}", pages.ModDetailHandler(app, cacheService, registry, modMetaService, appStore))
	e.GET("/collections", pages.CollectionsHandler(app, registry, cacheService, appStore))
	// Collections create flow
	e.GET("/collections/new", pages.CollectionsNewHandler(app, registry, cacheService, appStore))
	e.POST("/collections", pages.CollectionsCreateHandler(app, registry, cacheService, appStore))
	// Collections detail
	e.GET("/collections/{slug}", pages.CollectionsShowHandler(app, registry, cacheService, translationService, appStore))
	// Collections edit/update/delete
	e.GET("/collections/{slug}/edit", pages.CollectionsEditHandler(app, registry, cacheService, appStore))
	e.POST("/collections/{slug}", pages.CollectionsUpdateHandler(app, registry, cacheService, moderationService, appStore))
	e.POST("/collections/{slug}/delete", pages.CollectionsDeleteHandler(app, appStore))
	// Collections reorder (author-only)
	e.POST("/collections/{slug}/reorder", pages.CollectionsReorderHandler(app, appStore))
	// API keys (user settings)
	e.POST("/settings/api-keys/new", pages.APIKeyCreateHandler(app, cacheService, appStore))
	e.POST("/settings/api-keys/{id}/revoke", pages.APIKeyRevokeHandler(app, appStore))
	e.POST("/api/keys/generate", pages.APIKeyCreateJSONHandler(app, appStore))
	// Language setter
	e.GET("/lang", pages.SetLanguageHandler())
	// Schematics
	e.GET("/schematics", pages.SchematicsHandler(app, cacheService, registry, appStore))
	e.GET("/schematics/{name}", pages.SchematicHandler(app, searchService, cacheService, registry, promotionService, discordService, translationService, appStore))
	// Partial comments endpoint for HTMX refresh
	e.GET("/schematics/{name}/comments", pages.SchematicCommentsHandler(app, searchService, cacheService, registry, discordService, appStore))
	// Add to collection
	e.POST("/schematics/{name}/add-to-collection", pages.SchematicAddToCollectionHandler(app, appStore))
	// Download endpoint to track download metrics separately
	e.GET("/download/{name}", pages.DownloadHandler(app, cacheService, appStore))
	// Download interstitial page
	e.GET("/get/{name}", pages.DownloadInterstitialHandler(app, registry, cacheService, appStore))
	// External link interstitial (encrypted token, no raw URL exposed)
	e.GET("/out/{token}", pages.ExternalLinkInterstitialHandler(app, registry, cacheService, outSecret, appStore))
	e.GET("/schematics/{name}/edit", pages.EditSchematicHandler(app, searchService, cacheService, registry, appStore))
	// Search autocomplete
	e.GET("/api/search/suggest", pages.SearchSuggestHandler(searchService))
	e.GET("/search/{term}/page/{page}", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.GET("/search/{term}", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.POST("/search/{term}", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.GET("/search/page/{page}", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.GET("/search", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.GET("/search/", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.POST("/search/", pages.SearchHandler(app, searchService, cacheService, registry, appStore))
	e.POST("/search", pages.SearchPostHandler(app, cacheService, registry, appStore))
	// User
	e.GET("/author/{username}", pages.ProfileHandler(app, cacheService, registry, appStore))
	e.GET("/profile", pages.ProfileHandler(app, cacheService, registry, appStore))
	// Fallback
	e.GET("/{any}", pages.FourOhFourHandler(app, registry, appStore))

}

func legacyCategoryCompat() func(e *core.RequestEvent) error {
	// to /search/?category=apple
	urlMatches := []string{
		"/schematics/category/",
		"/schematic_categories/",
	}
	return func(e *core.RequestEvent) error {
		path := e.Request.URL.Path
		for _, match := range urlMatches {
			if strings.HasPrefix(path, match) {
				return e.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/?category=%s", strings.ReplaceAll(strings.Replace(path, match, "", 1), "/", "")))
			}
		}
		return e.Next() // proceed with the request chain
	}
}

// cookieAuth authenticates requests using PostgreSQL sessions.
// It populates the request context with the session and bridges to PocketBase's
// e.Auth by looking up the PB user record, so PB hooks still work.
func cookieAuth(app *pocketbase.PocketBase, sessStore *session.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth != nil {
			return e.Next()
		}

		cookie, err := e.Request.Cookie(auth.CookieName)
		if err != nil {
			return e.Next()
		}

		token := strings.TrimSpace(cookie.Value)
		if token == "" {
			return e.Next()
		}

		// Validate session in PostgreSQL
		sess, err := sessStore.Validate(e.Request.Context(), token)
		if err != nil || sess == nil {
			return e.Next()
		}

		// Put session in request context for handlers
		ctx := session.ContextWithSession(e.Request.Context(), sess)
		e.Request = e.Request.WithContext(ctx)

		// Bridge: populate e.Auth from PB so PB API hooks (comments, ratings, schematics) still work
		if sess.UserID != "" {
			record, err := app.FindRecordById("users", sess.UserID)
			if err == nil && record != nil {
				e.Auth = record
			}
		}

		return e.Next()
	}
}

func legacyFileCompat() func(e *core.RequestEvent) error {
	fileMatches := map[string]string{
		"/wp-sitemap.xml":    "/sitemaps/sitemap.xml",
		"/upload-schematic":  "/upload",
		"/upload-schematics": "/upload",
	}
	return func(e *core.RequestEvent) error {
		path := e.Request.URL.Path
		for match, newRoute := range fileMatches {
			if path == match || strings.HasPrefix(path, match) {
				return e.Redirect(http.StatusMovedPermanently, newRoute)
			}
		}
		return e.Next() // proceed with the request chain
	}
}

func legacyTagCompat() func(e *core.RequestEvent) error {
	// to /search/?tag=apple
	urlMatches := []string{
		"/schematics/tag/",
	}
	queryMatches := []string{
		"schematic_tags",
	}
	return func(e *core.RequestEvent) error {
		path := e.Request.URL.Path
		for _, match := range urlMatches {
			if strings.HasPrefix(path, match) {
				return e.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/?tag=%s", strings.ReplaceAll(strings.Replace(path, match, "", 1), "/", "")))
			}
		}
		query := e.Request.URL.Query()
		for _, match := range queryMatches {
			if query.Has(match) && query.Get(match) != "" {
				return e.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/?tag=%s", query.Get(match)))
			}
		}
		return e.Next() // proceed with the request chain
	}
}

func legacySearchCompat() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// ?s=test&id=95&post_type=schematics
		// test is the term above
		path := e.Request.URL.Path
		query := e.Request.URL.Query()
		fmt.Sprintln("should redirect", e.Request.URL.Path, e.Request.URL.Query())
		if (path == "" || path == "/") && query.Has("s") && query.Get("s") != "" {
			searchSlug := slug.Make(query.Get("s"))
			return e.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/%s", searchSlug))
		}

		return e.Next() // proceed with the request chain
	}
}

func getContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Read body: %v", err)
	}

	return string(data), nil
}

// deriveOutSecret returns a stable HMAC key for signing /out redirect URLs.
// It uses the OUT_SECRET env var if set, otherwise derives one from the
// PocketBase data directory path so it is stable across restarts.
func deriveOutSecret(app *pocketbase.PocketBase) string {
	if s := os.Getenv("OUT_SECRET"); s != "" {
		return s
	}
	h := sha256.Sum256([]byte("createmod-out-url-sign:" + app.DataDir()))
	return hex.EncodeToString(h[:])
}
