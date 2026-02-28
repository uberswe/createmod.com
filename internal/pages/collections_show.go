package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/translation"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"html/template"
	"net/http"
	"sort"
	"time"
)

var collectionsShowTemplates = append([]string{
	"./template/collections_show.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

// CollectionsShowData represents data for a single collection view page.
type CollectionsShowData struct {
	DefaultData
	TitleText       string
	DescriptionText string // raw description from DB (may be empty)
	DescriptionHTML template.HTML
	BannerURL       string
	Views           int
	Featured        bool
	Published       bool
	IsOwner         bool
	Schematics      []models.Schematic
	ShareURL        string
	CollectionID    string
	AuthorName      string
	IsTranslated    bool
}

// CollectionsShowHandler renders a basic collection detail page by slug or id.
// It degrades gracefully if the PocketBase collection is not available yet.
func CollectionsShowHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, translationService *translation.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")

		d := CollectionsShowData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)
		d.Slug = "/collections/" + slug

		if coll, err := app.FindCollectionByNameOrId("collections"); err == nil && coll != nil {
			// Try to find by slug first, fallback to id
			var rec *core.Record
			// by slug
			if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
				rec = r[0]
			}
			if rec == nil {
				if r, err := app.FindRecordById(coll.Id, slug); err == nil {
					rec = r
				}
			}
			if rec != nil {
				d.Published = rec.GetBool("published")
				d.CollectionID = rec.Id
				d.TitleText = rec.GetString("title")
				if d.TitleText == "" {
					d.TitleText = rec.GetString("name")
				}
				d.DescriptionText = rec.GetString("description")
				d.DescriptionHTML = template.HTML(d.DescriptionText)
				d.BannerURL = rec.GetString("banner_url")
				d.Featured = rec.GetBool("featured")
				if e.Auth != nil && rec.GetString("author") == e.Auth.Id {
					d.IsOwner = true
				}

				// Load author name
				if authorID := rec.GetString("author"); authorID != "" {
					if u := findUserFromID(app, authorID); u != nil {
						d.AuthorName = u.Username
					}
				}

				// Build the share URL:
				// Public (published) collections use the SEO-friendly slug URL.
				// Private (unpublished) collections use the record ID so the link
				// works for anyone who has it, without requiring a slug.
				scheme := "https"
				host := e.Request.Host
				if host == "" {
					host = "createmod.com"
				}
				if e.Request.TLS == nil {
					scheme = "http"
				}
				if d.Published {
					collSlug := rec.GetString("slug")
					if collSlug == "" {
						collSlug = rec.Id
					}
					d.ShareURL = fmt.Sprintf("%s://%s/collections/%s", scheme, host, collSlug)
				} else {
					d.ShareURL = fmt.Sprintf("%s://%s/collections/%s", scheme, host, rec.Id)
				}

				// Visibility: unpublished collections are accessible to anyone with the
				// direct link (by ID). Only the owner sees them in listings.
				// No visibility block here — if the record was found, it's viewable.

				// Views increment with IP-based deduplication (1-hour window)
				currentViews := rec.GetInt("views")
				clientIP := e.RealIP()
				ipKey := fmt.Sprintf("viewip:%s:coll:%s", clientIP, rec.Id)
				if clientIP != "" && cacheService != nil {
					if _, already := cacheService.Get(ipKey); already {
						// Same IP viewed this collection recently — skip increment
						d.Views = currentViews
					} else {
						cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
						rec.Set("views", currentViews+1)
						if err := app.Save(rec); err == nil {
							d.Views = currentViews + 1
						} else {
							d.Views = currentViews
							app.Logger().Warn("collections: failed to increment views", "error", err)
						}
					}
				} else {
					d.Views = currentViews
				}

				// Load associated schematics
				d.Schematics = loadCollectionSchematics(app, cacheService, rec)

				// Translation: show translated content if user's language is not English
				showOriginal := e.Request.URL.Query().Get("lang") == "original"
				if !showOriginal && translationService != nil && d.Language != "" && d.Language != "en" {
					t := translationService.GetCollectionTranslation(app, cacheService, rec.Id, d.Language)
					if t != nil && t.Title != "" {
						d.TitleText = t.Title
						if t.Description != "" {
							d.DescriptionText = t.Description
							d.DescriptionHTML = template.HTML(t.Description)
						}
						d.IsTranslated = true
					}
				}

				// SEO/meta
				d.Title = d.TitleText
				if d.Title == "" {
					d.Title = "Collection"
				}
				if d.DescriptionText != "" {
					d.Description = d.DescriptionText
				} else {
					d.Description = "Collection details"
				}
			} else {
				// Not found
				d.Title = "Collection not found"
				d.Description = "We couldn't find this collection."
			}
		} else {
			// No PB schema available yet
			d.Title = "Collection"
			d.Description = "Collection details"
		}

		html, err := registry.LoadFiles(collectionsShowTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// loadCollectionSchematics discovers and loads schematics associated with a
// collection record, using the same join-table-then-multi-rel strategy as the
// edit and download handlers.
func loadCollectionSchematics(app *pocketbase.PocketBase, cacheService *cache.Service, rec *core.Record) []models.Schematic {
	type pair struct {
		sid string
		pos int
		idx int
	}

	ids := make([]string, 0, 64)

	// Start with multi-rel field as fallback.
	if rel := rec.GetStringSlice("schematics"); len(rel) > 0 {
		seen := make(map[string]struct{}, len(rel))
		for _, s := range rel {
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			ids = append(ids, s)
		}
	}

	// Try join tables (preferred — supports position-based ordering).
	for _, jn := range []string{"collections_schematics", "collection_schematics"} {
		jcoll, jerr := app.FindCollectionByNameOrId(jn)
		if jerr != nil || jcoll == nil {
			continue
		}
		links, lerr := app.FindRecordsByFilter(jcoll.Id, "collection = {:c}", "-created", 5000, 0, dbx.Params{"c": rec.Id})
		if lerr != nil || len(links) == 0 {
			continue
		}
		best := make([]pair, 0, len(links))
		seen := make(map[string]struct{}, len(links))
		for i, l := range links {
			sid := l.GetString("schematic")
			if sid == "" {
				continue
			}
			if _, ok := seen[sid]; ok {
				continue
			}
			seen[sid] = struct{}{}
			best = append(best, pair{sid: sid, pos: l.GetInt("position"), idx: i})
		}
		anyPos := false
		for _, it := range best {
			if it.pos > 0 {
				anyPos = true
				break
			}
		}
		if anyPos {
			sort.SliceStable(best, func(i, j int) bool {
				if best[i].pos != best[j].pos {
					return best[i].pos < best[j].pos
				}
				return best[i].idx < best[j].idx
			})
		}
		ids = ids[:0]
		for _, it := range best {
			ids = append(ids, it.sid)
		}
		break // prefer the first join table found
	}

	if len(ids) == 0 {
		return nil
	}

	// Load schematic records by ID, preserving order.
	smColl, err := app.FindCollectionByNameOrId("schematics")
	if err != nil || smColl == nil {
		return nil
	}
	records := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		r, err := app.FindRecordById(smColl.Id, id)
		if err != nil || r == nil {
			continue
		}
		records = append(records, r)
	}
	return MapResultsToSchematic(app, records, cacheService)
}
