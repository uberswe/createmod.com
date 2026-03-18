package modmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"createmod/internal/slowlog"
	"createmod/internal/store"
)

// ModMetadata holds enriched mod information from Modrinth/CurseForge.
type ModMetadata struct {
	Namespace          string
	DisplayName        string
	Description        string
	IconURL            string
	ModrinthSlug       string
	ModrinthURL        string
	CurseForgeID       string
	CurseForgeURL      string
	SourceURL          string
	BlocksitemsMatched bool
}

// Service fetches and caches mod metadata from Modrinth and CurseForge.
type Service struct {
	curseForgeKey string
	httpClient    *http.Client
	stopChan      chan struct{}
	appStore      *store.Store
}

// New creates a new mod metadata service.
func New(curseForgeKey string, appStore *store.Store) *Service {
	return &Service{
		curseForgeKey: curseForgeKey,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: &slowlog.SlowHTTPTransport{Base: http.DefaultTransport, Subsystem: "modmeta"},
		},
		stopChan: make(chan struct{}),
		appStore: appStore,
	}
}

// manualOverrides maps mod namespaces to their correct CurseForge URLs.
// Entries here are upserted with ManuallySet=true on startup so the
// automatic enrichment scheduler will never overwrite them.
var manualOverrides = map[string]string{
	"dndecor":   "https://www.curseforge.com/minecraft/mc-mods/create-design-n-decor",
	"tfmg":      "https://www.curseforge.com/minecraft/mc-mods/create-industry",
	"create_dd": "https://www.curseforge.com/minecraft/mc-mods/create-dreams-desires",
}

// applyManualOverrides upserts hardcoded CurseForge URL corrections into the
// database with ManuallySet=true so the scheduler won't overwrite them.
func (s *Service) applyManualOverrides() {
	ctx := context.Background()
	for ns, cfURL := range manualOverrides {
		existing, err := s.appStore.ModMetadata.GetByNamespace(ctx, ns)
		if err == nil && existing != nil && existing.CurseforgeURL == cfURL && existing.ManuallySet {
			continue // already correct
		}
		meta := &store.ModMetadata{
			Namespace:     ns,
			ManuallySet:   true,
			CurseforgeURL: cfURL,
		}
		// Preserve existing fields if the record already exists
		if existing != nil {
			meta.DisplayName = existing.DisplayName
			meta.Description = existing.Description
			meta.IconURL = existing.IconURL
			meta.ModrinthSlug = existing.ModrinthSlug
			meta.ModrinthURL = existing.ModrinthURL
			meta.CurseforgeID = existing.CurseforgeID
			meta.SourceURL = existing.SourceURL
			meta.BlocksitemsMatched = existing.BlocksitemsMatched
			meta.LastFetched = existing.LastFetched
		}
		if err := s.appStore.ModMetadata.Upsert(ctx, meta); err != nil {
			slog.Error("modmeta: failed to apply manual override", "namespace", ns, "error", err)
		} else {
			slog.Info("modmeta: applied manual override", "namespace", ns, "curseforge_url", cfURL)
		}
	}
}

