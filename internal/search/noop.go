package search

import (
	"context"
	"errors"
)

// noopEngine is a no-op SearchEngine used when no real backend (e.g. Meilisearch) is configured.
// It returns empty results for all operations instead of panicking on nil.
type noopEngine struct{}

// NewNoopEngine returns a SearchEngine that always returns empty results.
func NewNoopEngine() SearchEngine {
	return &noopEngine{}
}

func (n *noopEngine) Search(_ context.Context, _ SearchQuery) ([]string, error) {
	return nil, nil
}

func (n *noopEngine) SearchSimilar(_ context.Context, _ string, _ []string, _ int) ([]string, error) {
	return nil, nil
}

func (n *noopEngine) Suggest(_ string, _ int) []Suggestion {
	return nil
}

func (n *noopEngine) Ready() bool {
	return false
}

func (n *noopEngine) Health(_ context.Context) error {
	return errors.New("no search engine configured")
}
