package jobs

import (
	"context"
	"log/slog"

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
	if w.deps.Store == nil {
		return nil
	}

	// Get all Twitch-linked users
	links, err := w.deps.Store.SocialLinks.ListByPlatform(ctx, "twitch")
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}

	// TODO: use Twitch Helix API to batch check live streams
	// GET https://api.twitch.tv/helix/streams?user_login=...
	// Store results in Redis with 5min TTL: twitch:live:{user_id}
	slog.Info("twitch live check completed", "users_checked", len(links))
	return nil
}
