package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Updates the "How to upload a schematic" guide to link to a concrete page (/upload)
// rather than the generic /guide page.
func init() {
	m.Register(func(app core.App) error {
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return nil
		}
		recs, err := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 10, 0, map[string]any{"t": "How to upload a schematic"})
		if err != nil {
			return nil
		}
		for _, r := range recs {
			// Only update if currently pointing to generic guide page
			if r.GetString("wiki_url") == "https://createmod.com/guide" || r.GetString("wiki_url") == "/guide" || r.GetString("wiki_url") == "" {
				r.Set("wiki_url", "/upload")
				_ = app.Save(r)
			}
		}
		return nil
	}, func(app core.App) error {
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return nil
		}
		recs, err := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 10, 0, map[string]any{"t": "How to upload a schematic"})
		if err != nil {
			return nil
		}
		for _, r := range recs {
			if r.GetString("wiki_url") == "/upload" {
				// revert to the generic guide page used previously
				r.Set("wiki_url", "https://createmod.com/guide")
				_ = app.Save(r)
			}
		}
		return nil
	})
}
