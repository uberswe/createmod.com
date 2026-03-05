package main

import (
	"context"
	"createmod/internal/database"
	"createmod/internal/migrate"
	"createmod/internal/storage"
	"createmod/server"
	"database/sql"
	"log"
	"os"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	_ "modernc.org/sqlite"
)

const (
	AutoMigrate         = "AUTO_MIGRATE"
	CreateAdmin         = "CREATE_ADMIN"
	DiscordWebhookUrl   = "DISCORD_WEBHOOK_URL"
	OpenAIAPIKey        = "OPENAI_API_KEY"
	CurseForgeAPIKey    = "CURSEFORGE_API_KEY"
	DevEnv              = "DEV"
	DatabaseURL         = "DATABASE_URL"
	S3Endpoint          = "S3_ENDPOINT"
	S3AccessKey         = "S3_ACCESS_KEY"
	S3SecretKey         = "S3_SECRET_KEY"
	S3Bucket            = "S3_BUCKET"
	S3UseSSL            = "S3_USE_SSL"
	DiscordClientID     = "DISCORD_CLIENT_ID"
	DiscordClientSecret = "DISCORD_CLIENT_SECRET"
	GithubClientID      = "GITHUB_CLIENT_ID"
	GithubClientSecret  = "GITHUB_CLIENT_SECRET"
	BaseURL             = "BASE_URL"
)

// getEnv returns the value from the envFile map if present,
// otherwise falls back to the OS environment variable.
// This allows the app to work both with .env files (local dev)
// and Kubernetes environment variables (production).
func getEnv(envFile map[string]string, key string) string {
	if val, ok := envFile[key]; ok && val != "" {
		return val
	}
	return os.Getenv(key)
}

func main() {
	envFile, err := godotenv.Read(".env")

	if err != nil {
		// Continue without env file - will use OS environment variables
		log.Println("No .env file found, using environment variables")
		envFile = make(map[string]string)
	}

	// Export .env values to OS environment so os.Getenv works throughout the app
	for k, v := range envFile {
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}

	conf := server.Config{
		Dev:                 getEnv(envFile, DevEnv) == "true",
		AutoMigrate:         getEnv(envFile, AutoMigrate) == "true",
		CreateAdmin:         getEnv(envFile, CreateAdmin) == "true",
		DiscordWebhookUrl:   getEnv(envFile, DiscordWebhookUrl),
		OpenAIApiKey:        getEnv(envFile, OpenAIAPIKey),
		CurseForgeApiKey:    getEnv(envFile, CurseForgeAPIKey),
		DatabaseURL:         getEnv(envFile, DatabaseURL),
		DiscordClientID:     getEnv(envFile, DiscordClientID),
		DiscordClientSecret: getEnv(envFile, DiscordClientSecret),
		GithubClientID:      getEnv(envFile, GithubClientID),
		GithubClientSecret:  getEnv(envFile, GithubClientSecret),
		BaseURL:             getEnv(envFile, BaseURL),
	}

	// PostgreSQL is required
	if conf.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Println("Connecting to PostgreSQL...")
	if err := database.RunMigrations(conf.DatabaseURL); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database migrations complete")

	ctx := context.Background()
	pool, err := database.Connect(ctx, database.DefaultConfig(conf.DatabaseURL))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// Run River queue migrations (creates river_job, river_queue, etc.)
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		log.Fatalf("Failed to create River migrator: %v", err)
	}
	if _, err := riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		log.Fatalf("Failed to run River migrations: %v", err)
	}
	log.Println("River migrations complete")

	conf.Store = database.NewStore(pool)
	conf.Pool = pool

	// Initialize S3/Minio storage service (optional — if not configured, storage
	// features like file serving and thumbnailing will be unavailable).
	if s3Endpoint := getEnv(envFile, S3Endpoint); s3Endpoint != "" {
		storageSvc, err := storage.New(storage.Config{
			Endpoint:  s3Endpoint,
			AccessKey: getEnv(envFile, S3AccessKey),
			SecretKey: getEnv(envFile, S3SecretKey),
			Bucket:    getEnv(envFile, S3Bucket),
			UseSSL:    getEnv(envFile, S3UseSSL) == "true",
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		conf.Storage = storageSvc
		log.Println("Connected to S3 storage")
	} else {
		log.Println("WARNING: S3_ENDPOINT not set, file storage features will be unavailable")
	}

	// Check if SQLite auto-migration is needed. If so, enable maintenance mode
	// so the server serves a "Coming Back Soon" page while data is being
	// migrated, then disable it when done.
	maintenance := &atomic.Bool{}
	needsMigration := sqliteMigrationNeeded(pool, ctx)
	if needsMigration {
		maintenance.Store(true)
		log.Println("SQLite migration pending — enabling maintenance mode")
	}
	conf.MaintenanceMode = maintenance

	s := server.New(conf)

	if needsMigration {
		go func() {
			autoMigrateFromSQLite(pool, ctx)
			maintenance.Store(false)
			log.Println("Maintenance mode disabled")
			s.PostMigrationRebuild()
		}()
	}

	s.Start()
}

// sqliteMigrationNeeded checks whether a SQLite database exists and PostgreSQL
// is empty, without actually running the migration.
func sqliteMigrationNeeded(pool *pgxpool.Pool, ctx context.Context) bool {
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./pb_data/data.db"
	}
	if _, err := os.Stat(sqlitePath); err != nil {
		return false
	}
	var count int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false
	}
	return count == 0
}

// autoMigrateFromSQLite checks for a PocketBase SQLite database and automatically
// migrates data to PostgreSQL if the PG users table is empty. This is a one-time
// operation — once PG has data, the SQLite file is ignored.
func autoMigrateFromSQLite(pool *pgxpool.Pool, ctx context.Context) {
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./pb_data/data.db"
	}
	if _, err := os.Stat(sqlitePath); err != nil {
		return // no SQLite DB found
	}

	// Check if PostgreSQL already has data
	var count int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return // table may not exist yet
	}
	if count > 0 {
		return // PG already has data, skip migration
	}

	log.Printf("SQLite database found at %s and PostgreSQL is empty — starting automatic migration...", sqlitePath)

	sqliteDB, err := sql.Open("sqlite", sqlitePath+"?mode=ro")
	if err != nil {
		log.Printf("WARNING: Failed to open SQLite for migration: %v", err)
		return
	}
	defer sqliteDB.Close()

	if err := sqliteDB.Ping(); err != nil {
		log.Printf("WARNING: Failed to read SQLite database: %v", err)
		return
	}

	m := migrate.New(sqliteDB, pool, false)
	if err := m.Run(ctx); err != nil {
		log.Printf("WARNING: SQLite migration failed: %v", err)
		log.Println("You can retry manually with: go run ./cmd/migrate-data --pg=$DATABASE_URL")
		return
	}

	log.Println("=== Automatic SQLite migration complete ===")
	m.Validate(ctx)
}
