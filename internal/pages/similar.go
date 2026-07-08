package pages

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/similarity"
	"createmod/internal/store"
)

// similarMinOverall is the floor below which results are noise, not matches.
const similarMinOverall = 0.35

const similarMaxResults = 24

// ComponentView is one breakdown entry with a display-ready percentage.
type ComponentView struct {
	Name    string `json:"name"`
	Percent int    `json:"percent"`
}

// SimilarResultView is one match shaped for templates/JSON.
type SimilarResultView struct {
	Schematic models.Schematic `json:"schematic"`
	Percent   int              `json:"percent"`
	Breakdown []ComponentView  `json:"breakdown"`
}

func toComponentViews(cs []schematic.ComponentScore) []ComponentView {
	out := make([]ComponentView, 0, len(cs))
	for _, c := range cs {
		out = append(out, ComponentView{Name: c.Name, Percent: int(c.Score*100 + 0.5)})
	}
	return out
}

var similarTemplates = append([]string{
	"./template/similar.html",
	"./template/include/schematic_card.html",
}, commonTemplates...)

type similarPageData struct {
	DefaultData
	Source  models.Schematic
	Results []SimilarResultView
	Indexed int
}

// buildSimilarResults resolves index hits into card view models.
func buildSimilarResults(ctx context.Context, appStore *store.Store, cacheService *cache.Service, hits []similarity.Result, lang string) []SimilarResultView {
	if len(hits) == 0 {
		return nil
	}
	ids := make([]string, len(hits))
	for i, h := range hits {
		ids[i] = h.SchematicID
	}
	rows, err := appStore.Schematics.ListByIDs(ctx, ids)
	if err != nil {
		return nil
	}
	mapped := MapStoreSchematics(appStore, rows, cacheService)
	byID := make(map[string]models.Schematic, len(mapped))
	for _, m := range mapped {
		byID[m.ID] = m
	}
	out := make([]SimilarResultView, 0, len(hits))
	for _, h := range hits {
		m, ok := byID[h.SchematicID]
		if !ok {
			continue
		}
		m.Language = lang
		out = append(out, SimilarResultView{
			Schematic: m,
			Percent:   int(h.Similarity.Overall*100 + 0.5),
			Breakdown: toComponentViews(h.Similarity.Components),
		})
	}
	return out
}

// SimilarSchematicsHandler renders /schematics/{name}/similar.
func SimilarSchematicsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, simService *similarity.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		s, err := appStore.Schematics.GetByName(e.Request.Context(), name)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return FourOhFourHandler(registry, appStore)(e)
		}

		d := similarPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.NoIndex = true // dynamic thin content; the tool page carries the SEO
		d.Title = fmt.Sprintf(i18n.T(d.Language, "Schematics similar to %s"), s.Title)
		d.Description = i18n.T(d.Language, "Builds with a similar structure, shape and materials.")
		d.Slug = "/schematics/" + name + "/similar"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Schematics"), "/schematics", s.Title, "/schematics/"+name, i18n.T(d.Language, "Similar"))

		if simService != nil {
			d.Indexed = simService.Size()
			if fp := simService.Get(s.ID); fp != nil {
				hits := simService.FindSimilar(fp, s.ID, similarMaxResults, similarMinOverall)
				d.Results = buildSimilarResults(e.Request.Context(), appStore, cacheService, hits, d.Language)
			}
		}
		mapped := MapStoreSchematics(appStore, []store.Schematic{*s}, cacheService)
		if len(mapped) > 0 {
			d.Source = mapped[0]
			d.Source.Language = d.Language
		}

		html, err := registry.LoadFiles(similarTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

var similarToolTemplates = append([]string{
	"./template/similar_tool.html",
}, commonTemplates...)

// SimilarToolHandler renders /tools/similar — upload a schematic, search the
// library by structure.
func SimilarToolHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		setPublicCacheControl(e, 600)
		d := safetyPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "Find Similar Minecraft Schematics - Search by Structure")
		d.Description = i18n.T(d.Language, "Upload any schematic and find builds like it on CreateMod.com. Similarity is scored by shape, materials, Create components and proportions - with a full breakdown per match.")
		d.Slug = "/tools/similar"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "Find Similar"))
		html, err := registry.LoadFiles(similarToolTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// SimilarAPIHandler searches the library with an uploaded schematic.
// POST /api/similar (multipart: file). Stateless.
func SimilarAPIHandler(cacheService *cache.Service, appStore *store.Store, simService *similarity.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if simService == nil {
			return writeJSON(e, http.StatusServiceUnavailable, map[string]string{"error": "similarity index unavailable"})
		}
		if err := e.Request.ParseMultipartForm(maxUploadSize + 1<<20); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid form"})
		}
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing file"})
		}
		defer file.Close()
		if header.Size > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil || int64(len(data)) > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}

		format, err := schematic.Detect(data)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}
		model, err := schematic.Read(data, format)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}
		fp := schematic.ComputeFingerprint(model)
		hits := simService.FindSimilar(fp, "", similarMaxResults, similarMinOverall)
		lang := "en"
		results := buildSimilarResults(e.Request.Context(), appStore, cacheService, hits, lang)
		return writeJSON(e, http.StatusOK, map[string]interface{}{
			"indexed": simService.Size(),
			"results": results,
		})
	}
}

// GetSimilarAPIHandler returns similar schematics for a library schematic.
// GET /api/schematics/{name}/similar
func GetSimilarAPIHandler(cacheService *cache.Service, appStore *store.Store, simService *similarity.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if simService == nil {
			return writeJSON(e, http.StatusServiceUnavailable, map[string]string{"error": "similarity index unavailable"})
		}
		name := e.Request.PathValue("name")
		s, err := appStore.Schematics.GetByName(e.Request.Context(), name)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic not found"})
		}
		fp := simService.Get(s.ID)
		if fp == nil {
			return writeJSON(e, http.StatusOK, map[string]interface{}{"results": []SimilarResultView{}, "pending": true})
		}
		hits := simService.FindSimilar(fp, s.ID, similarMaxResults, similarMinOverall)
		results := buildSimilarResults(e.Request.Context(), appStore, cacheService, hits, "en")
		return writeJSON(e, http.StatusOK, map[string]interface{}{"results": results})
	}
}
