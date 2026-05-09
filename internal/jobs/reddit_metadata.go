package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/riverqueue/river"
)

type RedditMetadataArgs struct{}

func (RedditMetadataArgs) Kind() string { return "reddit_metadata_refresh" }

type RedditMetadataWorker struct {
	river.WorkerDefaults[RedditMetadataArgs]
	deps Deps
}

func (w *RedditMetadataWorker) Work(ctx context.Context, job *river.Job[RedditMetadataArgs]) error {
	slog.Info("reddit metadata refresh started")
	if w.deps.Store == nil {
		return nil
	}

	stale, err := w.deps.Store.RedditLinks.ListStale(ctx, 50)
	if err != nil {
		return err
	}

	if len(stale) == 0 {
		slog.Info("reddit metadata: no stale links")
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	updated := 0

	for _, link := range stale {
		u, err := url.Parse(link.RedditURL)
		if err != nil {
			slog.Warn("reddit metadata: invalid URL", "id", link.ID, "url", link.RedditURL)
			continue
		}

		jsonURL := fmt.Sprintf("https://www.reddit.com%s.json", u.Path)
		req, err := http.NewRequestWithContext(ctx, "GET", jsonURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "createmod.com:v1.0 (by /u/createmod)")

		resp, err := client.Do(req)
		if err != nil {
			slog.Warn("reddit metadata: fetch failed", "id", link.ID, "err", err)
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode != 200 {
			slog.Warn("reddit metadata: non-200 response", "id", link.ID, "status", resp.StatusCode)
			continue
		}

		title, upvotes, comments, thumbnail := parseRedditJSON(body)
		if title == "" {
			continue
		}

		if err := w.deps.Store.RedditLinks.UpdateMetadata(ctx, link.ID, title, upvotes, comments, thumbnail); err != nil {
			slog.Warn("reddit metadata: update failed", "id", link.ID, "err", err)
		} else {
			updated++
		}

		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("reddit metadata refresh completed", "checked", len(stale), "updated", updated)
	return nil
}

func parseRedditJSON(data []byte) (title string, upvotes, comments int, thumbnail string) {
	var listing []struct {
		Data struct {
			Children []struct {
				Data struct {
					Title        string `json:"title"`
					Ups          int    `json:"ups"`
					NumComments  int    `json:"num_comments"`
					Thumbnail    string `json:"thumbnail"`
					ThumbnailURL string `json:"thumbnail_url"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &listing); err != nil || len(listing) == 0 {
		return "", 0, 0, ""
	}

	if len(listing[0].Data.Children) == 0 {
		return "", 0, 0, ""
	}

	post := listing[0].Data.Children[0].Data
	thumb := post.Thumbnail
	if thumb == "self" || thumb == "default" || thumb == "nsfw" || thumb == "spoiler" || thumb == "" {
		thumb = ""
	}
	return post.Title, post.Ups, post.NumComments, thumb
}
