package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/riverqueue/river"
)

type TwitchLiveCheckArgs struct{}

func (TwitchLiveCheckArgs) Kind() string { return "twitch_live_check" }

type TwitchLiveCheckWorker struct {
	river.WorkerDefaults[TwitchLiveCheckArgs]
	deps Deps
}

func (w *TwitchLiveCheckWorker) Work(ctx context.Context, job *river.Job[TwitchLiveCheckArgs]) error {
	slog.Info("twitch live check started")
	if w.deps.Store == nil || w.deps.Cache == nil || w.deps.TwitchClientID == "" || w.deps.TwitchClientSecret == "" {
		slog.Info("twitch live check skipped: missing deps")
		return nil
	}

	links, err := w.deps.Store.SocialLinks.ListByPlatform(ctx, "twitch")
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}

	token, err := getTwitchAppToken(ctx, w.deps.Cache, w.deps.TwitchClientID, w.deps.TwitchClientSecret)
	if err != nil {
		slog.Error("twitch live check: failed to get app token", "error", err)
		return nil
	}

	// Batch check in groups of 100 (Twitch API limit)
	for i := 0; i < len(links); i += 100 {
		end := i + 100
		if end > len(links) {
			end = len(links)
		}
		batch := links[i:end]

		url := "https://api.twitch.tv/helix/streams?"
		for j, link := range batch {
			if j > 0 {
				url += "&"
			}
			url += "user_login=" + link.Username
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			slog.Error("twitch live check: request build failed", "error", err)
			continue
		}
		req.Header.Set("Client-ID", w.deps.TwitchClientID)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("twitch live check: request failed", "error", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			slog.Error("twitch live check: bad response", "status", resp.StatusCode, "body", string(b))
			continue
		}

		var streamsResp twitchStreamsResponse
		if err := json.NewDecoder(resp.Body).Decode(&streamsResp); err != nil {
			resp.Body.Close()
			slog.Error("twitch live check: decode failed", "error", err)
			continue
		}
		resp.Body.Close()

		liveSet := make(map[string]bool, len(streamsResp.Data))
		for _, s := range streamsResp.Data {
			liveSet[s.UserLogin] = true
		}

		for _, link := range batch {
			cacheKey := fmt.Sprintf("twitch:live:%s", link.UserID)
			if liveSet[link.Username] {
				w.deps.Cache.SetWithTTL(cacheKey, true, 6*time.Minute)
			} else {
				w.deps.Cache.SetWithTTL(cacheKey, false, 6*time.Minute)
			}
		}
	}

	slog.Info("twitch live check completed", "users_checked", len(links))
	return nil
}

