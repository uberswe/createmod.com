package jobs

import (
	"context"
	"createmod/internal/cache"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func getTwitchAppToken(ctx context.Context, cacheService *cache.Service, clientID, clientSecret string) (string, error) {
	if cached, found := cacheService.Get("twitch_app_token"); found {
		if token, ok := cached.(string); ok {
			return token, nil
		}
	}

	body := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials",
		clientID, clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://id.twitch.tv/oauth2/token",
		strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("twitch token request failed: %d %s", resp.StatusCode, string(b))
	}

	var tokenResp twitchTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	ttl := time.Duration(tokenResp.ExpiresIn-60) * time.Second
	if ttl < time.Minute {
		ttl = time.Minute
	}
	cacheService.SetWithTTL("twitch_app_token", tokenResp.AccessToken, ttl)

	return tokenResp.AccessToken, nil
}
