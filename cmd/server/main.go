package main

import (
	"context"
	"createmod/internal/database"
	"createmod/server"
	"github.com/joho/godotenv"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	AutoMigrate         = "AUTO_MIGRATE"
	CreateAdmin         = "CREATE_ADMIN"
	DiscordWebhookUrl   = "DISCORD_WEBHOOK_URL"
	OpenAIAPIKey        = "OPENAI_API_KEY"
	CurseForgeAPIKey    = "CURSEFORGE_API_KEY"
	DevEnv              = "DEV"
	DatabaseURL         = "DATABASE_URL"
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

	conf.Store = database.NewStore(pool)
	conf.Pool = pool

	// Check for pending SQLite migration
	checkPendingSQLiteMigration(pool, ctx)

	s := server.New(conf)
	s.Start()
}

// checkPendingSQLiteMigration logs a warning if a PocketBase SQLite database
// exists but the PostgreSQL users table is empty, suggesting data migration
// has not been run yet.
func checkPendingSQLiteMigration(pool *pgxpool.Pool, ctx context.Context) {
	if _, err := os.Stat("./pb_data/data.db"); err != nil {
		return // no SQLite DB found, nothing to warn about
	}
	var count int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return // table may not exist yet
	}
	if count == 0 {
		log.Println("WARNING: pb_data/data.db exists but PostgreSQL users table is empty.")
		log.Println("Run: go run ./cmd/migrate-data --pg=$DATABASE_URL")
	}
}
