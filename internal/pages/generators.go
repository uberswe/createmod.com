package pages

import (
	"createmod/internal/cache"
	"createmod/internal/generator"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"net/http"
)

var generatorPropellerTemplates = append([]string{
	"./template/generator-propeller.html",
}, commonTemplates...)

var generatorBalloonTemplates = append([]string{
	"./template/generator-balloon.html",
}, commonTemplates...)

var generatorHullTemplates = append([]string{
	"./template/generator-hull.html",
}, commonTemplates...)

type GeneratorData struct {
	DefaultData
}

func GeneratorPropellerHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Propeller Generator")
		d.Description = "Generate custom propeller schematics for Minecraft Create mod airships."
		d.Slug = "/generators/propeller"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators/propeller", i18n.T(d.Language, "Propeller"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorPropellerTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func GeneratorBalloonHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Airship Balloon Generator")
		d.Description = "Generate custom airship balloon and envelope schematics for Minecraft Create mod."
		d.Slug = "/generators/balloon"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators/balloon", i18n.T(d.Language, "Balloon"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorBalloonTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func GeneratorHullHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := GeneratorData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Ship Hull Generator")
		d.Description = "Generate custom ship hull schematics for Minecraft Create mod airships."
		d.Slug = "/generators/hull"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, "Generators", "/generators/hull", i18n.T(d.Language, "Ship Hull"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(generatorHullTemplates...).Render(d)
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