// StartScheduler runs the enrichment scheduler in a background goroutine.
// It enriches mods on startup then every 6 hours.
func (s *Service) StartScheduler() {
	go func() {
		// Apply hardcoded overrides immediately
		s.applyManualOverrides()

		// Initial enrichment after a short delay to let boot complete
		time.Sleep(30 * time.Second)
		s.EnrichAll()

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.EnrichAll()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// Stop signals the scheduler to stop.
func (s *Service) Stop() {
	close(s.stopChan)
}

// GetMetadata retrieves cached metadata for a namespace from the database.
func (s *Service) GetMetadata(namespace string) *ModMetadata {
	ctx := context.Background()
	m, err := s.appStore.ModMetadata.GetByNamespace(ctx, namespace)
	if err != nil || m == nil {
		return nil
	}
	return &ModMetadata{
		Namespace:     m.Namespace,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		IconURL:       m.IconURL,
		ModrinthSlug:  m.ModrinthSlug,
		ModrinthURL:   m.ModrinthURL,
		CurseForgeID:  m.CurseforgeID,
		CurseForgeURL: m.CurseforgeURL,
		SourceURL:     m.SourceURL,
	}
}

// GetMetadataMap retrieves metadata for a list of namespaces as a map.
func (s *Service) GetMetadataMap(namespaces []string) map[string]*ModMetadata {
	result := make(map[string]*ModMetadata, len(namespaces))
	if len(namespaces) == 0 {
		return result
	}

	for _, ns := range namespaces {
		meta := s.GetMetadata(ns)
		if meta != nil {
			result[ns] = meta
		}
	}
	return result
}

// expandNamespace takes a raw namespace (e.g. "createbigcannons", "design_decor")
// and produces a list of search query variants, ordered most-specific first.
// It uses gse dictionary-based word segmentation with Norvig's English corpus
// to split concatenated words (e.g. "createbigcannons" → "create big cannons").
func expandNamespace(namespace string) []string {
	// Replace underscores with spaces
	spaced := strings.ReplaceAll(namespace, "_", " ")

	// Split camelCase boundaries first (for "CreateBigCannons" → "Create Big Cannons")
	spaced = splitCamelCase(spaced)

	// Use gse to segment any remaining concatenated words
	words := seg.Cut(strings.ToLower(spaced))

	// Filter empty/whitespace tokens
	var filtered []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			filtered = append(filtered, w)
		}
	}
	segmented := strings.Join(filtered, " ")

	seen := make(map[string]bool)
	var variants []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			variants = append(variants, v)
		}
	}

	parts := strings.Fields(segmented)
	if len(parts) > 1 && strings.EqualFold(parts[0], "create") {
		// Already starts with "create" — add segmented then "Create: rest" variant
		add(segmented)
		rest := strings.Join(parts[1:], " ")
		add("Create: " + rest)
	} else if !strings.EqualFold(segmented, "create") {
		// This is a Create mod platform — try "create ..." first so
		// searches find Create-ecosystem mods (e.g. "design decor"
		// → "create design decor" finds "Create: Design n' Decor")
		add("create " + segmented)
		add(segmented)
	} else {
		add(segmented)
	}

	// Add the original namespace as a fallback
	add(namespace)

	return variants
}

// nameSimilarity computes a word-level Dice coefficient between two strings.
// It returns a value between 0.0 (no overlap) and 1.0 (identical words).
// Both inputs are lowercased and split on whitespace before comparison.
// Punctuation like ":" and "'" is stripped so "Create: Design n' Decor"
// matches "create design decor" well.
func nameSimilarity(reference, candidate string) float64 {
	normalize := func(s string) []string {
		s = strings.ToLower(s)
		s = strings.NewReplacer(":", "", "'", "", "'", "", "-", " ", "_", " ").Replace(s)
		fields := strings.Fields(s)
		// Filter short noise words like "n" from "n' Decor"
		var out []string
		for _, f := range fields {
			if len(f) > 1 {
				out = append(out, f)
			}
		}
		return out
	}

	refWords := normalize(reference)
	candWords := normalize(candidate)

	if len(refWords) == 0 || len(candWords) == 0 {
		return 0
	}

	refSet := make(map[string]bool, len(refWords))
	for _, w := range refWords {
		refSet[w] = true
	}

	matches := 0
	for _, w := range candWords {
		if refSet[w] {
			matches++
		}
	}

	return 2.0 * float64(matches) / float64(len(refWords)+len(candWords))
}

// splitCamelCase splits a string on camelCase boundaries and returns a
// space-separated result. This is applied before gse segmentation so that
// camelCase inputs like "CreateBigCannons" become "Create Big Cannons".
func splitCamelCase(s string) string {
	parts := strings.Fields(s)
	var result []string
	for _, part := range parts {
		var current []rune
		runes := []rune(part)
		for i, r := range runes {
			if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				if len(current) > 0 {
					result = append(result, string(current))
				}
				current = []rune{r}
			} else {
				current = append(current, r)
			}
		}
		if len(current) > 0 {
			result = append(result, string(current))
		}
	}
	return strings.Join(result, " ")
}

