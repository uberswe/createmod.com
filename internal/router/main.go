package router

import (
	"createmod/internal/auth"
	"createmod/internal/pages"
	"createmod/internal/search"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tokens"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/spf13/cast"
	"html/template"
	"net/http"
	"os"
	"strings"
)

func Register(app *pocketbase.PocketBase, e *echo.Echo, searchService *search.Service) {
	// HTML Template Renderer
	templateBuilder := template.New("")

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	glob, err := templateBuilder.Funcs(funcMap).ParseGlob("template/dist/*/*.html")
	if err != nil {
		return
	}
	t := &Template{
		templates: template.Must(glob.ParseGlob("template/dist/*.html")),
	}

	e.Renderer = t
	e.Use(legacyFileCompat)
	e.Use(legacySearchCompat)
	e.Use(legacyTagCompat)
	e.Use(cookieAuth(app))
	// Frontend routes
	e.GET("/sitemaps/*", apis.StaticDirectoryHandler(os.DirFS("./template/sitemaps"), false))
	e.Static("/libs/", "./template/dist/libs")
	e.GET("/assets/*", apis.StaticDirectoryHandler(os.DirFS("./template/dist/assets"), false))
	// Index
	e.GET("/", pages.IndexHandler(app))
	// Removed the about page, not relevant anymore
	e.GET("/upload", pages.UploadHandler(app))
	e.GET("/contact", pages.ContactHandler(app))
	e.GET("/guide", pages.GuideHandler(app))
	e.GET("/rules", pages.RulesHandler(app))
	e.GET("/terms-of-service", pages.TermsOfServiceHandler(app))
	// Auth
	e.GET("/login", pages.LoginHandler(app))
	e.GET("/register", pages.RegisterHandler(app))
	e.GET("/reset-password", pages.PasswordResetHandler(app))
	// News
	e.GET("/news", pages.NewsHandler(app))
	e.GET("/news/:slug", pages.NewsPostHandler(app))
	// Schematics
	e.GET("/schematics", pages.SchematicsHandler(app))
	e.GET("/schematics/:name", pages.SchematicHandler(app, searchService))
	e.GET("/search/:term", pages.SearchHandler(app, searchService))
	e.POST("/search/:term", pages.SearchHandler(app, searchService))
	e.GET("/search", pages.SearchHandler(app, searchService))
	e.GET("/search/", pages.SearchHandler(app, searchService))
	e.POST("/search/", pages.SearchHandler(app, searchService))
	e.POST("/search", pages.SearchPostHandler(app))
	// User
	e.GET("/author/:username", pages.ProfileHandler(app))
	e.GET("/profile", pages.ProfileHandler(app))
	// Fallback
	e.GET("/*", pages.FourOhFourHandler(app))

}

// cookieAuth was added so that requests can be authenticated on the backend when HTML templates are rendered
func cookieAuth(app *pocketbase.PocketBase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get(apis.ContextAuthRecordKey) != nil {
				return next(c)
			}

			cookie, err := c.Cookie(auth.CookieName)
			if err != nil {
				return next(c)
			}

			token := strings.TrimSpace(cookie.Value)

			claims, _ := security.ParseUnverifiedJWT(token)
			tokenType := cast.ToString(claims["type"])

			switch tokenType {
			case tokens.TypeAdmin:
				return next(c)

				// Disable admin auth via cookie for now

				//admin, err := app.Dao().FindAdminByToken(
				//	token,
				//	app.Settings().AdminAuthToken.Secret,
				//)
				//if err == nil && admin != nil {
				//	c.Set(apis.ContextAdminKey, admin)
				//}
			case tokens.TypeAuthRecord:
				record, err := app.Dao().FindAuthRecordByToken(
					token,
					app.Settings().RecordAuthToken.Secret,
				)
				if err == nil && record != nil {
					c.Set(apis.ContextAuthRecordKey, record)
				}
			}

			return next(c)
		}
	}
}

func legacyFileCompat(next echo.HandlerFunc) echo.HandlerFunc {
	fileMatches := map[string]string{
		"/wp-sitemap.xml": "/sitemaps/sitemap.xml",
	}
	return func(c echo.Context) error {
		path := c.Request().URL.Path
		for match, newRoute := range fileMatches {
			if path == match || strings.HasPrefix(path, match) {
				return c.Redirect(http.StatusMovedPermanently, newRoute)
			}
		}
		return next(c) // proceed with the request chain
	}
}

func legacyTagCompat(next echo.HandlerFunc) echo.HandlerFunc {
	// to /search/?tag=apple
	urlMatches := []string{
		"/schematics/tag/",
	}
	queryMatches := []string{
		"schematic_tags",
	}
	return func(c echo.Context) error {
		path := c.Request().URL.Path
		for _, match := range urlMatches {
			if strings.HasPrefix(path, match) {
				return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/?tag=%s", strings.ReplaceAll(strings.Replace(path, match, "", 1), "/", "")))
			}
		}
		query := c.Request().URL.Query()
		for _, match := range queryMatches {
			if query.Has(match) && query.Get(match) != "" {
				return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/?tag=%s", query.Get(match)))
			}
		}
		return next(c) // proceed with the request chain
	}
}

func legacySearchCompat(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Request().URL.Path
		query := c.Request().URL.Query()
		fmt.Sprintln("should redirect", c.Request().URL.Path, c.Request().URL.Query())
		if (path == "" || path == "/") && query.Has("s") && query.Get("s") != "" {
			searchSlug := slug.Make(query.Get("s"))
			return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/search/%s", searchSlug))
		}

		return next(c) // proceed with the request chain
	}
}
