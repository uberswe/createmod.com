package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Seeds a couple of example guides to help users discover the feature.
// Up: insert records if they don't already exist (by title).
// Down: delete the inserted records by title.
func init() {
	m.Register(func(app core.App) error {
		// Ensure guides collection exists
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return nil
		}
		type guideSeed struct{ Title, Excerpt, WikiURL, VideoURL string }
		seeds := []guideSeed{
			{
				Title:    "How to upload a schematic",
				Excerpt:  "Step-by-step guide to uploading your Create Mod schematic to CreateMod.com.",
				WikiURL:  "https://createmod.com/guide",
				VideoURL: "",
			},
			{
				Title:    "Getting started with Create",
				Excerpt:  "New to Create? Learn the basics and find your first schematics.",
				WikiURL:  "https://create.fandom.com/wiki/Create_Mod",
				VideoURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			},
		}
		for _, s := range seeds {
			recs, _ := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 1, 0, map[string]any{"t": s.Title})
			if len(recs) > 0 {
				continue
			}
			rec := core.NewRecord(coll)
			rec.Set("title", s.Title)
			rec.Set("name", s.Title)
			if s.Excerpt != "" {
				rec.Set("excerpt", s.Excerpt)
			}
			if s.WikiURL != "" {
				rec.Set("wiki_url", s.WikiURL)
			}
			if s.VideoURL != "" {
				rec.Set("video_url", s.VideoURL)
			}
			_ = app.Save(rec)
		}
		return nil
	}, func(app core.App) error {
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return nil
		}
		titles := []string{
			"How to upload a schematic",
			"Getting started with Create",
		}
		for _, t := range titles {
			recs, _ := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 10, 0, map[string]any{"t": t})
			for _, r := range recs {
				_ = app.Delete(r)
			}
		}
		return nil
	})
}