// EnrichMod fetches metadata for a single mod namespace from external APIs.
func (s *Service) EnrichMod(namespace string) error {
	ctx := context.Background()

	// Check if manually set - skip if so
	existing, err := s.appStore.ModMetadata.GetByNamespace(ctx, namespace)
	if err == nil && existing != nil && existing.ManuallySet {
		return nil
	}

	meta := &ModMetadata{Namespace: namespace}

	// 1. Try BlocksItems lookup to get a proper display name
	searchName := namespace // default search term for Modrinth/CurseForge fallbacks
	blocksitemsMatched := false
	if biName, matched := s.tryBlocksItemsLookup(namespace); matched && biName != "" {
		meta.DisplayName = biName
		meta.BlocksitemsMatched = true
		blocksitemsMatched = true
		slog.Debug("modmeta: BlocksItems matched", "namespace", namespace, "name", biName)
		// Use the proper name for subsequent search queries if it differs
		if !strings.EqualFold(biName, namespace) {
			searchName = biName
		}
	}

	// Generate search variants from the search name
	searchVariants := expandNamespace(searchName)

	// The reference name is used for similarity scoring against search results.
	// If BlocksItems matched, use that display name; otherwise use searchName
	// (which is the best available name from the namespace).
	referenceName := searchName
	if blocksitemsMatched && meta.DisplayName != "" {
		referenceName = meta.DisplayName
	}

	// 2. Try Modrinth direct lookup (uses namespace as project slug)
	if err := s.tryModrinthDirect(namespace, meta); err != nil {
		slog.Debug("modmeta: Modrinth direct lookup failed", "namespace", namespace, "error", err)

		// 3. Try Modrinth search fallback with each variant
		for _, variant := range searchVariants {
			if err := s.tryModrinthSearch(variant, namespace, referenceName, blocksitemsMatched, meta); err != nil {
				slog.Debug("modmeta: Modrinth search failed", "namespace", namespace, "variant", variant, "error", err)
			} else {
				break
			}
		}
	}

	// 4. Try CurseForge if we don't have a CurseForge URL yet
	if meta.CurseForgeURL == "" && s.curseForgeKey != "" {
		// Try slug lookups first: raw namespace, then hyphenated variants
		slugFound := false
		if err := s.tryCurseForgeSlug(namespace, referenceName, meta); err == nil {
			slugFound = true
		} else {
			slog.Debug("modmeta: CurseForge slug lookup failed", "namespace", namespace, "error", err)
			// Try hyphenated slug variants from search terms
			// e.g. "Create Deco" → "create-deco", "design decor" → "design-decor"
			for _, variant := range searchVariants {
				slug := strings.ToLower(strings.ReplaceAll(variant, " ", "-"))
				if slug != namespace {
					if err := s.tryCurseForgeSlug(slug, referenceName, meta); err == nil {
						slugFound = true
						break
					}
				}
			}
		}

		if !slugFound {
			// 5. Try CurseForge text search with each variant
			for _, variant := range searchVariants {
				if err := s.tryCurseForgeSearch(variant, namespace, referenceName, blocksitemsMatched, meta); err != nil {
					slog.Debug("modmeta: CurseForge search failed", "namespace", namespace, "variant", variant, "error", err)
				} else {
					break
				}
			}
		}
	}

	// If we got nothing useful, still save the record so we don't retry too often
	return s.upsertMetadata(meta)
}

func (s *Service) EnrichAll() {
	slog.Info("modmeta: starting enrichment run")

	ctx := context.Background()

	// Get all unique mod namespaces from schematics
	modCounts, err := s.appStore.Schematics.ListModCounts(ctx)
	if err != nil {
		slog.Error("modmeta: failed to query mod namespaces", "error", err)
		return
	}

	enriched := 0
	skipped := 0
	for _, mc := range modCounts {
		ns := strings.TrimSpace(mc.ModName)
		if ns == "" {
			continue
		}

		// Check if we already have a recent record
		existing, err := s.appStore.ModMetadata.GetByNamespace(ctx, ns)
		if err == nil && existing != nil {
			if existing.ManuallySet {
				skipped++
				continue
			}
			if existing.LastFetched != nil && time.Since(*existing.LastFetched) < 7*24*time.Hour {
				skipped++
				continue
			}
		}

		if err := s.EnrichMod(ns); err != nil {
			slog.Warn("modmeta: failed to enrich mod", "namespace", ns, "error", err)
		} else {
			enriched++
		}

		// Rate limit: 1 request/second between mods (each mod may make multiple API calls)
		time.Sleep(1 * time.Second)
	}

	slog.Info("modmeta: enrichment run complete", "enriched", enriched, "skipped", skipped, "total", len(modCounts))
}

// --- BlocksItems API ---

type blocksitemsLookupResponse struct {
	Data    []blocksitemsModEntry `json:"data"`
	Matched bool                  `json:"matched"`
	Status  string                `json:"status"`
}

