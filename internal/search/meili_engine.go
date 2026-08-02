package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// MeiliEngine implements SearchEngine using a Meilisearch index.
type MeiliEngine struct {
	client   meilisearch.ServiceManager
	indexUID string
	svc      *Service // for suggest, trending scores, and filter index

	// Resync trigger state. The actual resync is a River job enqueued via
	// resyncEnqueue so exactly ONE pod cluster-wide rebuilds the index —
	// before 2026-07-24 every replica ran its own in-process full-corpus
	// resync on the same 0-result signal, and the synchronized memory
	// balloon OOMKilled the whole deployment.
	resyncMu      sync.Mutex
	resyncEnqueue func()
	lastResync    time.Time

	// indexDocCount returns the live document count of the Meilisearch index.
	// The default queries Meilisearch; overridable in tests. Used to confirm
	// the index is ACTUALLY empty before enqueuing a rebuild — see
	// triggerResyncIfEmpty and the 2026-08-02 rebuild-storm incident.
	indexDocCount func(ctx context.Context) (int64, error)
}

// resyncTriggerCooldown rate-limits how often a single pod may enqueue a
// resync. The enqueue itself is further deduplicated cluster-wide by River
// unique opts, so this only keeps a pod from hammering inserts while
// Meilisearch is degraded.
const resyncTriggerCooldown = 5 * time.Minute

// SetResyncEnqueuer wires the callback used to enqueue an index rebuild job.
// Called once the River job worker is running.
func (m *MeiliEngine) SetResyncEnqueuer(fn func()) {
	m.resyncMu.Lock()
	defer m.resyncMu.Unlock()
	m.resyncEnqueue = fn
}

// NewMeiliEngine creates a SearchEngine backed by Meilisearch.
func NewMeiliEngine(client meilisearch.ServiceManager, indexUID string, svc *Service) *MeiliEngine {
	m := &MeiliEngine{
		client:   client,
		indexUID: indexUID,
		svc:      svc,
	}
	m.indexDocCount = func(ctx context.Context) (int64, error) {
		stats, err := client.Index(indexUID).GetStatsWithContext(ctx, nil)
		if err != nil {
			return 0, err
		}
		return stats.NumberOfDocuments, nil
	}
	return m
}

