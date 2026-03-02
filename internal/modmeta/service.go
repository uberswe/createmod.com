package modmeta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ModMetadata holds enriched mod information from Modrinth/CurseForge.
type ModMetadata struct {
	Namespace    string
	DisplayName  string
	Description  string
	IconURL      string
	ModrinthSlug string
	ModrinthURL  string
	CurseForgeID string
	CurseForgeURL string
	SourceURL    string
}

// Service fetches and caches mod metadata from Modrinth and CurseForge.
type Service struct {
	curseForgeKey string
	httpClient    *http.Client
	stopChan      chan struct{}
}

// New creates a new mod metadata service.
func New(curseForgeKey string) *Service {
	return &Service{
		curseForgeKey: curseForgeKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// StartScheduler runs the enrichment scheduler in a background goroutine.
// It enriches mods on startup then every 6 hours.
func (s *Service) StartScheduler(app *pocketbase.PocketBase) {
	go func() {
		// Initial enrichment after a short delay to let boot complete
		time.Sleep(30 * time.Second)
		s.EnrichAll(app)

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.EnrichAll(app)
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
func (s *Service) GetMetadata(app *pocketbase.PocketBase, namespace string) *ModMetadata {
	records, err := app.FindRecordsByFilter(
		"mod_metadata",
		"namespace = {:ns}",
		"",
		1,
		0,
		dbx.Params{"ns": namespace},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	r := records[0]
	return &ModMetadata{
		Namespace:    r.GetString("namespace"),
		DisplayName:  r.GetString("display_name"),
		Description:  r.GetString("description"),
		IconURL:      r.GetString("icon_url"),
		ModrinthSlug: r.GetString("modrinth_slug"),
		ModrinthURL:  r.GetString("modrinth_url"),
		CurseForgeID: r.GetString("curseforge_id"),
		CurseForgeURL: r.GetString("curseforge_url"),
		SourceURL:    r.GetString("source_url"),
	}
}

// GetMetadataMap retrieves metadata for a list of namespaces as a map.
func (s *Service) GetMetadataMap(app *pocketbase.PocketBase, namespaces []string) map[string]*ModMetadata {
	result := make(map[string]*ModMetadata, len(namespaces))
	if len(namespaces) == 0 {
		return result
	}

	// Build filter with OR conditions
	for _, ns := range namespaces {
		meta := s.GetMetadata(app, ns)
		if meta != nil {
			result[ns] = meta
		}
	}
	return result
}

// EnrichMod fetches metadata for a single mod namespace from external APIs.
func (s *Service) EnrichMod(app *pocketbase.PocketBase, namespace string) error {
	// Check if manually set — skip if so
	existing, _ := app.FindRecordsByFilter(
		"mod_metadata",
		"namespace = {:ns}",
		"",
		1,
		0,
		dbx.Params{"ns": namespace},
	)
	if len(existing) > 0 && existing[0].GetBool("manually_set") {
		return nil
	}

	meta := &ModMetadata{Namespace: namespace}

	// 1. Try Modrinth direct lookup
	if err := s.tryModrinthDirect(namespace, meta); err != nil {
		app.Logger().Debug("modmeta: Modrinth direct lookup failed", "namespace", namespace, "error", err)

		// 2. Try Modrinth search fallback
		if err := s.tryModrinthSearch(namespace, meta); err != nil {
			app.Logger().Debug("modmeta: Modrinth search failed", "namespace", namespace, "error", err)
		}
	}

	// 3. Try CurseForge if we don't have a CurseForge URL yet
	if meta.CurseForgeURL == "" && s.curseForgeKey != "" {
		if err := s.tryCurseForgeSlug(namespace, meta); err != nil {
			app.Logger().Debug("modmeta: CurseForge slug lookup failed", "namespace", namespace, "error", err)

			// 4. Try CurseForge text search
			if err := s.tryCurseForgeSearch(namespace, meta); err != nil {
				app.Logger().Debug("modmeta: CurseForge search failed", "namespace", namespace, "error", err)
			}
		}
	}

	// If we got nothing useful, still save the record so we don't retry too often
	return s.upsertMetadata(app, existing, meta)
}

func (s *Service) EnrichAll(app *pocketbase.PocketBase) {
	app.Logger().Info("modmeta: starting enrichment run")

	// Get all unique mod namespaces from schematics
	modRows := []struct {
		ModName string `db:"mod_name"`
	}{}
	err := app.DB().NewQuery(`
		SELECT DISTINCT j.value AS mod_name
		FROM schematics, json_each(schematics.mods) AS j
		WHERE (schematics.deleted = '' OR schematics.deleted IS NULL)
		  AND schematics.moderated = 1
	`).All(&modRows)
	if err != nil {
		app.Logger().Error("modmeta: failed to query mod namespaces", "error", err)
		return
	}

	enriched := 0
	skipped := 0
	for _, row := range modRows {
		ns := strings.TrimSpace(row.ModName)
		if ns == "" {
			continue
		}

		// Check if we already have a recent record
		records, _ := app.FindRecordsByFilter(
			"mod_metadata",
			"namespace = {:ns}",
			"",
			1,
			0,
			dbx.Params{"ns": ns},
		)
		if len(records) > 0 {
			r := records[0]
			if r.GetBool("manually_set") {
				skipped++
				continue
			}
			lastFetched := r.GetDateTime("last_fetched").Time()
			if time.Since(lastFetched) < 7*24*time.Hour {
				skipped++
				continue
			}
		}

		if err := s.EnrichMod(app, ns); err != nil {
			app.Logger().Warn("modmeta: failed to enrich mod", "namespace", ns, "error", err)
		} else {
			enriched++
		}

		// Rate limit: 1 request/second between mods (each mod may make multiple API calls)
		time.Sleep(1 * time.Second)
	}

	app.Logger().Info("modmeta: enrichment run complete", "enriched", enriched, "skipped", skipped, "total", len(modRows))
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
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Summary string `json:"summary"`
	Logo  struct {
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

func (s *Service) upsertMetadata(app *pocketbase.PocketBase, existing []*core.Record, meta *ModMetadata) error {
	var record *core.Record
	if len(existing) > 0 {
		record = existing[0]
	} else {
		coll, err := app.FindCollectionByNameOrId("mod_metadata")
		if err != nil {
			return err
		}
		record = core.NewRecord(coll)
		record.Set("namespace", meta.Namespace)
	}

	if meta.DisplayName != "" {
		record.Set("display_name", meta.DisplayName)
	}
	if meta.Description != "" {
		record.Set("description", meta.Description)
	}
	if meta.IconURL != "" {
		record.Set("icon_url", meta.IconURL)
	}
	if meta.ModrinthSlug != "" {
		record.Set("modrinth_slug", meta.ModrinthSlug)
	}
	if meta.ModrinthURL != "" {
		record.Set("modrinth_url", meta.ModrinthURL)
	}
	if meta.CurseForgeID != "" {
		record.Set("curseforge_id", meta.CurseForgeID)
	}
	if meta.CurseForgeURL != "" {
		record.Set("curseforge_url", meta.CurseForgeURL)
	}
	if meta.SourceURL != "" {
		record.Set("source_url", meta.SourceURL)
	}
	record.Set("last_fetched", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))

	return app.Save(record)
}
