package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// This migration is a no-op for PocketBase.
		// The password_reset_tokens table is created via the PostgreSQL
		// migration system (internal/database/migrations.go).
		return nil
	}, func(app core.App) error {
		return nil
	})
}