func (m *MeiliEngine) Search(ctx context.Context, q SearchQuery) ([]string, error) {
	filter := m.buildFilter(q)
	sort := m.buildSort(q.Order)

	searchReq := &meilisearch.SearchRequest{
		Limit:  5000,
		Filter: filter,
		Sort:   sort,
		// Only the IDs are used — without this Meilisearch returns 5000 FULL
		// documents (descriptions included) per broad query, which at ~30
		// req/s of concurrent decodes was a large share of the pods' memory
		// pressure in the 2026-07-24 OOM cascade.
		AttributesToRetrieve: []string{"id"},
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

func (m *MeiliEngine) SearchSimilar(ctx context.Context, schematicID string, tags []string, limit int) ([]string, error) {
	// Build a search query from the tags.
	term := strings.Join(tags, " ")
	if term == "" {
		return nil, nil
	}

	filter := fmt.Sprintf(`id != "%s"`, escapeMeiliString(schematicID))

	searchReq := &meilisearch.SearchRequest{
		Limit:                int64(limit),
		Filter:               filter,
		AttributesToRetrieve: []string{"id"},
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

	if len(q.CreateVersions) > 0 {
		var cvParts []string
		for _, cv := range q.CreateVersions {
			cvParts = append(cvParts, fmt.Sprintf(`"%s"`, escapeMeiliString(cv)))
		}
		parts = append(parts, fmt.Sprintf(`create_version IN [%s]`, strings.Join(cvParts, ", ")))
	} else if q.CreateVersion != "" && q.CreateVersion != "all" {
		parts = append(parts, fmt.Sprintf(`create_version = "%s"`, escapeMeiliString(q.CreateVersion)))
	}

	// Dimension and block count range filters
	// Upper-bound (max) size filters apply ONLY when the slider sits below its
	// cap. The filter form always submits the size sliders, and their maxima
	// (sliderMaxDimX/Y/Z, sliderMaxBlockCount) are hardcoded BELOW the largest
	// builds in the corpus. Treating "slider at its max" as an upper bound
	// silently hid every schematic bigger than the cap — e.g. Sea Laboratory
	// (215x215x63, ~280k blocks) never appeared in a normal search (#1604). At
	// the cap we apply no upper bound so large builds still match; the min side
	// is naturally 0 = unbounded already.
	if q.MinBlockCount > 0 {
		parts = append(parts, fmt.Sprintf("block_count >= %d", q.MinBlockCount))
	}
	if q.MaxBlockCount > 0 && q.MaxBlockCount < sliderMaxBlockCount {
		parts = append(parts, fmt.Sprintf("block_count <= %d", q.MaxBlockCount))
	}
	if q.MinHorizontal > 0 {
		parts = append(parts, fmt.Sprintf("(dim_x >= %d OR dim_z >= %d)", q.MinHorizontal, q.MinHorizontal))
	}
	if q.MaxHorizontal > 0 && q.MaxHorizontal < sliderMaxDimX {
		parts = append(parts, fmt.Sprintf("(dim_x <= %d AND dim_z <= %d)", q.MaxHorizontal, q.MaxHorizontal))
	}
	if q.MinDimX > 0 {
		parts = append(parts, fmt.Sprintf("dim_x >= %d", q.MinDimX))
	}
	if q.MaxDimX > 0 && q.MaxDimX < sliderMaxDimX {
		parts = append(parts, fmt.Sprintf("dim_x <= %d", q.MaxDimX))
	}
	if q.MinDimY > 0 {
		parts = append(parts, fmt.Sprintf("dim_y >= %d", q.MinDimY))
	}
	if q.MaxDimY > 0 && q.MaxDimY < sliderMaxDimY {
		parts = append(parts, fmt.Sprintf("dim_y <= %d", q.MaxDimY))
	}
	if q.MinDimZ > 0 {
		parts = append(parts, fmt.Sprintf("dim_z >= %d", q.MinDimZ))
	}
	if q.MaxDimZ > 0 && q.MaxDimZ < sliderMaxDimZ {
		parts = append(parts, fmt.Sprintf("dim_z <= %d", q.MaxDimZ))
	}

	// Mod filter: require all selected mods to be present
	for _, mod := range q.Mods {
		parts = append(parts, fmt.Sprintf(`mod_names = "%s"`, escapeMeiliString(mod)))
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
	case TrendingOrder:
		return []string{"trending_score:desc"}
	default:
		// BestMatch: use Meilisearch relevancy (no sort).
		return nil
	}
}

// triggerResyncIfEmpty enqueues a search index rebuild only when Meilisearch
// has genuinely lost its index. A broad query (empty term, no filters) that
// returns 0 hits is the initial signal, but it is NOT sufficient on its own —
// under load Meilisearch can transiently return 0 while the index is fully
// populated. Before enqueuing, this confirms the live index is actually empty
// (doc count 0). Without that check, a rebuild slows Meilisearch, which makes
// broad queries return 0, which triggers more rebuilds: the 2026-08-02 storm.
// The rebuild runs as a River job (deduplicated cluster-wide), NOT in-process:
// one pod does the work no matter how many replicas observe the condition.
func (m *MeiliEngine) triggerResyncIfEmpty(q SearchQuery, hitCount int) {
	if hitCount > 0 || q.Term != "" || q.Category != "" && q.Category != "all" || len(q.Tags) > 0 {
		return
	}
	if len(m.svc.GetIndex()) == 0 {
		return
	}

	// Cheap per-pod cooldown gate. Consume the cooldown even when the rebuild
	// is skipped below, so a degraded Meilisearch can't make a pod re-check on
	// every broad query.
	m.resyncMu.Lock()
	if m.resyncEnqueue == nil || m.indexDocCount == nil || time.Since(m.lastResync) < resyncTriggerCooldown {
		m.resyncMu.Unlock()
		return
	}
	m.lastResync = time.Now()
	enqueue := m.resyncEnqueue
	docCount := m.indexDocCount
	m.resyncMu.Unlock()

	// A broad query returning 0 hits does NOT prove the index is stale: under
	// load Meilisearch can transiently return 0 while it actually holds the
	// full corpus. Only rebuild when the index is GENUINELY empty. Rebuilding
	// on a transient 0-result adds indexing load that causes more transient
	// 0-results, which triggers more rebuilds — the 2026-08-02 rebuild storm.
	// If the count can't be confirmed, do nothing (fail safe: no rebuild).
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	count, err := docCount(ctx)
	if err != nil {
		slog.Warn("meili: index stats check failed, skipping rebuild", "index", m.indexUID, "error", err)
		return
	}
	if count > 0 {
		return // index has documents — the 0-result was transient, not stale
	}

	slog.Info("meili: index confirmed empty, enqueueing rebuild", "index", m.indexUID)
	enqueue()
}

// escapeMeiliString escapes double quotes in filter values.
func escapeMeiliString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
