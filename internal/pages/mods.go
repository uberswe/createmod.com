package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/modmeta"
	"createmod/internal/store"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var modsTemplates = append([]string{
	"./template/mods.html",
}, commonTemplates...)

var modDetailTemplates = append([]string{
	"./template/mod_detail.html",
	"./template/include/schematic_card.html",
}, commonTemplates...)

// ModEntry represents a single mod with its display info and schematic count.
type ModEntry struct {
	Slug          string
	Name          string
	Description   string
	IconURL       string
	ModrinthURL   string
	CurseForgeURL string
	Count         int
	IsVanilla     bool
}

// ModsListData holds the data for the mods listing page.
type ModsListData struct {
	DefaultData
	Mods      []ModEntry
	TotalMods int
}

// ModDetailData holds the data for a single mod's schematics page.
type ModDetailData struct {
	DefaultData
	Mod        ModEntry
	Subtitle   string
	Schematics []models.Schematic
	Page       int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	TotalCount int
}

const modsCacheKey = "mods_listing"

// ModsHandler renders the mods listing page at GET /mods.
func ModsHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, modMetaService *modmeta.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		mods, found := getCachedMods(cacheService)
		if !found {
			var err error
			mods, err = queryModEntries(app)
			if err != nil {
				return err
			}
			enrichModEntries(app, mods, modMetaService)
			cacheService.SetWithTTL(modsCacheKey, mods, 30*time.Minute)
		}

		d := ModsListData{
			Mods:      mods,
			TotalMods: len(mods),
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Mods")
		d.Description = i18n.T(d.Language, "Browse schematics by mod")
		d.Slug = "/mods"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(modsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// ModDetailHandler renders a specific mod's schematics at GET /mods/{slug}.
func ModDetailHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, modMetaService *modmeta.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/mods"))
		}

		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		limit := pageSize + 1
		offset := (page - 1) * pageSize

		isVanilla := slug == "vanilla"
		var results []*core.Record
		var totalCount int
		var err error

		if isVanilla {
			results, totalCount, err = queryVanillaSchematics(app, limit, offset)
		} else {
			results, totalCount, err = queryModSchematics(app, slug, limit, offset)
		}
		if err != nil {
			return err
		}

		hasNext := len(results) > pageSize
		if hasNext {
			results = results[:pageSize]
		}

		caser := cases.Title(language.English)
		modName := caser.String(strings.ReplaceAll(slug, "_", " "))
		if isVanilla {
			modName = "Vanilla"
		}

		mod := ModEntry{
			Slug:      slug,
			Name:      modName,
			Count:     totalCount,
			IsVanilla: isVanilla,
		}

		// Enrich with metadata from Modrinth/CurseForge
		if !isVanilla && modMetaService != nil {
			if meta := modMetaService.GetMetadata(app, slug); meta != nil {
				if meta.DisplayName != "" {
					mod.Name = meta.DisplayName
				}
				mod.Description = meta.Description
				mod.IconURL = meta.IconURL
				mod.ModrinthURL = meta.ModrinthURL
				mod.CurseForgeURL = meta.CurseForgeURL
			}
		}

		d := ModDetailData{
			Mod:        mod,
			Schematics: MapResultsToSchematic(app, results, cacheService),
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
			TotalCount: totalCount,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/mods/%s?p=%d", slug, page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/mods/%s?p=%d", slug, page+1)
		}

		d.Populate(e)
		d.Title = mod.Name + " " + i18n.T(d.Language, "Schematics")
		if isVanilla {
			d.Subtitle = i18n.T(d.Language, "Schematics that require no mods in Minecraft")
			d.Description = d.Subtitle
		} else {
			d.Subtitle = fmt.Sprintf(i18n.T(d.Language, "Schematics using the %s mod in Minecraft"), mod.Name)
			d.Description = d.Subtitle
		}
		d.Slug = "/mods/" + slug
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(modDetailTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// getCachedMods retrieves the mod entries from cache if available.
func getCachedMods(cacheService *cache.Service) ([]ModEntry, bool) {
	val, found := cacheService.Get(modsCacheKey)
	if !found {
		return nil, false
	}
	mods, ok := val.([]ModEntry)
	if !ok {
		return nil, false
	}
	return mods, true
}

// queryModEntries fetches all mods and their schematic counts from the database.
func queryModEntries(app *pocketbase.PocketBase) ([]ModEntry, error) {
	caser := cases.Title(language.English)

	// Query mod names and counts using json_each
	modRows := []struct {
		ModName string `db:"mod_name"`
		Count   int    `db:"count"`
	}{}

	err := app.DB().NewQuery(`
		SELECT j.value AS mod_name, COUNT(DISTINCT schematics.id) AS count
		FROM schematics, json_each(schematics.mods) AS j
		WHERE (schematics.deleted = '' OR schematics.deleted IS NULL)
		  AND schematics.moderated = 1
		  AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))
		GROUP BY j.value
		ORDER BY count DESC
	`).All(&modRows)
	if err != nil {
		return nil, fmt.Errorf("querying mod entries: %w", err)
	}

	// Query vanilla count
	var vanillaCount int
	err = app.DB().NewQuery(`
		SELECT COUNT(*) FROM schematics
		WHERE (deleted = '' OR deleted IS NULL)
		  AND moderated = 1
		  AND (scheduled_at IS NULL OR scheduled_at <= DATETIME('now'))
		  AND (mods IS NULL OR mods = '' OR mods = '[]' OR mods = 'null')
	`).Row(&vanillaCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("querying vanilla count: %w", err)
	}

	// Build entries: Vanilla first, then mods sorted by count desc
	entries := make([]ModEntry, 0, len(modRows)+1)
	if vanillaCount > 0 {
		entries = append(entries, ModEntry{
			Slug:      "vanilla",
			Name:      "Vanilla",
			Count:     vanillaCount,
			IsVanilla: true,
		})
	}

	for _, row := range modRows {
		name := strings.TrimSpace(row.ModName)
		if name == "" {
			continue
		}
		entries = append(entries, ModEntry{
			Slug:  strings.ToLower(strings.ReplaceAll(name, " ", "_")),
			Name:  caser.String(strings.ReplaceAll(name, "_", " ")),
			Count: row.Count,
		})
	}

	return entries, nil
}

// queryModSchematics fetches schematics that include the given mod.
func queryModSchematics(app *pocketbase.PocketBase, mod string, limit, offset int) ([]*core.Record, int, error) {
	// Get the total count for this mod
	var totalCount int
	err := app.DB().NewQuery(`
		SELECT COUNT(DISTINCT schematics.id)
		FROM schematics, json_each(schematics.mods) AS j
		WHERE j.value = {:mod}
		  AND (schematics.deleted = '' OR schematics.deleted IS NULL)
		  AND schematics.moderated = 1
		  AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))
	`).Bind(dbx.Params{"mod": mod}).Row(&totalCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, fmt.Errorf("counting mod schematics: %w", err)
	}

	// Get the IDs for this page
	idRows := []struct {
		ID string `db:"id"`
	}{}
	err = app.DB().NewQuery(`
		SELECT DISTINCT schematics.id
		FROM schematics, json_each(schematics.mods) AS j
		WHERE j.value = {:mod}
		  AND (schematics.deleted = '' OR schematics.deleted IS NULL)
		  AND schematics.moderated = 1
		  AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))
		ORDER BY schematics.created DESC
		LIMIT {:limit} OFFSET {:offset}
	`).Bind(dbx.Params{"mod": mod, "limit": limit, "offset": offset}).All(&idRows)
	if err != nil {
		return nil, 0, fmt.Errorf("querying mod schematics: %w", err)
	}

	if len(idRows) == 0 {
		return nil, totalCount, nil
	}

	// Fetch full records by IDs
	ids := make([]string, len(idRows))
	for i, row := range idRows {
		ids[i] = row.ID
	}

	results, err := app.FindRecordsByIds("schematics", ids)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching schematic records: %w", err)
	}

	// Preserve the order from the query
	ordered := orderRecordsByIDs(results, ids)
	return ordered, totalCount, nil
}

