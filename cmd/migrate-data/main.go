package main

import (
	"context"
	"createmod/internal/migrate"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"
)

func main() {
	sqlitePath := flag.String("sqlite", "./pb_data/data.db", "Path to PocketBase SQLite database")
	pgURL := flag.String("pg", "", "PostgreSQL connection URL (required)")
	dryRun := flag.Bool("dry-run", false, "Print row counts without writing to PostgreSQL")
	flag.Parse()

	if *pgURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --pg flag is required")
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/migrate-data --pg=postgres://user:pass@localhost:5432/createmod")
		os.Exit(1)
	}

	if _, err := os.Stat(*sqlitePath); os.IsNotExist(err) {
		log.Fatalf("SQLite database not found at %s", *sqlitePath)
	}

	// Open SQLite
	sqliteDB, err := sql.Open("sqlite", *sqlitePath+"?mode=ro")
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}
	defer sqliteDB.Close()

	// Verify SQLite is readable
	if err := sqliteDB.Ping(); err != nil {
		log.Fatalf("Failed to ping SQLite: %v", err)
	}
	log.Printf("Opened SQLite database: %s", *sqlitePath)

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, *pgURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	m := migrate.New(sqliteDB, pool, *dryRun)

	if *dryRun {
		log.Println("=== DRY RUN MODE — no data will be written ===")
		m.PrintCounts(ctx)
		return
	}

	if err := m.Run(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("=== Migration complete ===")
	m.Validate(ctx)
}
