package pages

import (
	"bytes"
	"createmod/internal/cache"
	"createmod/internal/generator"
	"createmod/internal/generator/render"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
)

var generatorsLandingTemplates = append([]string{
	"./template/generators.html",
}, commonTemplates...)

var generatorPropellerTemplates = append([]string{
	"./template/generator-propeller.html",
}, commonTemplates...)

var generatorBalloonTemplates = append([]string{
	"./template/generator-balloon.html",
}, commonTemplates...)

var generatorHullTemplates = append([]string{
	"./template/generator-hull.html",
}, commonTemplates...)

var generatorGuideTemplates = append([]string{
	"./template/generator-guide.html",
}, commonTemplates...)

type GeneratorData struct {
	DefaultData
	InitHash      string
	GeneratorType string
}

func GeneratorsLandingHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Generators")
		d.Description = "Generate custom schematics for your Minecraft Create mod builds."
		d.Slug = "/generators"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorsLandingTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func generatorThumbnail(_ *storage.Service, hash string, _ *server.RequestEvent) string {
	if hash != "" {
		return "https://createmod.com/api/generators/preview/" + hash
	}
	return "https://createmod.com/assets/x/logo_sq_lg.png"
}

func GeneratorPropellerHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.InitHash = chi.URLParam(e.Request, "hash")
		d.Title = i18n.T(d.Language, "Propeller Generator")
		d.Description = "Generate custom propeller schematics for Minecraft Create mod airships."
		d.Slug = "/generators/propeller"
		d.Thumbnail = generatorThumbnail(storageSvc, d.InitHash, e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators", i18n.T(d.Language, "Propeller"))
		d.BreadcrumbOverlay = true
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorPropellerTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func GeneratorBalloonHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.InitHash = chi.URLParam(e.Request, "hash")
		d.Title = i18n.T(d.Language, "Airship Balloon Generator")
		d.Description = "Generate custom airship balloon and envelope schematics for Minecraft Create mod."
		d.Slug = "/generators/balloon"
		d.Thumbnail = generatorThumbnail(storageSvc, d.InitHash, e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators", i18n.T(d.Language, "Balloon"))
		d.BreadcrumbOverlay = true
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorBalloonTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func GeneratorHullHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.InitHash = chi.URLParam(e.Request, "hash")
		d.Title = i18n.T(d.Language, "Ship Hull Generator")
		d.Description = "Generate custom ship hull schematics for Minecraft Create mod airships."
		d.Slug = "/generators/hull"
		d.Thumbnail = generatorThumbnail(storageSvc, d.InitHash, e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators", i18n.T(d.Language, "Ship Hull"))
		d.BreadcrumbOverlay = true
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorHullTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

var generatorGuideNames = map[string]string{
	"propeller": "Propeller",
	"balloon":   "Balloon",
	"hull":      "Ship Hull",
}

func GeneratorGuideHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		genType := chi.URLParam(e.Request, "type")
		name, ok := generatorGuideNames[genType]
		if !ok {
			return e.NotFoundError("generator not found", nil)
		}
		d := GeneratorData{}
		d.Populate(e)
		d.InitHash = chi.URLParam(e.Request, "hash")
		d.GeneratorType = genType
		d.Title = i18n.T(d.Language, name+" Building Guide")
		d.Description = fmt.Sprintf("Step by step building guide for the %s generator.", name)
		d.Slug = "/generators/" + genType
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators", i18n.T(d.Language, name), "/generators/"+genType, "Guide")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorGuideTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

const maxGeneratedBlocks = 500000

func GeneratorPropellerAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var params generator.PropellerParams
		if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
			return e.BadRequestError("invalid parameters", nil)
		}
		result, err := generator.GeneratePropeller(params)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if len(result.Blocks) > maxGeneratedBlocks {
			return e.BadRequestError("too many blocks generated", nil)
		}
		return e.JSON(http.StatusOK, result)
	}
}

func GeneratorBalloonAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var params generator.BalloonParams
		if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
			return e.BadRequestError("invalid parameters", nil)
		}
		result, err := generator.GenerateBalloon(params)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if len(result.Blocks) > maxGeneratedBlocks {
			return e.BadRequestError("too many blocks generated", nil)
		}
		return e.JSON(http.StatusOK, result)
	}
}

func GeneratorHullAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var params generator.HullParams
		if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
			return e.BadRequestError("invalid parameters", nil)
		}
		result, err := generator.GenerateHull(params)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if len(result.Blocks) > maxGeneratedBlocks {
			return e.BadRequestError("too many blocks generated", nil)
		}
		return e.JSON(http.StatusOK, result)
	}
}

func GeneratorDownloadHandler(genType string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var result *generator.GenerateResult
		var filename string
		var err error

		switch genType {
		case "propeller":
			var params generator.PropellerParams
			if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
				return e.BadRequestError("invalid parameters", nil)
			}
			result, err = generator.GeneratePropeller(params)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}
			swept := "flat"
			if params.Swept {
				swept = "swept"
			}
			filename = fmt.Sprintf("propeller_%dblade_r%d_%s.nbt", params.Blades, params.Length, swept)
		case "balloon":
			var params generator.BalloonParams
			if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
				return e.BadRequestError("invalid parameters", nil)
			}
			result, err = generator.GenerateBalloon(params)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}
			filename = fmt.Sprintf("airship_%dx%dx%d.nbt", params.LengthX, params.WidthZ, params.HeightY)
		case "hull":
			var params generator.HullParams
			if err := json.NewDecoder(e.Request.Body).Decode(&params); err != nil {
				return e.BadRequestError("invalid parameters", nil)
			}
			result, err = generator.GenerateHull(params)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}
			filename = fmt.Sprintf("hull_%dx%dx%d.nbt", params.Length, params.Beam, params.Depth)
		default:
			return e.BadRequestError("unknown generator type", nil)
		}

		if len(result.Blocks) > maxGeneratedBlocks {
			return e.BadRequestError("too many blocks generated", nil)
		}

		data, err := generator.ExportNBT(result)
		if err != nil {
			return e.InternalServerError("failed to generate NBT file", nil)
		}

		e.Response.Header().Set("Content-Type", "application/octet-stream")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeContentDispositionFilename(filename)))
		e.Response.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		_, writeErr := e.Response.Write(data)
		return writeErr
	}
}

var validHashPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// GeneratorPreviewHandler serves server-rendered isometric previews for generator hashes.
// GET /api/generators/preview/{hash}
func GeneratorPreviewHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		hash := chi.URLParam(e.Request, "hash")
		if hash == "" || len(hash) > 300 || !validHashPattern.MatchString(hash) {
			return e.BadRequestError("invalid hash", nil)
		}

		etag := `"gen-` + hash[:min(len(hash), 16)] + `"`
		e.Response.Header().Set("ETag", etag)
		if match := e.Request.Header.Get("If-None-Match"); match == etag {
			e.Response.WriteHeader(http.StatusNotModified)
			return nil
		}
		e.Response.Header().Set("Cache-Control", "public, max-age=86400")

		result, _, err := generator.DecodeHash(hash)
		if err != nil {
			return e.BadRequestError("invalid generator hash", nil)
		}
		if len(result.Blocks) > maxGeneratedBlocks {
			return e.BadRequestError("too many blocks", nil)
		}

		img := render.Isometric(result)

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return e.InternalServerError("failed to encode image", nil)
		}

		e.Response.Header().Set("Content-Type", "image/png")
		return e.Blob(http.StatusOK, "image/png", buf.Bytes())
	}
}
