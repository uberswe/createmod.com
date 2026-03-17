package slowlog

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// captureHandler is a slog.Handler that captures log records for assertions.
type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func TestThresholdValue(t *testing.T) {
	if Threshold != 500*time.Millisecond {
		t.Errorf("Threshold = %v, want 500ms", Threshold)
	}
}

func TestPgxTracer_BelowThreshold(t *testing.T) {
	h := &captureHandler{}
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	tracer := &PgxTracer{}

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL: "SELECT 1",
	})

	// Simulate immediate completion (well under 500ms)
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.NewCommandTag("SELECT 1"),
	})

	if len(h.records) != 0 {
		t.Errorf("expected no log records for fast query, got %d", len(h.records))
	}
}

func TestPgxTracer_AboveThreshold(t *testing.T) {
	h := &captureHandler{}
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	tracer := &PgxTracer{}

	// Manually set start time in the past to simulate a slow query
	ctx := context.WithValue(context.Background(), pgxTracerCtxKey{}, &pgxTraceData{
		startTime: time.Now().Add(-600 * time.Millisecond),
		sql:       "SELECT * FROM schematics WHERE id = $1",
	})

	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.NewCommandTag("SELECT 1"),
	})

	if len(h.records) != 1 {
		t.Fatalf("expected 1 log record for slow query, got %d", len(h.records))
	}

	if h.records[0].Level != slog.LevelWarn {
		t.Errorf("expected Warn level, got %v", h.records[0].Level)
	}

	if h.records[0].Message != "slow query" {
		t.Errorf("expected message 'slow query', got %q", h.records[0].Message)
	}
}

func TestPgxTracer_SQLTruncation(t *testing.T) {
	h := &captureHandler{}
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	tracer := &PgxTracer{}
	longSQL := strings.Repeat("x", 300)

	ctx := context.WithValue(context.Background(), pgxTracerCtxKey{}, &pgxTraceData{
		startTime: time.Now().Add(-600 * time.Millisecond),
		sql:       longSQL,
	})

	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.NewCommandTag("SELECT 1"),
	})

	if len(h.records) != 1 {
		t.Fatalf("expected 1 log record, got %d", len(h.records))
	}

	var sqlVal string
	h.records[0].Attrs(func(a slog.Attr) bool {
		if a.Key == "sql" {
			sqlVal = a.Value.String()
			return false
		}
		return true
	})

	// 200 chars + "..."
	if len(sqlVal) != 203 {
		t.Errorf("expected truncated SQL length 203, got %d", len(sqlVal))
	}
	if !strings.HasSuffix(sqlVal, "...") {
		t.Error("expected truncated SQL to end with '...'")
	}
}

func TestRedisHook_Interfaces(t *testing.T) {
	// Verify RedisHook implements redis.Hook at compile time.
	var _ redis.Hook = (*RedisHook)(nil)

	hook := &RedisHook{}

	// Verify all three hook methods return non-nil functions.
	if hook.DialHook(nil) == nil {
		t.Error("DialHook returned nil")
	}
	if hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error { return nil }) == nil {
		t.Error("ProcessHook returned nil")
	}
	if hook.ProcessPipelineHook(func(ctx context.Context, cmds []redis.Cmder) error { return nil }) == nil {
		t.Error("ProcessPipelineHook returned nil")
	}
}

func TestSlowHTTPTransport_Interface(t *testing.T) {
	// Verify SlowHTTPTransport implements http.RoundTripper at compile time.
	var _ http.RoundTripper = (*SlowHTTPTransport)(nil)
}
