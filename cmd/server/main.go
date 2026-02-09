package main

import (
	"createmod/server"
	"github.com/joho/godotenv"
	"log"
	"os"
)

const (
	AutoMigrate       = "AUTO_MIGRATE"
	CreateAdmin       = "CREATE_ADMIN"
	DiscordWebhookUrl = "DISCORD_WEBHOOK_URL"
	OpenAIAPIKey      = "OPENAI_API_KEY"
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

	s := server.New(server.Config{
		AutoMigrate:       getEnv(envFile, AutoMigrate) == "true",
		CreateAdmin:       getEnv(envFile, CreateAdmin) == "true",
		DiscordWebhookUrl: getEnv(envFile, DiscordWebhookUrl),
		OpenAIApiKey:      getEnv(envFile, OpenAIAPIKey),
	})
	s.Start()
}
