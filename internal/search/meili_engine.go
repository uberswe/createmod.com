package search

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/meilisearch/meilisearch-go"
)

// MeiliEngine implements SearchEngine using a Meilisearch index.
type MeiliEngine struct {
	client   meilisearch.ServiceManager
	indexUID string
	svc      *Service // for suggest, trending scores, and filter index
	resync   sync.Once
}

// NewMeiliEngine creates a SearchEngine backed by Meilisearch.
func NewMeiliEngine(client meilisearch.ServiceManager, indexUID string, svc *Service) *MeiliEngine {
	return &MeiliEngine{
		client:   client,
		indexUID: indexUID,
		svc:      svc,
	}
}

func (m *MeiliEngine) Search(ctx context.Context, q SearchQuery) ([]string, error) {
	// Trending sort: query Meilisearch for filtered results, then re-sort
	// by in-memory trending scores.
	if q.Order == TrendingOrder {
		return m.trendingSearch(ctx, q)
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
		return nil, fmt.Errorf("meili search error: %w", err)
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

	m.triggerResyncIfEmpty(q, len(ids))
	return ids, nil
}

// trendingSearch queries Meilisearch for filtered results, then re-sorts
// them by in-memory trending scores.
func (m *MeiliEngine) trendingSearch(ctx context.Context, q SearchQuery) ([]string, error) {
	filter := m.buildFilter(q)

	searchReq := &meilisearch.SearchRequest{
		Limit:  5000,
		Filter: filter,
	}

	index := m.client.Index(m.indexUID)
	result, err := index.SearchWithContext(ctx, q.Term, searchReq)
	if err != nil {
		return nil, fmt.Errorf("meili trending search error: %w", err)
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

	m.triggerResyncIfEmpty(q, len(ids))

	// Re-sort by trending scores from the in-memory service.
	scores := m.svc.trendingScores
	if scores != nil && len(ids) > 1 {
		slices.SortFunc(ids, func(a, b string) int {
			sa := scores[a]
			sb := scores[b]
			if sa > sb {
				return -1
			}
			if sa < sb {
				return 1
			}
			return 0
		})
	}

	return ids, nil
}

func (m *MeiliEngine) SearchSimilar(ctx context.Context, schematicID string, tags []string, limit int) ([]string, error) {
	// Build a search query from the tags.
	term := strings.Join(tags, " ")
	if term == "" {
		return nil, nil
	}

	filter := fmt.Sprintf(`id != "%s"`, escapeMeiliString(schematicID))

	searchReq := &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Filter: filter,
	}

	index := m.client.Index(m.indexUID)
	result, err := index.SearchWithContext(ctx, term, searchReq)
	if err != nil {
		slog.Error("meili SearchSimilar error", "error", err)
		return nil, err
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
	return m.svc.Suggest(q, limit)
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
		// BestMatch: use Meilisearch relevancy (no sort).
		return nil
	}
}

// triggerResyncIfEmpty kicks off a one-time background Meilisearch sync when a
// broad query (empty term, no filters) returns 0 hits but the in-memory index
// has documents. This covers the case where the pod started before Meilisearch
// was reachable or before auth was configured.
func (m *MeiliEngine) triggerResyncIfEmpty(q SearchQuery, hitCount int) {
	if hitCount > 0 || q.Term != "" || q.Category != "" && q.Category != "all" || len(q.Tags) > 0 {
		return
	}
	idx := m.svc.GetIndex()
	if len(idx) == 0 {
		return
	}
	m.resync.Do(func() {
		go func() {
			slog.Info("meili: 0-result broad query detected, triggering background resync", "index", m.indexUID, "docs", len(idx))
			docs := MapToMeiliDocuments(idx)
			if err := SyncMeiliIndex(m.client, m.indexUID, docs); err != nil {
				slog.Error("meili: background resync failed", "index", m.indexUID, "error", err)
			} else {
				slog.Info("meili: background resync complete", "index", m.indexUID, "docs", len(docs))
			}
		}()
	})
}

// escapeMeiliString escapes double quotes in filter values.
func escapeMeiliString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