type blocksitemsModEntry struct {
	ModID string `json:"mod_id"`
	Name  string `json:"name"`
}

// tryBlocksItemsLookup queries BlocksItems.com to resolve namespace → display name.
// Returns the display name if matched, empty string otherwise.
func (s *Service) tryBlocksItemsLookup(namespace string) (string, bool) {
	reqURL := fmt.Sprintf("https://blocksitems.com/api/v1/mods/lookup/%s", url.PathEscape(namespace))
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Debug("modmeta: BlocksItems request build failed", "namespace", namespace, "error", err)
		return "", false
	}
	req.Header.Set("User-Agent", "CreateMod.com/1.0 (hello@createmod.com)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.Debug("modmeta: BlocksItems request failed", "namespace", namespace, "error", err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("modmeta: BlocksItems returned non-OK status", "namespace", namespace, "status", resp.StatusCode)
		return "", false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Debug("modmeta: BlocksItems read body failed", "namespace", namespace, "error", err)
		return "", false
	}

	var result blocksitemsLookupResponse
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Debug("modmeta: BlocksItems JSON parse failed", "namespace", namespace, "error", err)
		return "", false
	}

	if !result.Matched || len(result.Data) == 0 {
		return "", false
	}

	return result.Data[0].Name, true
}

// --- Modrinth API ---

type modrinthProject struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
	SourceURL   string `json:"source_url"`
}

type modrinthSearchResult struct {
	Hits []modrinthSearchHit `json:"hits"`
}

type modrinthSearchHit struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
	Downloads   int    `json:"downloads"`
}

func (s *Service) tryModrinthDirect(namespace string, meta *ModMetadata) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.modrinth.com/v2/project/%s", url.PathEscape(namespace)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CreateMod.com/1.0 (hello@createmod.com)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found on Modrinth")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Modrinth returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var project modrinthProject
	if err := json.Unmarshal(body, &project); err != nil {
		return err
	}

	meta.ModrinthSlug = project.Slug
	meta.ModrinthURL = fmt.Sprintf("https://modrinth.com/mod/%s", project.Slug)
	if meta.DisplayName == "" {
		meta.DisplayName = project.Title
	}
	if meta.Description == "" {
		meta.Description = project.Description
	}
	if meta.IconURL == "" {
		meta.IconURL = project.IconURL
	}
	if meta.SourceURL == "" {
		meta.SourceURL = project.SourceURL
	}
	return nil
}

func (s *Service) tryModrinthSearch(searchName string, namespace string, referenceName string, blocksitemsMatched bool, meta *ModMetadata) error {
	searchURL := fmt.Sprintf(
		"https://api.modrinth.com/v2/search?query=%s&facets=[[\"project_type:mod\"]]&limit=5",
		url.QueryEscape(searchName),
	)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CreateMod.com/1.0 (hello@createmod.com)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Modrinth search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result modrinthSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Hits) == 0 {
		return fmt.Errorf("no Modrinth search results")
	}

	// Score each hit using name similarity as the primary factor.
	// The referenceName is the BlocksItems display name (if available)
	// or a search variant derived from the namespace.
	var best *modrinthSearchHit
	bestScore := -1.0

	for i := range result.Hits {
		hit := &result.Hits[i]

		// Apply download threshold for search-fallback matches
		if !blocksitemsMatched && hit.Downloads < 10000 {
			continue
		}

		// Primary: name similarity (0.0–1.0) scaled to dominate scoring
		sim := nameSimilarity(referenceName, hit.Title)
		score := sim * 10000

		// Exact slug match is a strong signal
		if strings.EqualFold(hit.Slug, namespace) {
			score += 5000
		}

		// Tiebreaker: log-scale downloads so popularity helps but can't
		// overwhelm a good name match (a 1M-download mod gets ~138 points)
		if hit.Downloads > 0 {
			dl := float64(hit.Downloads)
			// ln(1M) ≈ 13.8, scale by 10 → 138 points
			lnDl := 0.0
			for dl > 1 {
				lnDl++
				dl /= 2.718281828
			}
			score += lnDl * 10
		}

		if best == nil || score > bestScore {
			best = hit
			bestScore = score
		}
	}

	if best == nil {
		return fmt.Errorf("no Modrinth search results above threshold")
	}

	meta.ModrinthSlug = best.Slug
	meta.ModrinthURL = fmt.Sprintf("https://modrinth.com/mod/%s", best.Slug)
	if meta.DisplayName == "" {
		meta.DisplayName = best.Title
	}
	if meta.Description == "" {
		meta.Description = best.Description
	}
	if meta.IconURL == "" {
		meta.IconURL = best.IconURL
	}
	return nil
}

