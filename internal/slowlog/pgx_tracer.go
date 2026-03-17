package slowlog

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// pgxTracerCtxKey is the context key for storing trace data between
// TraceQueryStart and TraceQueryEnd.
type pgxTracerCtxKey struct{}

type pgxTraceData struct {
	startTime time.Time
	sql       string
}

// PgxTracer implements pgx.QueryTracer to log queries that exceed Threshold.
type PgxTracer struct{}

var _ pgx.QueryTracer = (*PgxTracer)(nil)

// TraceQueryStart records the start time and SQL text in the context.
func (t *PgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, pgxTracerCtxKey{}, &pgxTraceData{
		startTime: time.Now(),
		sql:       data.SQL,
	})
}

// TraceQueryEnd checks the elapsed time and logs a warning if it exceeds Threshold.
func (t *PgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	td, ok := ctx.Value(pgxTracerCtxKey{}).(*pgxTraceData)
	if !ok || td == nil {
		return
	}

	elapsed := time.Since(td.startTime)
	if elapsed < Threshold {
		return
	}

	sql := td.sql
	if len(sql) > 200 {
		sql = sql[:200] + "..."
	}

	var errVal any
	if data.Err != nil {
		errVal = data.Err.Error()
	}

	slog.Warn("slow query",
		"subsystem", "postgres",
		"duration_ms", elapsed.Milliseconds(),
		"sql", sql,
		"command_tag", data.CommandTag.String(),
		"error", errVal,
	)
}
