package models

import "time"

// SchematicVersion is a minimal view model for exposing version history
// on a schematic. Snapshot contents are stored as raw JSON string in DB
// and are not required for the minimal UI.
type SchematicVersion struct {
	Version int
	Created time.Time
	Note    string
}
