package router

import (
	"context"
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/i18n"
	"createmod/internal/jobs"
	"createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/outurl"
	"createmod/internal/pages"
	"createmod/internal/promotion"
	"createmod/internal/ratelimit"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"createmod/internal/webhook"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	html "html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gosimple/slug"
	"github.com/riverqueue/river"
)

// Adapt converts a server.RequestEvent handler into an http.HandlerFunc.
// It creates a RequestEvent from the standard HTTP primitives and handles
// error responses via server.APIError.
func Adapt(h func(e *server.RequestEvent) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e := server.NewRequestEvent(w, r)
		if err := h(e); err != nil {
			if apiErr, ok := err.(*server.APIError); ok {
				http.Error(w, apiErr.Message, apiErr.Status)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

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

// RegisterParams holds all dependencies needed for route registration.
type RegisterParams struct {
	SearchService      *search.Service
	CacheService       *cache.Service
	RateLimiter        ratelimit.Limiter
	DiscordService     *discord.Service
	ModerationService  *moderation.Service
	TranslationService *translation.Service
	ModMetaService     interface{}
	AppStore           *store.Store
	SessionStore       *session.Store
	StorageService     *storage.Service
	DiscordOAuth       *auth.OAuthProvider
	GithubOAuth        *auth.OAuthProvider
	MailService        *mailer.Service
	JobWorker          *jobs.Worker
	MaintenanceMode    *atomic.Bool // runtime-togglable maintenance flag
}

func Register(p RegisterParams) chi.Router {
	promotionService := promotion.New()
	registry := server.NewRegistry()

	assetVer := computeAssetVersion()

	// Derive a stable HMAC key for signing outgoing redirect URLs.
	outSecret := deriveOutSecret()

	funcMap := html.FuncMap{
		"ToLower":        strings.ToLower,
		"mod":            func(i, j int) bool { return i%j == 0 },
		"urlPathEscape":  url.PathEscape,
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
		"externalDomain": pages.ExternalDomain,
		"LangFlag": func(code string) html.HTML {
			cc := "gb"
			switch code {
			case "en":
				cc = "gb"
			case "pt-BR":
				cc = "br"
			case "pt-PT":
				cc = "pt"
			case "es":
				cc = "es"
			case "de":
				cc = "de"
			case "pl":
				cc = "pl"
			case "ru":
				cc = "ru"
			case "zh-Hans":
				cc = "cn"
			case "fr":
				cc = "fr"
			}
			return html.HTML(`<span class="fi fi-` + cc + `"></span>`)
		},
	}

	registry.AddFuncs(funcMap)

	r := chi.NewRouter()

	// Maintenance mode — toggled at runtime via the shared atomic flag.
	// Also activatable via MAINTENANCE_MODE=true env var at startup.
	// The /api/health endpoint is excluded so load balancers can still probe.
	maintenanceFlag := p.MaintenanceMode
	if maintenanceFlag == nil {
		maintenanceFlag = &atomic.Bool{}
	}
	if os.Getenv("MAINTENANCE_MODE") == "true" {
		maintenanceFlag.Store(true)
	}
	r.Use(headMethodSupport)
	r.Use(requestLogger)
	r.Use(securityHeaders)
	r.Use(maintenanceModeMiddleware(maintenanceFlag))
	r.Use(legacyFileCompat)
	r.Use(legacySearchCompat)
	r.Use(legacyCategoryCompat)
	r.Use(legacyTagCompat)
	r.Use(cookieAuth(p.SessionStore))
	r.Use(csrfOriginCheck)

	// Health check endpoint — excluded from maintenance mode via the middleware itself.
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Custom file serving (replaces PB's /api/files/ handler with image resizing support)
	r.Get("/api/files/{collection}/{recordID}/{filename}", Adapt(pages.FileServingHandler(p.StorageService)))

	// Frontend routes
	// Serve sitemaps from S3 storage
	r.Get("/sitemaps/{file}", func(w http.ResponseWriter, req *http.Request) {
		filename := chi.URLParam(req, "file")
		if filename == "" || p.StorageService == nil {
			http.NotFound(w, req)
			return
		}
		reader, err := p.StorageService.DownloadRaw(req.Context(), "_sitemaps/"+filename)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		defer reader.Close()
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		io.Copy(w, reader)
	})
	// Static assets with long cache (files use ?v=hash cache-busting)
	staticFS := http.StripPrefix("/assets/x/", http.FileServer(http.Dir("./template/static")))
	r.Handle("/assets/x/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		staticFS.ServeHTTP(w, req)
	}))
	r.Get("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# Site-specific rules (Cloudflare prepends its managed block above)\nUser-agent: *\nDisallow: /_/\nDisallow: /get/\nDisallow: /out/\n\nSitemap: https://createmod.com/sitemaps/sitemap.xml\n"))
	})
	r.Get("/feed.xml", Adapt(pages.RSSFeedHandler(p.AppStore, p.CacheService)))
	r.Get("/ads.txt", func(w http.ResponseWriter, req *http.Request) {
		s, ok := p.CacheService.GetString("ads.txt")
		if ok {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(s))
			return
		}
		s, err := getContent("https://api.nitropay.com/v1/ads-2143.txt")
		if err != nil || s == "" {
			http.Error(w, "Could not determine content", 500)
			return
		}
		p.CacheService.SetString("ads.txt", s)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(s))
	})

	// Index
	r.Get("/", Adapt(pages.IndexHandler(p.CacheService, registry, p.AppStore)))
	r.Get("/upload", Adapt(pages.UploadHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/upload/nbt", Adapt(pages.UploadNBTHandler(registry, p.CacheService, p.AppStore, p.StorageService)))
	// Private preview URL for temporary uploads
	r.Get("/u/{token}", Adapt(pages.UploadPreviewHandler(registry, p.CacheService, p.AppStore)))
	// Download endpoint for temporary uploads
	r.Get("/u/{token}/download", Adapt(pages.UploadDownloadHandler(p.AppStore, p.StorageService)))
	r.Post("/u/{token}/claim", Adapt(pages.UploadClaimHandler(p.AppStore)))
	r.Post("/u/{token}/add-file", Adapt(pages.UploadAddFileHandler(p.AppStore, p.StorageService)))
	r.Delete("/u/{token}/files/{fileId}", Adapt(pages.UploadDeleteFileHandler(p.AppStore, p.StorageService)))
	r.Get("/u/{token}/files/{fileId}/download", Adapt(pages.UploadFileDownloadHandler(p.AppStore, p.StorageService)))
	// Build the moderation enqueuer callback that closes over the job worker.
	var enqueueModeration pages.ModerationEnqueuer
	if p.JobWorker != nil {
		jw := p.JobWorker
		enqueueModeration = func(ctx context.Context, args pages.ModerationJobArgs) error {
			return jw.Insert(ctx, jobs.ModerationArgs{
				SchematicID: args.SchematicID,
				Title:       args.Title,
				Description: args.Description,
				ImageURL:    args.ImageURL,
				Slug:        args.Slug,
			}, &river.InsertOpts{Queue: "ai"})
		}
	}
	r.Post("/u/{token}/make-public", Adapt(pages.UploadMakePublicHandler(registry, p.CacheService, p.AppStore, p.StorageService, p.ModerationService, p.MailService, enqueueModeration)))
	// Publish form for temporary uploads (requires auth)
	r.Get("/u/{token}/publish", Adapt(pages.UploadPublishHandler(registry, p.CacheService, p.AppStore)))
	// Upload moderation pending confirmation page
	r.Get("/upload/pending", Adapt(pages.UploadPendingHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/contact", Adapt(pages.ContactHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/api/contact", Adapt(pages.ContactSubmitHandler(p.AppStore, p.MailService)))
	// Comments and ratings API (replaces PB REST endpoints)
	r.Post("/api/comments", Adapt(pages.CommentCreateHandler(p.AppStore, p.MailService)))
	r.Delete("/api/comments/{id}", Adapt(pages.CommentDeleteHandler(p.AppStore)))
	r.Post("/api/ratings", Adapt(pages.RatingUpsertHandler(p.AppStore)))
	// User profile API (replaces PB REST endpoints)
	r.Patch("/api/users/{id}", Adapt(pages.UserUpdateHandler(p.AppStore)))
	r.Delete("/api/users/{id}", Adapt(pages.UserDeleteHandler(p.AppStore, p.CacheService, p.SessionStore)))
	// Schematic edit/delete API (replaces PB REST endpoints)
	r.Post("/schematics/{id}/update", Adapt(pages.SchematicUpdateHandler(p.SearchService, p.CacheService, p.StorageService, p.AppStore)))
	r.Delete("/schematics/{id}", Adapt(pages.SchematicDeleteHandler(p.CacheService, p.AppStore)))
	r.Get("/blacklist-request", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, pages.LangRedirectURLFromRequest(req, "/settings/blacklist"), http.StatusMovedPermanently)
	})
	// Redirect legacy single guide page to the guides listing
	r.Get("/guide", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, pages.LangRedirectURLFromRequest(req, "/guides"), http.StatusMovedPermanently)
	})
	r.Get("/rules", Adapt(pages.RulesHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/explore", Adapt(pages.ExploreHandler(p.CacheService, registry, p.AppStore)))
	r.Get("/api/explore/images", Adapt(pages.ExploreAPIHandler(p.CacheService, p.AppStore)))
	r.Get("/terms-of-service", Adapt(pages.TermsOfServiceHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/privacy-policy", Adapt(pages.PrivacyPolicyHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/settings", Adapt(pages.UserSettingsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/settings/password", Adapt(pages.UserPasswordHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/settings/password", Adapt(pages.UserPasswordPostHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/settings/points", Adapt(pages.UserPointsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/settings/gamification", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, pages.LangRedirectURLFromRequest(req, "/settings/points"), http.StatusMovedPermanently)
	})
	r.Get("/settings/api-keys", Adapt(pages.UserAPIKeysHandler(registry, p.CacheService, p.AppStore)))
	webhookSecret := webhook.Secret()
	r.Get("/settings/webhooks", Adapt(pages.UserWebhooksHandler(registry, p.CacheService, p.AppStore, webhookSecret)))
	r.Post("/settings/webhooks", Adapt(pages.UserWebhookSaveHandler(p.CacheService, p.AppStore, webhookSecret)))
	r.Post("/settings/webhooks/delete", Adapt(pages.UserWebhookDeleteHandler(p.AppStore)))
	r.Get("/settings/statistics", Adapt(pages.UserStatsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/settings/blacklist", Adapt(pages.BlacklistRequestHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/settings/blacklist/upload", Adapt(pages.BlacklistUploadHandler(p.AppStore)))
	r.Delete("/settings/blacklist/{id}", Adapt(pages.BlacklistDeleteHandler(p.AppStore)))
	// API Docs
	r.Get("/api", Adapt(pages.APIDocsHandler(registry, p.CacheService, p.AppStore)))
	// Public JSON API (beta)
	r.Get("/api/schematics", Adapt(pages.APISchematicsListHandler(p.SearchService, p.RateLimiter, p.CacheService, p.AppStore)))
	r.Get("/api/schematics/{name}", Adapt(pages.APISchematicDetailHandler(p.RateLimiter, p.CacheService, p.AppStore)))
	r.Post("/api/schematics/upload", Adapt(pages.APIUploadHandler(p.RateLimiter, p.CacheService, p.AppStore, p.StorageService)))
	r.Post("/api/schematics/upload-anonymous", Adapt(pages.APIUploadAnonymousHandler(p.RateLimiter, p.CacheService, p.AppStore, p.StorageService)))
	// Reports
	r.Post("/reports", Adapt(pages.ReportSubmitHandler(p.MailService, p.AppStore)))
	// Admin
	r.Get("/admin", Adapt(pages.AdminDashboardHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/admin/reports", Adapt(pages.AdminReportsHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/admin/reports/{id}/resolve", Adapt(pages.AdminReportResolveHandler(p.AppStore, p.MailService)))
	r.Post("/admin/reports/{id}/delete-target", Adapt(pages.AdminReportDeleteTargetHandler(p.AppStore)))
	r.Post("/admin/reports/{id}/ignore", Adapt(pages.AdminReportIgnoreHandler(p.AppStore)))
	r.Get("/admin/schematics", Adapt(pages.AdminSchematicsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/admin/schematics/{id}", Adapt(pages.AdminSchematicEditHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/admin/schematics/{id}", Adapt(pages.AdminSchematicUpdateHandler(p.SearchService, p.CacheService, p.AppStore, p.MailService)))
	r.Post("/admin/schematics/{id}/delete", Adapt(pages.AdminSchematicDeleteHandler(p.CacheService, p.AppStore)))
	r.Get("/admin/tags", Adapt(pages.AdminTagsHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/admin/tags/{id}/approve", Adapt(pages.AdminTagApproveHandler(p.CacheService, p.AppStore)))
	r.Post("/admin/tags/{id}/reject", Adapt(pages.AdminTagRejectHandler(p.CacheService, p.AppStore)))
	r.Get("/admin/mods", Adapt(pages.AdminModsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/admin/mods/{namespace}", Adapt(pages.AdminModEditHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/admin/mods/{namespace}", Adapt(pages.AdminModUpdateHandler(p.AppStore)))
	// Auth — rate-limited to 10 POST requests per IP per minute
	authRateLimit := rateLimitMiddlewareNew(p.RateLimiter, 10, time.Minute)
	r.Get("/login", Adapt(pages.LoginHandler(registry, p.AppStore)))
	r.With(authRateLimit).Post("/login", Adapt(pages.LoginPostHandler(p.AppStore, p.SessionStore)))
	r.Get("/register", Adapt(pages.RegisterHandler(registry, p.AppStore)))
	r.With(authRateLimit).Post("/register", Adapt(pages.RegisterPostHandler(p.AppStore, p.SessionStore)))
	r.Get("/reset-password", Adapt(pages.PasswordResetHandler(registry, p.AppStore)))
	r.With(authRateLimit).Post("/reset-password", Adapt(pages.PasswordResetPostHandler(p.MailService, registry, p.AppStore)))
	r.Get("/reset-password/{token}", Adapt(pages.PasswordResetConfirmHandler(registry, p.AppStore)))
	r.With(authRateLimit).Post("/reset-password/{token}", Adapt(pages.PasswordResetConfirmPostHandler(registry, p.AppStore, p.SessionStore)))
	// OAuth routes
	r.Get("/auth/discord", Adapt(pages.OAuthRedirectHandler(p.DiscordOAuth)))
	r.Get("/auth/discord/callback", Adapt(pages.OAuthCallbackHandler(p.DiscordOAuth, p.AppStore, p.SessionStore)))
	r.Get("/auth/github", Adapt(pages.OAuthRedirectHandler(p.GithubOAuth)))
	r.Get("/auth/github/callback", Adapt(pages.OAuthCallbackHandler(p.GithubOAuth, p.AppStore, p.SessionStore)))
	r.Get("/logout", func(w http.ResponseWriter, req *http.Request) {
		secure := req.TLS != nil || strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https")

		// Delete PostgreSQL session
		if cookie, err := req.Cookie(auth.CookieName); err == nil {
			_ = p.SessionStore.Delete(req.Context(), cookie.Value)
		}

		auth.ClearAuthCookie(w, secure)
		if req.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Redirect", pages.LangRedirectURLFromRequest(req, "/"))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, req, pages.LangRedirectURLFromRequest(req, "/"), http.StatusFound)
	})
	// News
	r.Get("/news", Adapt(pages.NewsHandler(registry, p.CacheService, p.AppStore)))
	r.Get("/news/{slug}", Adapt(pages.NewsPostHandler(registry, p.CacheService, p.AppStore)))
	// Users listing
	r.Get("/users", Adapt(pages.UsersHandler(registry, p.CacheService, p.AppStore)))
	// Videos listing
	r.Get("/videos", Adapt(pages.VideosHandler(registry, p.CacheService, p.AppStore)))
	// Guides
	r.Get("/guides", Adapt(pages.GuidesHandler(registry, p.CacheService, outSecret, p.AppStore)))
	r.Get("/guides/new", Adapt(pages.GuidesNewHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/guides", Adapt(pages.GuidesCreateHandler(p.CacheService, p.AppStore, p.StorageService, p.ModerationService)))
	r.Get("/guides/{id}", Adapt(pages.GuidesShowHandler(registry, p.CacheService, p.TranslationService, p.AppStore)))
	r.Get("/guides/{id}/edit", Adapt(pages.GuidesEditHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/guides/{id}", Adapt(pages.GuidesUpdateHandler(p.CacheService, p.AppStore, p.StorageService, p.ModerationService)))
	r.Post("/guides/{id}/delete", Adapt(pages.GuidesDeleteHandler(p.AppStore)))
	// Mods
	r.Get("/mods", Adapt(pages.ModsHandler(p.CacheService, registry, p.ModMetaService, p.AppStore)))
	r.Get("/mods/{slug}", Adapt(pages.ModDetailHandler(p.CacheService, registry, p.ModMetaService, p.AppStore)))
	// Collections
	r.Get("/collections", Adapt(pages.CollectionsHandler(registry, p.CacheService, p.AppStore, p.StorageService)))
	r.Get("/collections/new", Adapt(pages.CollectionsNewHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/api/images/upload", Adapt(pages.ImageUploadHandler(p.StorageService)))
	r.Post("/collections", Adapt(pages.CollectionsCreateHandler(registry, p.CacheService, p.AppStore, p.StorageService, p.ModerationService)))
	r.Get("/collections/{slug}", Adapt(pages.CollectionsShowHandler(registry, p.CacheService, p.TranslationService, p.AppStore, p.StorageService)))
	r.Get("/collections/{slug}/edit", Adapt(pages.CollectionsEditHandler(registry, p.CacheService, p.AppStore)))
	r.Post("/collections/{slug}", Adapt(pages.CollectionsUpdateHandler(registry, p.CacheService, p.ModerationService, p.AppStore, p.StorageService, p.MailService)))
	r.Post("/collections/{slug}/delete", Adapt(pages.CollectionsDeleteHandler(p.AppStore)))
	r.Post("/collections/{slug}/reorder", Adapt(pages.CollectionsReorderHandler(p.AppStore, p.StorageService)))
	// API keys (user settings)
	r.Post("/settings/api-keys/new", Adapt(pages.APIKeyCreateHandler(p.CacheService, p.AppStore)))
	r.Post("/settings/api-keys/{id}/revoke", Adapt(pages.APIKeyRevokeHandler(p.AppStore)))
	r.Post("/api/keys/generate", Adapt(pages.APIKeyCreateJSONHandler(p.AppStore)))
	// Language setter
	r.Get("/lang", Adapt(pages.SetLanguageHandler()))
	// Schematics
	r.Get("/schematics", Adapt(pages.SchematicsHandler(p.CacheService, registry, p.AppStore)))
	r.Get("/schematics/{name}", Adapt(pages.SchematicHandler(p.SearchService, p.CacheService, registry, promotionService, p.DiscordService, p.TranslationService, p.AppStore, webhookSecret)))
	// Partial comments endpoint for HTMX refresh
	r.Get("/schematics/{name}/comments", Adapt(pages.SchematicCommentsHandler(p.SearchService, p.CacheService, registry, p.DiscordService, p.AppStore)))
	// Add to collection
	r.Post("/schematics/{name}/add-to-collection", Adapt(pages.SchematicAddToCollectionHandler(p.AppStore)))
	// Download endpoint to track download metrics separately
	r.Get("/download/{name}", Adapt(pages.DownloadHandler(p.RateLimiter, p.CacheService, p.AppStore)))
	// Download interstitial page
	r.Get("/get/{name}", Adapt(pages.DownloadInterstitialHandler(registry, p.CacheService, p.AppStore)))
	// API endpoint to fetch download URL after interstitial delay
	r.Get("/api/download-url/{id}", Adapt(pages.DownloadURLHandler(p.AppStore)))
	// External link interstitial (encrypted token, no raw URL exposed)
	r.Get("/out/{token}", Adapt(pages.ExternalLinkInterstitialHandler(registry, p.CacheService, outSecret, p.AppStore)))
	r.Get("/schematics/{name}/edit", Adapt(pages.EditSchematicHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	// Search autocomplete
	r.Get("/api/search/suggest", Adapt(pages.SearchSuggestHandler(p.SearchService)))
	r.Get("/search/{term}/page/{page}", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Get("/search/{term}", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Post("/search/{term}", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Get("/search/page/{page}", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Get("/search", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Get("/search/", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Post("/search/", Adapt(pages.SearchHandler(p.SearchService, p.CacheService, registry, p.AppStore)))
	r.Post("/search", Adapt(pages.SearchPostHandler(p.CacheService, registry, p.AppStore)))
	// User
	r.Get("/author/{username}", Adapt(pages.ProfileHandler(p.CacheService, registry, p.AppStore)))
	r.Get("/profile", Adapt(pages.ProfileHandler(p.CacheService, registry, p.AppStore)))
	// Fallback
	r.Get("/*", Adapt(pages.FourOhFourHandler(registry, p.AppStore)))

	return r
}

func legacyCategoryCompat(next http.Handler) http.Handler {
	urlMatches := []string{
		"/schematics/category/",
		"/schematic_categories/",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for _, match := range urlMatches {
			if strings.HasPrefix(path, match) {
				http.Redirect(w, r, fmt.Sprintf("/search/?category=%s", strings.ReplaceAll(strings.Replace(path, match, "", 1), "/", "")), http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// cookieAuth authenticates requests using PostgreSQL sessions.
func cookieAuth(sessStore *session.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			token := strings.TrimSpace(cookie.Value)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate session in PostgreSQL
			sess, err := sessStore.Validate(r.Context(), token)
			if err != nil || sess == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Put session in request context for handlers
			ctx := session.ContextWithSession(r.Context(), sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func legacyFileCompat(next http.Handler) http.Handler {
	fileMatches := map[string]string{
		"/wp-sitemap.xml":    "/sitemaps/sitemap.xml",
		"/upload-schematic":  "/upload",
		"/upload-schematics": "/upload",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for match, newRoute := range fileMatches {
			if path == match || strings.HasPrefix(path, match) {
				http.Redirect(w, r, newRoute, http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func legacyTagCompat(next http.Handler) http.Handler {
	urlMatches := []string{
		"/schematics/tag/",
	}
	queryMatches := []string{
		"schematic_tags",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for _, match := range urlMatches {
			if strings.HasPrefix(path, match) {
				http.Redirect(w, r, fmt.Sprintf("/search/?tag=%s", strings.ReplaceAll(strings.Replace(path, match, "", 1), "/", "")), http.StatusMovedPermanently)
				return
			}
		}
		query := r.URL.Query()
		for _, match := range queryMatches {
			if query.Has(match) && query.Get(match) != "" {
				http.Redirect(w, r, fmt.Sprintf("/search/?tag=%s", query.Get(match)), http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func legacySearchCompat(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		query := r.URL.Query()
		if (path == "" || path == "/") && query.Has("s") && query.Get("s") != "" {
			searchSlug := slug.Make(query.Get("s"))
			http.Redirect(w, r, fmt.Sprintf("/search/%s", searchSlug), http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Read body: %v", err)
	}

	return string(data), nil
}

// responseRecorder wraps http.ResponseWriter to capture the status code.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

// requestLogger logs each HTTP request with method, path, status, and duration.
// headMethodSupport converts HEAD requests to GET for routing purposes.
// Chi does not automatically serve HEAD for registered GET routes, so
// without this middleware HEAD requests return 405 Method Not Allowed.
// The http.ResponseWriter automatically suppresses the response body
// for HEAD requests per the net/http specification.
func headMethodSupport(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			r.Method = http.MethodGet
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rr.status,
			"duration", time.Since(start).Round(time.Millisecond).String(),
			"ip", r.RemoteAddr,
		)
	})
}

// maintenanceModeMiddleware returns a middleware that checks the given flag on
// every request. When the flag is true it serves a 503 page; when false it
// passes through to the next handler. The /api/health endpoint is excluded
// (registered before middleware) so load balancers can still probe.
func maintenanceModeMiddleware(flag *atomic.Bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flag.Load() && r.URL.Path != "/api/health" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(maintenancePage))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

const maintenancePage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CreateMod — Coming Back Soon</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
       background:#1a1a2e;color:#e0e0e0;display:flex;align-items:center;
       justify-content:center;min-height:100vh;text-align:center;padding:2rem}
  .container{max-width:520px}
  h1{font-size:2rem;margin-bottom:1rem;color:#fff}
  p{font-size:1.1rem;line-height:1.6;color:#b0b0c0;margin-bottom:0.5rem}
  .status{font-size:0.9rem;color:#888;margin-top:1.5rem}
  .gear{font-size:4rem;margin-bottom:1.5rem;display:inline-block;animation:spin 4s linear infinite}
  @keyframes spin{from{transform:rotate(0deg)}to{transform:rotate(360deg)}}
</style>
</head>
<body>
<div class="container">
  <div class="gear">&#9881;</div>
  <h1>Coming Back Soon</h1>
  <p>CreateMod.com is temporarily unavailable while we perform maintenance.</p>
  <p>We'll be back shortly. Thank you for your patience!</p>
  <p class="status" id="status">Checking again in 30s&hellip;</p>
</div>
<script>
(function(){
  var seconds = 30;
  var el = document.getElementById("status");
  var timer = setInterval(function(){
    seconds--;
    if(seconds > 0){
      el.textContent = "Checking again in " + seconds + "s\u2026";
      return;
    }
    seconds = 30;
    el.textContent = "Checking\u2026";
    fetch("/api/health").then(function(r){
      if(r.ok) location.reload();
      else el.textContent = "Still down. Checking again in 30s\u2026";
    }).catch(function(){
      el.textContent = "Still down. Checking again in 30s\u2026";
    });
  }, 1000);
})();
</script>
</body>
</html>`

// csrfOriginCheck is a middleware that validates the Origin or Referer header
// on state-changing requests (POST, PUT, PATCH, DELETE). If neither header is
// present or they point to a different host, the request is rejected.
// Combined with SameSite=Lax session cookies, this provides robust CSRF protection.
func csrfOriginCheck(next http.Handler) http.Handler {
	safeMethods := map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if safeMethods[r.Method] {
			next.ServeHTTP(w, r)
			return
		}

		host := r.Host
		if host == "" {
			host = r.URL.Host
		}

		// Check Origin header first (most reliable)
		if origin := r.Header.Get("Origin"); origin != "" {
			// Allow CORS-approved origins on API routes
			if strings.HasPrefix(r.URL.Path, "/api/") && corsAllowedOrigins[origin] {
				next.ServeHTTP(w, r)
				return
			}
			// Parse to extract host
			// Origin is usually "scheme://host" or "scheme://host:port"
			originHost := origin
			if idx := strings.Index(origin, "://"); idx >= 0 {
				originHost = origin[idx+3:]
			}
			// Remove trailing path if any
			if idx := strings.Index(originHost, "/"); idx >= 0 {
				originHost = originHost[:idx]
			}
			if strings.EqualFold(originHost, host) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden: origin mismatch", http.StatusForbidden)
			return
		}

		// Fall back to Referer header
		if referer := r.Header.Get("Referer"); referer != "" {
			refererHost := referer
			if idx := strings.Index(referer, "://"); idx >= 0 {
				refererHost = referer[idx+3:]
			}
			if idx := strings.Index(refererHost, "/"); idx >= 0 {
				refererHost = refererHost[:idx]
			}
			if strings.EqualFold(refererHost, host) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden: referer mismatch", http.StatusForbidden)
			return
		}

		// Neither Origin nor Referer present — allow for non-browser clients (API users).
		// Browser-based requests will always include at least one of these headers.
		next.ServeHTTP(w, r)
	})
}

// corsAllowedOrigins lists external origins permitted to make cross-origin
// requests to /api/ endpoints.
var corsAllowedOrigins = map[string]bool{
	"https://bloxelizer.com":     true,
	"https://www.bloxelizer.com": true,
}

// cspHeader is the Content-Security-Policy value applied to every response.
// It is built once at init time to avoid repeated string concatenation.
//
// 'unsafe-inline' is required for script-src and style-src because the site
// uses inline <script> blocks (theme init, sidebar state, NitroPay queue,
// Google Analytics, HTMX handlers, page-specific init) and inline style
// attributes throughout templates. Removing 'unsafe-inline' would require
// refactoring all inline code into external files with nonces or hashes.
//
// Third-party scripts dynamically load resources from many domains, so
// style-src, img-src, font-src, frame-src, and connect-src use https: broadly.
var cspHeader = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self' 'unsafe-inline' 'unsafe-eval' https:",
	"style-src 'self' 'unsafe-inline' https:",
	"img-src 'self' data: blob: https:",
	"font-src 'self' https:",
	"connect-src 'self' https:",
	"frame-src https:",
	"media-src 'self' https:",
	"object-src 'none'",
	"base-uri 'self'",
	"form-action 'self'",
	"frame-ancestors 'self'",
}, "; ")

// securityHeaders sets standard security response headers on every request.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", cspHeader)

		// CORS for allowed external origins on API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			origin := r.Header.Get("Origin")
			if corsAllowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
				w.Header().Set("Access-Control-Max-Age", "86400")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddlewareNew returns a middleware that limits requests per IP
// using the shared ratelimit.Limiter (Redis or in-memory).
func rateLimitMiddlewareNew(rl ratelimit.Limiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			// Use X-Forwarded-For if behind a proxy
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = strings.SplitN(forwarded, ",", 2)[0]
				ip = strings.TrimSpace(ip)
			}
			key := "auth:" + ip
			if ok, _ := rl.Allow(r.Context(), key, limit, window); !ok {
				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// deriveOutSecret returns a stable HMAC key for signing /out redirect URLs.
func deriveOutSecret() string {
	if s := os.Getenv("OUT_SECRET"); s != "" {
		return s
	}
	slog.Warn("OUT_SECRET environment variable is not set; using insecure default — set OUT_SECRET in production")
	h := sha256.Sum256([]byte("createmod-out-url-sign:default"))
	return hex.EncodeToString(h[:])
}