// queryVanillaSchematics fetches schematics with no mods.
func queryVanillaSchematics(app *pocketbase.PocketBase, limit, offset int) ([]*core.Record, int, error) {
	vanillaFilter := "deleted = '' && moderated = true && (scheduled_at = null || scheduled_at <= {:now}) && (mods = null || mods = '' || mods = '[]' || mods = 'null')"

	// Get total count
	var totalCount int
	err := app.DB().NewQuery(`
		SELECT COUNT(*) FROM schematics
		WHERE (deleted = '' OR deleted IS NULL)
		  AND moderated = 1
		  AND (scheduled_at IS NULL OR scheduled_at <= DATETIME('now'))
		  AND (mods IS NULL OR mods = '' OR mods = '[]' OR mods = 'null')
	`).Row(&totalCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, fmt.Errorf("counting vanilla schematics: %w", err)
	}

	results, err := app.FindRecordsByFilter(
		"schematics",
		vanillaFilter,
		"-created",
		limit,
		offset,
		dbx.Params{"now": time.Now()},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("querying vanilla schematics: %w", err)
	}

	return results, totalCount, nil
}

// enrichModEntries populates metadata fields on mod entries from the mod_metadata collection.
func enrichModEntries(app *pocketbase.PocketBase, entries []ModEntry, modMetaService *modmeta.Service) {
	if modMetaService == nil {
		return
	}
	namespaces := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsVanilla {
			namespaces = append(namespaces, e.Slug)
		}
	}
	metaMap := modMetaService.GetMetadataMap(app, namespaces)
	for i := range entries {
		if entries[i].IsVanilla {
			continue
		}
		if meta, ok := metaMap[entries[i].Slug]; ok {
			if meta.DisplayName != "" {
				entries[i].Name = meta.DisplayName
			}
			entries[i].Description = meta.Description
			entries[i].IconURL = meta.IconURL
			entries[i].ModrinthURL = meta.ModrinthURL
			entries[i].CurseForgeURL = meta.CurseForgeURL
		}
	}
}

// orderRecordsByIDs returns records in the order specified by ids.
func orderRecordsByIDs(records []*core.Record, ids []string) []*core.Record {
	byID := make(map[string]*core.Record, len(records))
	for _, r := range records {
		byID[r.Id] = r
	}
	ordered := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			ordered = append(ordered, r)
		}
	}
	return ordered
}
