package router

import (
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/pages"
	"createmod/internal/search"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/template"
	html "html/template"
	"net/http"
	"os"
	"strings"
)

func Register(app *pocketbase.PocketBase, e *router.Router[*core.RequestEvent], searchService *search.Service, cacheService *cache.Service) {
	registry := template.NewRegistry()

	funcMap := html.FuncMap{
		"ToLower": strings.ToLower,
	}

	registry.AddFuncs(funcMap)

	e.BindFunc(legacyFileCompat())
	e.BindFunc(legacySearchCompat())
	e.BindFunc(legacyCategoryCompat())
	e.BindFunc(legacyTagCompat())
	e.BindFunc(cookieAuth(app))
	// Frontend routes
	e.GET("/sitemaps/{path...}", apis.Static(os.DirFS("./template/dist/sitemaps"), false))
	e.GET("/libs/{path...}", apis.Static(os.DirFS("./template/dist/libs"), false))
	e.GET("/assets/{path...}", apis.Static(os.DirFS("./template/dist/assets"), false))
	e.GET("/assets/x/{path...}", apis.Static(os.DirFS("./template/static"), false))
	// Index
	e.GET("/", pages.IndexHandler(app, cacheService, registry))
	// Removed the about page, not relevant anymore
	e.GET("/upload", pages.UploadHandler(app, registry))
	e.GET("/contact", pages.ContactHandler(app, registry))
	e.GET("/guide", pages.GuideHandler(app, registry))
	e.GET("/rules", pages.RulesHandler(app, registry))
	e.GET("/terms-of-service", pages.TermsOfServiceHandler(app, registry))
	e.GET("/privacy-policy", pages.PrivacyPolicyHandler(app, registry))
	e.GET("/settings", pages.UserSettingsHandler(app, registry))
	// Auth
	e.GET("/login", pages.LoginHandler(app, registry))
	e.GET("/register", pages.RegisterHandler(app, registry))
	e.GET("/reset-password", pages.PasswordResetHandler(app, registry))
	// News
	e.GET("/news", pages.NewsHandler(app, registry))
	e.GET("/news/:slug", pages.NewsPostHandler(app, registry))
	// Schematics
	e.GET("/schematics", pages.SchematicsHandler(app, cacheService, registry))
	e.GET("/schematics/{name}", pages.SchematicHandler(app, searchService, cacheService, registry))
	e.GET("/search/{term}", pages.SearchHandler(app, searchService, cacheService, registry))
	e.POST("/search/{term}", pages.SearchHandler(app, searchService, cacheService, registry))
	e.GET("/search", pages.SearchHandler(app, searchService, cacheService, registry))
	e.GET("/search/", pages.SearchHandler(app, searchService, cacheService, registry))
	e.POST("/search/", pages.SearchHandler(app, searchService, cacheService, registry))
	e.POST("/search", pages.SearchPostHandler(app, cacheService, registry))
	// User
	e.GET("/author/{username}", pages.ProfileHandler(app, cacheService, registry))
	e.GET("/profile", pages.ProfileHandler(app, cacheService, registry))
	// Fallback
	e.GET("/{any}", pages.FourOhFourHandler(app, registry))

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

// cookieAuth was added so that requests can be authenticated on the backend when HTML templates are rendered
func cookieAuth(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth != nil {
			return e.Next()
		}

		cookie, err := e.Request.Cookie(auth.CookieName)
		if err != nil {
			return e.Next()
		}

		token := strings.TrimSpace(cookie.Value)

		record, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
		if err == nil && record != nil {
			e.Auth = record
		}
		return e.Next()
	}
}

func legacyFileCompat() func(e *core.RequestEvent) error {
	fileMatches := map[string]string{
		"/wp-sitemap.xml": "/sitemaps/sitemap.xml",
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
