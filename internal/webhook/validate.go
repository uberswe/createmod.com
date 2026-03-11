package webhook

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// discordWebhookPattern matches /api/webhooks/{id}/{token}
var discordWebhookPattern = regexp.MustCompile(`^/api/webhooks/\d+/[\w-]+$`)

// maxWebhookURLLength is the maximum allowed length for a webhook URL.
// Discord webhook URLs are typically ~120 chars; this provides ample headroom.
const maxWebhookURLLength = 500

// ValidateDiscordWebhookURL validates that rawURL is a valid Discord webhook URL.
func ValidateDiscordWebhookURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if len(rawURL) > maxWebhookURLLength {
		return fmt.Errorf("webhook URL is too long")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	if u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS")
	}

	// Reject URLs with userinfo (e.g., https://discord.com@evil.com/...)
	// which could bypass host validation via URL parsing confusion.
	if u.User != nil {
		return fmt.Errorf("webhook URL must not contain credentials")
	}

	host := strings.ToLower(u.Host)
	if host != "discord.com" && host != "discordapp.com" {
		return fmt.Errorf("webhook URL must be a Discord webhook (discord.com or discordapp.com)")
	}

	if !discordWebhookPattern.MatchString(u.Path) {
		return fmt.Errorf("webhook URL must match the Discord webhook format: https://discord.com/api/webhooks/{id}/{token}")
	}

	return nil
}
