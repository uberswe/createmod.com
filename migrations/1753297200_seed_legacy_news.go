package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Seeds a few initial news posts to ensure /news isn't empty after migration
// from the legacy static template-based news page. This inserts records only if
// matching titles are not already present.
func init() {
	m.Register(func(app core.App) error {
		coll, err := app.FindCollectionByNameOrId("news")
		if err != nil || coll == nil {
			return nil
		}
		type seed struct{ Title, Excerpt, Content string }
		seeds := []seed{
			{
				Title:   "Welcome to the refreshed CreateMod.com",
				Excerpt: "We’ve updated the homepage layout and improved language switching.",
				Content: `<p>We recently overhauled parts of the site: the homepage now highlights Trending, Highest Rated, and Latest schematics. The language switcher now works consistently and supports HTMX navigation. Thanks for your feedback!</p>`,
			},
			{
				Title:   "Guides improvements",
				Excerpt: "New Guides listing and seeds to help you get started.",
				Content: `<p>There is now a dedicated Guides page listing entries with search and pagination. We also seeded a couple of example guides to help new users find their way.</p>`,
			},
			{
				Title:   "Login fixes",
				Excerpt: "Resolved issues with the login form submitting via GET and added backend handling.",
				Content: `<p>The login form now uses POST and we added a backend handler that integrates with PocketBase authentication (including HTMX redirects).</p>`,
			},
		}
		for _, s := range seeds {
			recs, _ := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 1, 0, map[string]any{"t": s.Title})
			if len(recs) > 0 {
				continue
			}
			rec := core.NewRecord(coll)
			rec.Set("title", s.Title)
			rec.Set("excerpt", s.Excerpt)
			rec.Set("content", s.Content)
			_ = app.Save(rec)
		}
		return nil
	}, func(app core.App) error {
		coll, err := app.FindCollectionByNameOrId("news")
		if err != nil || coll == nil {
			return nil
		}
		titles := []string{
			"Welcome to the refreshed CreateMod.com",
			"Guides improvements",
			"Login fixes",
		}
		for _, t := range titles {
			recs, _ := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 50, 0, map[string]any{"t": t})
			for _, r := range recs {
				_ = app.Delete(r)
			}
		}
		return nil
	})
}
