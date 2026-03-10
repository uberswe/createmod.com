package main

import (
	"context"
	"createmod/internal/database"
	"createmod/internal/storage"
	"createmod/server"
	"log"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
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

	conf.MaintenanceMode = &atomic.Bool{}

	s := server.New(conf)
	s.Start()
}
