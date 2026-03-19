package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

// MeiliEngine implements SearchEngine using a Meilisearch index.
type MeiliEngine struct {
	client   meilisearch.ServiceManager
	indexUID string
	fallback *Service // Bleve fallback for suggest, trending sort, and errors
}

// NewMeiliEngine creates a SearchEngine backed by Meilisearch.
func NewMeiliEngine(client meilisearch.ServiceManager, indexUID string, fallback *Service) *MeiliEngine {
	return &MeiliEngine{
		client:   client,
		indexUID: indexUID,
		fallback: fallback,
	}
}

func (m *MeiliEngine) Search(ctx context.Context, q SearchQuery) ([]string, error) {
	// Trending sort not supported by Meilisearch; fall back to Bleve.
	if q.Order == TrendingOrder {
		return m.fallbackSearch(q, "trending sort not supported in meili")
	}

	filter := m.buildFilter(q)
	sort := m.buildSort(q.Order)

	searchReq := &meilisearch.SearchRequest{
		Limit:  5000,
		Filter: filter,
		Sort:   sort,
	}

	index := m.client.Index(m.indexUID)
	result, err := index.SearchWithContext(ctx, q.Term, searchReq)
	if err != nil {
		return m.fallbackSearch(q, fmt.Sprintf("meili search error: %v", err))
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		var doc struct {
			ID string `json:"id"`
		}
		if err := hit.DecodeInto(&doc); err != nil || doc.ID == "" {
			continue
		}
		ids = append(ids, doc.ID)
	}

	return ids, nil
}

func (m *MeiliEngine) Suggest(q string, limit int) []Suggestion {
	// Delegate to Bleve — in-memory suggest is fast and Meilisearch
	// doesn't have an equivalent autocomplete API.
	return m.fallback.Suggest(q, limit)
}

func (m *MeiliEngine) Ready() bool {
	if m.client == nil {
		return false
	}
	_, err := m.client.Health()
	return err == nil
}

func (m *MeiliEngine) Health(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("meili client not configured")
	}
	_, err := m.client.Health()
	return err
}

// fallbackSearch logs a warning and falls back to Bleve base search.
func (m *MeiliEngine) fallbackSearch(q SearchQuery, reason string) ([]string, error) {
	slog.Warn("meili: falling back to bleve", "reason", reason, "index", m.indexUID)
	ids := m.fallback.Search(q.Term, q.Order, q.Rating, q.Category, q.Tags,
		q.MinecraftVersion, q.CreateVersion, q.HidePaid)
	return ids, nil
}

// buildFilter constructs a Meilisearch filter string from SearchQuery parameters.
func (m *MeiliEngine) buildFilter(q SearchQuery) string {
	var parts []string

	if q.Rating > 0 {
		parts = append(parts, fmt.Sprintf("rating >= %d", q.Rating))
	}

	if q.Category != "" && q.Category != "all" {
		cat := strings.ReplaceAll(q.Category, "-", " ")
		// Meilisearch filter values need quoting.
		parts = append(parts, fmt.Sprintf(`categories = "%s"`, escapeMeiliString(cat)))
	}

	if len(q.Tags) > 0 && !(len(q.Tags) == 1 && q.Tags[0] == "all") {
		for _, tag := range q.Tags {
			normalized := strings.ReplaceAll(tag, "-", " ")
			parts = append(parts, fmt.Sprintf(`tags = "%s"`, escapeMeiliString(normalized)))
		}
	}

	if q.MinecraftVersion != "" && q.MinecraftVersion != "all" {
		parts = append(parts, fmt.Sprintf(`minecraft_version = "%s"`, escapeMeiliString(q.MinecraftVersion)))
	}

	if q.CreateVersion != "" && q.CreateVersion != "all" {
		parts = append(parts, fmt.Sprintf(`create_version = "%s"`, escapeMeiliString(q.CreateVersion)))
	}

	if q.HidePaid {
		parts = append(parts, "paid = false")
	}

	return strings.Join(parts, " AND ")
}

// buildSort maps the order constant to Meilisearch sort syntax.
func (m *MeiliEngine) buildSort(order int) []string {
	switch order {
	case NewestOrder:
		return []string{"created_timestamp:desc"}
	case OldestOrder:
		return []string{"created_timestamp:asc"}
	case HighestRatingOrder:
		return []string{"rating:desc"}
	case LowestRatingOrder:
		return []string{"rating:asc"}
	case MostViewedOrder:
		return []string{"views:desc"}
	case LeastViewedOrder:
		return []string{"views:asc"}
	default:
		// BestMatch, Trending: use Meilisearch relevancy (no sort).
		return nil
	}
}

// escapeMeiliString escapes double quotes in filter values.
func escapeMeiliString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
