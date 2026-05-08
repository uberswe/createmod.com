package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type TwitchStreamSearchArgs struct{}

func (TwitchStreamSearchArgs) Kind() string { return "twitch_stream_search" }

type TwitchStreamSearchWorker struct {
	river.WorkerDefaults[TwitchStreamSearchArgs]
	deps Deps
}

func (w *TwitchStreamSearchWorker) Work(ctx context.Context, job *river.Job[TwitchStreamSearchArgs]) error {
	slog.Info("twitch stream search started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: search Twitch for Minecraft streams tagged #CreateMod
	// GET https://api.twitch.tv/helix/streams?game_id=27471&first=100
	// Filter by tags containing "CreateMod"
	// Cross-reference with site users, cache in Redis
	slog.Info("twitch stream search completed")
	return nil
}
