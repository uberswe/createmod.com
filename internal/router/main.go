package router

import (
	"createmod/internal/pages"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"html/template"
	"os"
	"strings"
)

func Register(app *pocketbase.PocketBase, e *echo.Echo) {
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

	// Frontend routes
	e.GET("/dist/*", apis.StaticDirectoryHandler(os.DirFS("./web/dist"), false))
	e.GET("/static/*", apis.StaticDirectoryHandler(os.DirFS("./static/dist"), false))
	// Index
	e.GET("/", pages.IndexHandler(app))
	e.GET("/about", pages.AboutHandler(app))
	e.GET("/upload", pages.UploadHandler(app))
	e.GET("/contact", pages.ContactHandler(app))
	e.GET("/rules", pages.RulesHandler(app))
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
	// TODO add middleware to handle old search
	e.GET("/search/:term", pages.SearchHandler(app))
	// User
	e.GET("/author/:username", pages.ProfileHandler(app))
	// Fallback
	e.GET("/*", pages.FourOhFourHandler(app))

}
