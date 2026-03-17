package slowlog

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisHook implements redis.Hook to log commands that exceed Threshold.
type RedisHook struct{}

var _ redis.Hook = (*RedisHook)(nil)

// DialHook passes through without timing.
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook wraps individual command execution with timing.
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		elapsed := time.Since(start)

		if elapsed >= Threshold {
			var errVal any
			if err != nil {
				errVal = err.Error()
			}
			slog.Warn("slow redis command",
				"subsystem", "redis",
				"duration_ms", elapsed.Milliseconds(),
				"cmd", cmd.Name(),
				"args", cmd.String(),
				"error", errVal,
			)
		}

		return err
	}
}

// ProcessPipelineHook wraps pipeline execution with timing.
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		elapsed := time.Since(start)

		if elapsed >= Threshold {
			var errVal any
			if err != nil {
				errVal = err.Error()
			}
			slog.Warn("slow redis pipeline",
				"subsystem", "redis",
				"duration_ms", elapsed.Milliseconds(),
				"cmd_count", len(cmds),
				"error", errVal,
			)
		}

		return err
	}
}