// --- CurseForge API ---

type curseForgeSearchResponse struct {
	Data []curseForgeProject `json:"data"`
}

type curseForgeProject struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Summary       string `json:"summary"`
	DownloadCount int    `json:"downloadCount"`
	Logo          struct {
		URL string `json:"url"`
	} `json:"logo"`
	Links struct {
		SourceURL string `json:"sourceUrl"`
	} `json:"links"`
}

func (s *Service) tryCurseForgeSlug(namespace string, referenceName string, meta *ModMetadata) error {
	cfURL := fmt.Sprintf(
		"https://api.curseforge.com/v1/mods/search?gameId=432&slug=%s&classId=6",
		url.QueryEscape(namespace),
	)
	// Slug lookup is a direct match, so pass blocksitemsMatched=true to skip download threshold
	return s.doCurseForgeSearch(cfURL, namespace, referenceName, true, meta)
}

func (s *Service) tryCurseForgeSearch(searchName string, namespace string, referenceName string, blocksitemsMatched bool, meta *ModMetadata) error {
	cfURL := fmt.Sprintf(
		"https://api.curseforge.com/v1/mods/search?gameId=432&searchFilter=%s&classId=6&pageSize=5",
		url.QueryEscape(searchName),
	)
	return s.doCurseForgeSearch(cfURL, namespace, referenceName, blocksitemsMatched, meta)
}

func (s *Service) doCurseForgeSearch(cfURL string, namespace string, referenceName string, blocksitemsMatched bool, meta *ModMetadata) error {
	req, err := http.NewRequest("GET", cfURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", s.curseForgeKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CurseForge returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result curseForgeSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		return fmt.Errorf("no CurseForge results")
	}

	// Score each result using name similarity as the primary factor.
	var best *curseForgeProject
	bestScore := -1.0

	for i := range result.Data {
		project := &result.Data[i]

		// Apply download threshold for search-fallback matches
		if !blocksitemsMatched && project.DownloadCount < 10000 {
			continue
		}

		// Primary: name similarity (0.0–1.0) scaled to dominate scoring
		sim := nameSimilarity(referenceName, project.Name)
		score := sim * 10000

		// Exact slug match is a strong signal
		if strings.EqualFold(project.Slug, namespace) {
			score += 5000
		}

		// Tiebreaker: log-scale downloads
		if project.DownloadCount > 0 {
			dl := float64(project.DownloadCount)
			lnDl := 0.0
			for dl > 1 {
				lnDl++
				dl /= 2.718281828
			}
			score += lnDl * 10
		}

		if best == nil || score > bestScore {
			best = project
			bestScore = score
		}
	}

	if best == nil {
		return fmt.Errorf("no CurseForge results above threshold")
	}

	meta.CurseForgeID = fmt.Sprintf("%d", best.ID)
	meta.CurseForgeURL = fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%s", best.Slug)

	if meta.DisplayName == "" {
		meta.DisplayName = best.Name
	}
	if meta.Description == "" {
		meta.Description = best.Summary
	}
	if meta.IconURL == "" {
		meta.IconURL = best.Logo.URL
	}
	if meta.SourceURL == "" && best.Links.SourceURL != "" {
		meta.SourceURL = best.Links.SourceURL
	}
	return nil
}

// --- Database ---

func (s *Service) upsertMetadata(meta *ModMetadata) error {
	ctx := context.Background()
	return s.appStore.ModMetadata.Upsert(ctx, &store.ModMetadata{
		Namespace:          meta.Namespace,
		DisplayName:        meta.DisplayName,
		Description:        meta.Description,
		IconURL:            meta.IconURL,
		ModrinthSlug:       meta.ModrinthSlug,
		ModrinthURL:        meta.ModrinthURL,
		CurseforgeID:       meta.CurseForgeID,
		CurseforgeURL:      meta.CurseForgeURL,
		SourceURL:          meta.SourceURL,
		BlocksitemsMatched: meta.BlocksitemsMatched,
	})
}
