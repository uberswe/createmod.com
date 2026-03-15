package search

import "context"

// SearchEngine abstracts a search backend so different implementations
// (Bleve, Meilisearch) can be swapped via the A/B test variant router.
type SearchEngine interface {
	// Search returns schematic IDs matching the given query.
	Search(ctx context.Context, query SearchQuery) ([]string, error)
	// Suggest returns autocomplete suggestions for the given prefix.
	Suggest(q string, limit int) []Suggestion
	// Ready reports whether the engine can serve queries.
	Ready() bool
	// Health performs a deeper health check (e.g. ping remote service).
	Health(ctx context.Context) error
}

// SearchQuery contains all parameters for a search request.
type SearchQuery struct {
	Term             string
	Category         string
	MinecraftVersion string
	CreateVersion    string
	Order            int
	Rating           int
	Tags             []string
	HidePaid         bool
}
