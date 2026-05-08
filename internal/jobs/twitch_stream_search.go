package jobs

import (
	"context"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/riverqueue/river"
)

type TwitchStreamSearchArgs struct{}

func (TwitchStreamSearchArgs) Kind() string { return "twitch_stream_search" }

type TwitchStreamSearchWorker struct {
	river.WorkerDefaults[TwitchStreamSearchArgs]
	deps Deps
}

type TwitchStream struct {
	UserName     string `json:"user_name"`
	UserLogin    string `json:"user_login"`
	Title        string `json:"title"`
	ViewerCount  int    `json:"viewer_count"`
	ThumbnailURL string `json:"thumbnail_url"`
	Tags         []string `json:"tags"`
	GameName     string `json:"game_name"`
}

type twitchStreamsResponse struct {
	Data       []TwitchStream `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

type twitchTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

const twitchMinecraftGameID = "27471"

func (w *TwitchStreamSearchWorker) Work(ctx context.Context, job *river.Job[TwitchStreamSearchArgs]) error {
	slog.Info("twitch stream search started")
	if w.deps.Cache == nil || w.deps.TwitchClientID == "" || w.deps.TwitchClientSecret == "" {
		slog.Info("twitch stream search skipped: missing credentials or cache")
		return nil
	}

	token, err := getTwitchAppToken(ctx, w.deps.Cache, w.deps.TwitchClientID, w.deps.TwitchClientSecret)
	if err != nil {
		slog.Error("twitch stream search: failed to get app token", "error", err)
		return nil
	}

	streams, err := w.fetchCreateModStreams(ctx, token)
	if err != nil {
		slog.Error("twitch stream search: failed to fetch streams", "error", err)
		return nil
	}

	cached := make([]store.CachedTwitchStream, len(streams))
	for i, s := range streams {
		cached[i] = store.CachedTwitchStream{
			UserName:     s.UserName,
			UserLogin:    s.UserLogin,
			Title:        s.Title,
			ViewerCount:  s.ViewerCount,
			ThumbnailURL: s.ThumbnailURL,
		}
	}
	w.deps.Cache.SetWithTTL("twitch_live_streams", cached, 6*time.Minute)

	if w.deps.Store != nil {
		siteMembers := make(map[string]bool)
		links, _ := w.deps.Store.SocialLinks.ListByPlatform(ctx, "twitch")
		for _, link := range links {
			siteMembers[strings.ToLower(link.Username)] = true
		}
		w.deps.Cache.SetWithTTL("twitch_site_members", siteMembers, 6*time.Minute)
	}

	slog.Info("twitch stream search completed", "streams_found", len(streams))
	return nil
}

func (w *TwitchStreamSearchWorker) fetchCreateModStreams(ctx context.Context, token string) ([]TwitchStream, error) {
	var allStreams []TwitchStream
	cursor := ""

	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("https://api.twitch.tv/helix/streams?game_id=%s&first=100", twitchMinecraftGameID)
		if cursor != "" {
			url += "&after=" + cursor
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Client-ID", w.deps.TwitchClientID)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("twitch streams request failed: %d %s", resp.StatusCode, string(b))
		}

		var streamsResp twitchStreamsResponse
		if err := json.NewDecoder(resp.Body).Decode(&streamsResp); err != nil {
			return nil, err
		}

		for _, s := range streamsResp.Data {
			if isCreateModStream(s) {
				thumb := s.ThumbnailURL
				thumb = strings.Replace(thumb, "{width}", "440", 1)
				thumb = strings.Replace(thumb, "{height}", "248", 1)
				s.ThumbnailURL = thumb
				allStreams = append(allStreams, s)
			}
		}

		if streamsResp.Pagination.Cursor == "" {
			break
		}
		cursor = streamsResp.Pagination.Cursor
	}

	return allStreams, nil
}

func isCreateModStream(s TwitchStream) bool {
	titleLower := strings.ToLower(s.Title)
	if strings.Contains(titleLower, "create mod") ||
		strings.Contains(titleLower, "createmod") ||
		strings.Contains(titleLower, "create:") {
		return true
	}
	for _, tag := range s.Tags {
		tagLower := strings.ToLower(tag)
		if tagLower == "createmod" || tagLower == "create mod" || tagLower == "create" {
			return true
		}
	}
	return false
}
