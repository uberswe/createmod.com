package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add composite index (type, created) on schematic_views for fast
		// hourly-stats queries: WHERE type = 0 AND created > cutoff.
		_, err := app.DB().NewQuery(`
			CREATE INDEX IF NOT EXISTS idx_sv_type_created
			ON schematic_views (type, created)
		`).Execute()
		if err != nil {
			return err
		}

		// Same composite index on schematic_downloads for consistency.
		_, err = app.DB().NewQuery(`
			CREATE INDEX IF NOT EXISTS idx_sdl_type_created
			ON schematic_downloads (type, created)
		`).Execute()
		return err
	}, func(app core.App) error {
		_, _ = app.DB().NewQuery(`DROP INDEX IF EXISTS idx_sv_type_created`).Execute()
		_, _ = app.DB().NewQuery(`DROP INDEX IF EXISTS idx_sdl_type_created`).Execute()
		return nil
	})
}
