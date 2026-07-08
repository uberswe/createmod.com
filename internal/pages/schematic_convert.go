package pages

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/store"
)

var convertTemplates = append([]string{
	"./template/convert.html",
}, commonTemplates...)

// convertFormats are the formats the converter UI offers, in display order.
var convertFormats = []struct {
	Format schematic.Format
	Slug   string // used in SEO pair URLs
	Label  string
	Ext    string
}{
	{schematic.FormatStructure, "nbt", "Create / vanilla structure (.nbt)", ".nbt"},
	{schematic.FormatSponge, "schem", "WorldEdit / Sponge (.schem)", ".schem"},
	{schematic.FormatLitematic, "litematic", "Litematica (.litematic)", ".litematic"},
	{schematic.FormatLegacy, "schematic", "Legacy MCEdit (.schematic)", ".schematic"},
	{schematic.FormatBlueprint, "blueprint", "MineColonies / Structurize (.blueprint)", ".blueprint"},
	{schematic.FormatBG, "bg", "Building Gadgets template (.json)", ".json"},
}

func convertFormatBySlug(slug string) (schematic.Format, string, bool) {
	for _, f := range convertFormats {
		if f.Slug == slug {
			return f.Format, f.Ext, true
		}
	}
	return schematic.FormatUnknown, "", false
}

func convertLabel(f schematic.Format) string {
	for _, cf := range convertFormats {
		if cf.Format == f {
			return cf.Label
		}
	}
	if f == schematic.FormatSable {
		return "Sable Blueprint (read-only)"
	}
	return string(f)
}

// ConvertPair describes one from→to SEO landing page.
type ConvertPair struct {
	FromSlug, ToSlug   string
	FromLabel, ToLabel string
}

// ConvertPairs lists every supported ordered conversion pair (used for the
// SEO landing pages and the sitemap).
func ConvertPairs() []ConvertPair {
	var out []ConvertPair
	for _, from := range convertFormats {
		for _, to := range convertFormats {
			if from.Slug == to.Slug {
				continue
			}
			out = append(out, ConvertPair{
				FromSlug: from.Slug, ToSlug: to.Slug,
				FromLabel: from.Label, ToLabel: to.Label,
			})
		}
	}
	return out
}

type convertPageData struct {
	DefaultData
	// Pair page fields; empty on the generic converter.
	FromSlug, ToSlug   string
	FromLabel, ToLabel string
	Pairs              []ConvertPair
}

// ConvertToolHandler renders the generic converter page (/tools/convert).
func ConvertToolHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return convertPage(registry, cacheService, appStore, "", "")
}

// ConvertPairHandler renders a from→to SEO landing page
// (/tools/convert/{from}-to-{to}).
func ConvertPairHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		pair := e.Request.PathValue("pair")
		parts := strings.SplitN(pair, "-to-", 2)
		if len(parts) != 2 {
			return FourOhFourHandler(registry, appStore)(e)
		}
		if _, _, ok := convertFormatBySlug(parts[0]); !ok {
			return FourOhFourHandler(registry, appStore)(e)
		}
		if _, _, ok := convertFormatBySlug(parts[1]); !ok || parts[0] == parts[1] {
			return FourOhFourHandler(registry, appStore)(e)
		}
		return convertPage(registry, cacheService, appStore, parts[0], parts[1])(e)
	}
}

func convertPage(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, fromSlug, toSlug string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		setPublicCacheControl(e, 300)
		d := convertPageData{Pairs: ConvertPairs()}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.HideOutstream = true

		if fromSlug != "" {
			fromFormat, _, _ := convertFormatBySlug(fromSlug)
			toFormat, _, _ := convertFormatBySlug(toSlug)
			d.FromSlug, d.ToSlug = fromSlug, toSlug
			d.FromLabel, d.ToLabel = convertLabel(fromFormat), convertLabel(toFormat)
			d.Title = fmt.Sprintf(i18n.T(d.Language, "Convert %s to %s - Free Minecraft Schematic Converter"), "."+fromSlug, "."+toSlug)
			d.Description = fmt.Sprintf(i18n.T(d.Language, "Convert Minecraft %s schematic files to %s online, free, in your browser. Blocks, block entities and Create mod data carry over."), "."+fromSlug, "."+toSlug)
			d.Slug = "/tools/convert/" + fromSlug + "-to-" + toSlug
			d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "Schematic Converter"), "/tools/convert", fmt.Sprintf(".%s → .%s", fromSlug, toSlug))
		} else {
			d.Title = i18n.T(d.Language, "Minecraft Schematic Converter - .nbt, .schem, .litematic")
			d.Description = i18n.T(d.Language, "Convert Minecraft schematics between Create .nbt, WorldEdit .schem and Litematica .litematic formats. Free, online, no signup.")
			d.Slug = "/tools/convert"
			d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "Schematic Converter"))
		}

		html, err := registry.LoadFiles(convertTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// ConvertAPIHandler converts an uploaded schematic and streams the result
// back as a download. Stateless: nothing is persisted. Warnings travel in
// the X-Conversion-Warnings header (JSON array) so the page can surface
// lossiness without a second request.
// POST /api/convert  (multipart: file, to)
func ConvertAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(maxUploadSize + 1<<20); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid form"})
		}
		toSlug := e.Request.FormValue("to")
		target, ext, ok := convertFormatBySlug(toSlug)
		if !ok {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "unsupported target format"})
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

		res, err := schematic.Convert(data, target)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}

		base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
		outName := sanitizeFilename(base + ext)
		if len(res.Warnings) > 0 {
			if wj, err := json.Marshal(res.Warnings); err == nil {
				e.Response.Header().Set("X-Conversion-Warnings", string(wj))
			}
		}
		e.Response.Header().Set("X-Detected-Format", string(res.From))
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeContentDispositionFilename(outName)+"\"")
		return e.Blob(http.StatusOK, "application/octet-stream", res.Data)
	}
}

// ConvertInspectHandler identifies an uploaded schematic without converting.
// POST /api/convert/inspect  (multipart: file)
func ConvertInspectHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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
		resp := map[string]interface{}{
			"format": string(format),
			"label":  convertLabel(format),
		}
		if s, err := schematic.Read(data, format); err == nil {
			caps := s.Capabilities()
			resp["size"] = caps.Size
			resp["blockCount"] = caps.BlockCount
			resp["dataVersion"] = caps.DataVersion
			resp["hasBlockEntities"] = caps.HasBlockEntities
			resp["warnings"] = s.Meta.LossyNotes
		} else {
			resp["readable"] = false
			resp["error"] = convertUserError(err)
		}
		return writeJSON(e, http.StatusOK, resp)
	}
}

// convertUserError trims internal prefixes for display.
func convertUserError(err error) string {
	return strings.TrimPrefix(err.Error(), "schematic: ")
}
