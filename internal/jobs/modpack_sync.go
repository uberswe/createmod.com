package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"createmod/internal/store"

	"github.com/riverqueue/river"
)

type ModpackSyncArgs struct{}

func (ModpackSyncArgs) Kind() string { return "modpack_sync" }

type ModpackSyncWorker struct {
	river.WorkerDefaults[ModpackSyncArgs]
	deps Deps
}

type modrinthSearchResponse struct {
	Hits      []modrinthHit `json:"hits"`
	TotalHits int           `json:"total_hits"`
	Offset    int           `json:"offset"`
	Limit     int           `json:"limit"`
}

type modrinthHit struct {
	ProjectID   string `json:"project_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
	Downloads   int    `json:"downloads"`
}

func (w *ModpackSyncWorker) Work(ctx context.Context, job *river.Job[ModpackSyncArgs]) error {
	slog.Info("modpack sync started")
	if w.deps.Store == nil {
		return nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	offset := 0
	limit := 100
	total := 0

	for {
		url := fmt.Sprintf(
			"https://api.modrinth.com/v2/search?facets=[[\"categories:create\"]]&project_type=modpack&limit=%d&offset=%d",
			limit, offset,
		)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("modpack sync: create request: %w", err)
		}
		req.Header.Set("User-Agent", "createmod.com/1.0 (markus@tenghamn.com)")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("modpack sync: request failed: %w", err)
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("modpack sync: API returned %d: %s", resp.StatusCode, string(body))
		}

		var result modrinthSearchResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("modpack sync: parse response: %w", err)
		}

		for _, hit := range result.Hits {
			modpack := &store.Modpack{
				ModrinthID:  hit.ProjectID,
				Slug:        hit.Slug,
				Name:        hit.Title,
				Description: hit.Description,
				IconURL:     hit.IconURL,
				ModrinthURL: fmt.Sprintf("https://modrinth.com/modpack/%s", hit.Slug),
				Downloads:   hit.Downloads,
			}
			if err := w.deps.Store.Modpacks.Upsert(ctx, modpack); err != nil {
				slog.Warn("modpack sync: upsert failed", "slug", hit.Slug, "err", err)
			} else {
				total++
			}
		}

		if offset+limit >= result.TotalHits {
			break
		}
		offset += limit
	}

	slog.Info("modpack sync completed", "upserted", total)
	return nil
}
