package discord

import (
	"context"
	"log/slog"

	"createmod/internal/store"
	"createmod/internal/webhook"

	"github.com/gtuk/discordwebhook"
)

type Service struct {
	webhookUrl string
}

func New(webhookUrl string) *Service {
	return &Service{
		webhookUrl: webhookUrl,
	}
}

func (s *Service) Post(content string) {
	if s.webhookUrl == "" {
		return
	}
	var username = "CreateMod.com"
	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	err := discordwebhook.SendMessage(s.webhookUrl, message)
	if err != nil {
		slog.Error("discord webhook failed", "error", err)
	}
}

// PostWithUserWebhooks sends the message to the site's own webhook and also
// to all active user webhooks. Failed user webhooks are tracked; after 3
// consecutive failures a webhook is automatically disabled.
func (s *Service) PostWithUserWebhooks(content string, webhookStore store.WebhookStore, webhookSecret string) {
	// Send to site webhook first
	s.Post(content)

	if webhookStore == nil || webhookSecret == "" {
		return
	}

	ctx := context.Background()
	activeWebhooks, err := webhookStore.ListActive(ctx)
	if err != nil {
		slog.Error("failed to list active user webhooks", "error", err)
		return
	}

	var username = "CreateMod.com"
	message := discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}

	for _, wh := range activeWebhooks {
		whURL, err := webhook.Decrypt(wh.WebhookURLEncrypted, webhookSecret)
		if err != nil {
			slog.Error("failed to decrypt user webhook URL", "webhookID", wh.ID, "error", err)
			_ = webhookStore.IncrementFailure(ctx, wh.ID, "decryption failed")
			continue
		}

		// Re-validate the decrypted URL as a defence-in-depth measure.
		if err := webhook.ValidateDiscordWebhookURL(whURL); err != nil {
			slog.Error("decrypted user webhook URL is invalid", "webhookID", wh.ID, "error", err)
			_ = webhookStore.IncrementFailure(ctx, wh.ID, "invalid webhook URL: "+err.Error())
			continue
		}

		if err := discordwebhook.SendMessage(whURL, message); err != nil {
			slog.Error("user webhook send failed", "webhookID", wh.ID, "error", err)
			_ = webhookStore.IncrementFailure(ctx, wh.ID, err.Error())
		} else {
			_ = webhookStore.ResetFailures(ctx, wh.ID)
		}
	}
}

