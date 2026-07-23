package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"createmod/internal/slowlog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds database connection configuration.
type Config struct {
	DatabaseURL     string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// DefaultConfig returns a Config with pool sizing appropriate for a shared
// PostgreSQL. The shared cluster instance has 200 connection slots serving
// ~10 applications; this app runs 2-6 pods (HPA) and rolling updates surge
// pods, so per-pod appetite multiplies: 25/pod peaked past the global limit
// during a deploy and starved other tenants (authentik outage, 2026-07-09).
// Observed steady-state usage is ~8 connections per pod; 10 gives headroom
// while capping the worst case (6 pods + surge) near 90. Override with
// DB_MAX_CONNS if a dedicated database ever warrants more.
func DefaultConfig(databaseURL string) Config {
	maxConns := int32(10)
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 2 && n <= 100 {
			maxConns = int32(n)
		}
	}
	return Config{
		DatabaseURL:     databaseURL,
		MaxConns:        maxConns,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
	}
}

// DefaultReplicaConfig returns pool sizing for the optional read-replica
// connection (DATABASE_REPLICA_URL). This URL is expected to point at
// pgbouncer's <db>_ro entry (transaction pooling → postgresql-read), where
// client connections are cheap, but the replica itself is small so keep the
// per-pod appetite modest. Override with DB_REPLICA_MAX_CONNS.
//
// pgbouncer runs transaction pooling with max_prepared_statements, which
// lets pgx's default query mode (cache_statement) work through the pooler.
// Do NOT add default_query_exec_mode overrides to pooled URLs: exec mode
// loses parameter type info (sqlc passes json columns as []byte -> SQLSTATE
// 22P02) and describe_exec races across round trips through a transaction
// pooler; both broke prod on 2026-07-23.
func DefaultReplicaConfig(databaseURL string) Config {
	maxConns := int32(5)
	if v := os.Getenv("DB_REPLICA_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 2 && n <= 100 {
			maxConns = int32(n)
		}
	}
	return Config{
		DatabaseURL:     databaseURL,
		MaxConns:        maxConns,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
	}
}

// DefaultRiverConfig returns pool sizing for the dedicated River pool used
// when DATABASE_DIRECT_URL is set (i.e. DATABASE_URL goes through pgbouncer).
// River needs a direct PostgreSQL connection — it relies on LISTEN/NOTIFY,
// which pgbouncer's transaction pooling does not support. The pool only
// carries queue bookkeeping (fetch/complete/insert + one LISTEN connection);
// job payload work goes through the regular stores. Override with
// DB_RIVER_MAX_CONNS.
func DefaultRiverConfig(databaseURL string) Config {
	maxConns := int32(5)
	if v := os.Getenv("DB_RIVER_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 2 && n <= 100 {
			maxConns = int32(n)
		}
	}
	return Config{
		DatabaseURL:     databaseURL,
		MaxConns:        maxConns,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
	}
}

// Connect creates a new pgxpool connected to PostgreSQL.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.ConnConfig.Tracer = &slowlog.PgxTracer{}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			stat := pool.Stat()
			slog.Info("pgpool",
				"acquired", stat.AcquiredConns(),
				"idle", stat.IdleConns(),
				"total", stat.TotalConns(),
				"max", stat.MaxConns(),
			)
		}
	}()

	return pool, nil
}

// RunMigrations runs all pending database migrations.
// Uses PostgreSQL advisory locks to ensure only one pod runs migrations
// even with multiple replicas.
//
// If a previous migration left the database in a dirty state, this function
// force-sets the version to the last clean version and retries.
func RunMigrations(databaseURL string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// If the database is in a dirty state from a previously failed migration,
		// force-set it to the previous clean version and retry.
		version, dirty, verr := m.Version()
		if verr == nil && dirty {
			fmt.Printf("Migration version %d is dirty, forcing to version %d and retrying\n", version, version-1)
			if ferr := m.Force(int(version) - 1); ferr != nil {
				return fmt.Errorf("forcing migration version: %w (original error: %w)", ferr, err)
			}
			if rerr := m.Up(); rerr != nil && rerr != migrate.ErrNoChange {
				return fmt.Errorf("running migrations after dirty fix: %w", rerr)
			}
			return nil
		}
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
