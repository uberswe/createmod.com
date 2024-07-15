package router

import (
	"createmod/internal/pages"
	"createmod/internal/search"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
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

	glob, err := templateBuilder.Funcs(funcMap).ParseGlob("template/*/*.html")
	if err != nil {
		return
	}
	t := &Template{
		templates: template.Must(glob.ParseGlob("template/*.html")),
	}

	e.Renderer = t
	e.Use(legacySearchCompat)
	// Frontend routes
	e.GET("/dist/*", apis.StaticDirectoryHandler(os.DirFS("./web/dist"), false))
	e.GET("/static/*", apis.StaticDirectoryHandler(os.DirFS("./static/dist"), false))
	// Index
	e.GET("/", pages.IndexHandler(app))
	e.GET("/about", pages.AboutHandler(app))
	e.GET("/upload", pages.UploadHandler(app))
	e.GET("/contact", pages.ContactHandler(app))
	e.GET("/guide", pages.GuideHandler(app))
	e.GET("/rules", pages.RulesHandler(app))
	e.GET("/terms-of-service", pages.TermsOfServiceHandler(app))
	// Auth
	e.GET("/login", pages.LoginHandler(app))
	e.GET("/register", pages.RegisterHandler(app))
	e.GET("/reset-password", pages.PasswordResetHandler(app))
	// User
	e.GET("/rules", pages.RulesHandler(app))
	// News
	e.GET("/news", pages.NewsHandler(app))
	e.GET("/news/:slug", pages.NewsPostHandler(app))
	// Schematics
	e.GET("/schematics", pages.SchematicsHandler(app))
	e.GET("/schematics/:name", pages.SchematicHandler(app))
	// Needs backwards compatible with
	// Tags /?schematic_tags=elevator
	// TODO fix this
	e.GET("/schematics/tag/:tag", pages.SchematicHandler(app))
	e.GET("/schematics/category/:category", pages.SchematicHandler(app))
	// Backwards compat
	// /schematic_categories/player-transport
	e.GET("/schematic_categories/:slug", pages.SchematicHandler(app))
	// Search
	// Needs to be backwards compatible with
	// /?s=searchterm&id=95&post_type=schematics
	// /?s=search+term+1+2+3&id=95&post_type=schematics
	// legacySearchCompat middleware should handle this
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
