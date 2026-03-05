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

	"createmod/internal/store"
)

// ModMetadata holds enriched mod information from Modrinth/CurseForge.
type ModMetadata struct {
	Namespace     string
	DisplayName   string
	Description   string
	IconURL       string
	ModrinthSlug  string
	ModrinthURL   string
	CurseForgeID  string
	CurseForgeURL string
	SourceURL     string
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
			Timeout: 15 * time.Second,
		},
		stopChan: make(chan struct{}),
		appStore: appStore,
	}
}

// StartScheduler runs the enrichment scheduler in a background goroutine.
// It enriches mods on startup then every 6 hours.
func (s *Service) StartScheduler() {
	go func() {
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

// EnrichMod fetches metadata for a single mod namespace from external APIs.
func (s *Service) EnrichMod(namespace string) error {
	ctx := context.Background()

	// Check if manually set - skip if so
	existing, err := s.appStore.ModMetadata.GetByNamespace(ctx, namespace)
	if err == nil && existing != nil && existing.ManuallySet {
		return nil
	}

	meta := &ModMetadata{Namespace: namespace}

	// 1. Try Modrinth direct lookup
	if err := s.tryModrinthDirect(namespace, meta); err != nil {
		slog.Debug("modmeta: Modrinth direct lookup failed", "namespace", namespace, "error", err)

		// 2. Try Modrinth search fallback
		if err := s.tryModrinthSearch(namespace, meta); err != nil {
			slog.Debug("modmeta: Modrinth search failed", "namespace", namespace, "error", err)
		}
	}

	// 3. Try CurseForge if we don't have a CurseForge URL yet
	if meta.CurseForgeURL == "" && s.curseForgeKey != "" {
		if err := s.tryCurseForgeSlug(namespace, meta); err != nil {
			slog.Debug("modmeta: CurseForge slug lookup failed", "namespace", namespace, "error", err)

			// 4. Try CurseForge text search
			if err := s.tryCurseForgeSearch(namespace, meta); err != nil {
				slog.Debug("modmeta: CurseForge search failed", "namespace", namespace, "error", err)
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

func (s *Service) tryModrinthSearch(namespace string, meta *ModMetadata) error {
	searchURL := fmt.Sprintf(
		"https://api.modrinth.com/v2/search?query=%s&facets=[[\"project_type:mod\"]]&limit=5",
		url.QueryEscape(namespace),
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

	// Score results: exact slug match > contains namespace > first result
	var best *modrinthSearchHit
	for i := range result.Hits {
		hit := &result.Hits[i]
		if strings.EqualFold(hit.Slug, namespace) {
			best = hit
			break
		}
	}
	if best == nil {
		nsLower := strings.ToLower(namespace)
		for i := range result.Hits {
			hit := &result.Hits[i]
			if strings.Contains(strings.ToLower(hit.Slug), nsLower) ||
				strings.Contains(strings.ToLower(hit.Title), nsLower) {
				best = hit
				break
			}
		}
	}
	if best == nil {
		best = &result.Hits[0]
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
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Summary string `json:"summary"`
	Logo    struct {
		URL string `json:"url"`
	} `json:"logo"`
	Links struct {
		SourceURL string `json:"sourceUrl"`
	} `json:"links"`
}

func (s *Service) tryCurseForgeSlug(namespace string, meta *ModMetadata) error {
	cfURL := fmt.Sprintf(
		"https://api.curseforge.com/v1/mods/search?gameId=432&slug=%s&classId=6",
		url.QueryEscape(namespace),
	)
	return s.doCurseForgeSearch(cfURL, meta)
}

func (s *Service) tryCurseForgeSearch(namespace string, meta *ModMetadata) error {
	cfURL := fmt.Sprintf(
		"https://api.curseforge.com/v1/mods/search?gameId=432&searchFilter=%s&classId=6&pageSize=5",
		url.QueryEscape(namespace),
	)
	return s.doCurseForgeSearch(cfURL, meta)
}

func (s *Service) doCurseForgeSearch(cfURL string, meta *ModMetadata) error {
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

	project := result.Data[0]
	meta.CurseForgeID = fmt.Sprintf("%d", project.ID)
	meta.CurseForgeURL = fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%s", project.Slug)

	if meta.DisplayName == "" {
		meta.DisplayName = project.Name
	}
	if meta.Description == "" {
		meta.Description = project.Summary
	}
	if meta.IconURL == "" {
		meta.IconURL = project.Logo.URL
	}
	if meta.SourceURL == "" && project.Links.SourceURL != "" {
		meta.SourceURL = project.Links.SourceURL
	}
	return nil
}

// --- Database ---

func (s *Service) upsertMetadata(meta *ModMetadata) error {
	ctx := context.Background()
	return s.appStore.ModMetadata.Upsert(ctx, &store.ModMetadata{
		Namespace:     meta.Namespace,
		DisplayName:   meta.DisplayName,
		Description:   meta.Description,
		IconURL:       meta.IconURL,
		ModrinthSlug:  meta.ModrinthSlug,
		ModrinthURL:   meta.ModrinthURL,
		CurseforgeID:  meta.CurseForgeID,
		CurseforgeURL: meta.CurseForgeURL,
		SourceURL:     meta.SourceURL,
	})
}
