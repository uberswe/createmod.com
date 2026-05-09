package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	steamOpenIDEndpoint = "https://steamcommunity.com/openid/login"
	steamProfileAPI     = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
)

// SteamProvider handles Steam OpenID 2.0 authentication.
type SteamProvider struct {
	APIKey      string
	RedirectURL string
}

// NewSteamProvider creates a Steam OpenID provider. The apiKey is a Steam Web
// API key used to fetch profile data after authentication.
func NewSteamProvider(apiKey, redirectURL string) *SteamProvider {
	return &SteamProvider{
		APIKey:      apiKey,
		RedirectURL: redirectURL,
	}
}

// AuthURL returns the Steam OpenID login URL.
func (p *SteamProvider) AuthURL() string {
	params := url.Values{
		"openid.ns":         {"http://specs.openid.net/auth/2.0"},
		"openid.mode":       {"checkid_setup"},
		"openid.return_to":  {p.RedirectURL},
		"openid.realm":      {extractRealm(p.RedirectURL)},
		"openid.identity":   {"http://specs.openid.net/auth/2.0/identifier_select"},
		"openid.claimed_id": {"http://specs.openid.net/auth/2.0/identifier_select"},
	}
	return steamOpenIDEndpoint + "?" + params.Encode()
}

// ValidateCallback verifies the OpenID response from Steam and returns the
// Steam ID (64-bit) on success.
func (p *SteamProvider) ValidateCallback(ctx context.Context, query url.Values) (string, error) {
	if query.Get("openid.mode") != "id_res" {
		return "", fmt.Errorf("steam: unexpected openid.mode: %s", query.Get("openid.mode"))
	}

	verification := url.Values{}
	for k, v := range query {
		verification[k] = v
	}
	verification.Set("openid.mode", "check_authentication")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(steamOpenIDEndpoint, verification)
	if err != nil {
		return "", fmt.Errorf("steam: verification request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", fmt.Errorf("steam: reading verification response: %w", err)
	}

	if !strings.Contains(string(body), "is_valid:true") {
		return "", fmt.Errorf("steam: OpenID verification failed")
	}

	claimedID := query.Get("openid.claimed_id")
	if claimedID == "" {
		return "", fmt.Errorf("steam: missing claimed_id")
	}

	// Extract Steam ID from: https://steamcommunity.com/openid/id/76561198012345678
	parts := strings.Split(claimedID, "/")
	steamID := parts[len(parts)-1]
	if steamID == "" || len(steamID) < 10 {
		return "", fmt.Errorf("steam: invalid Steam ID in claimed_id: %s", claimedID)
	}

	return steamID, nil
}

// FetchUser fetches the Steam user's profile using the Steam Web API.
func (p *SteamProvider) FetchUser(ctx context.Context, steamID string) (*OAuthUser, error) {
	if p.APIKey == "" {
		return &OAuthUser{ID: steamID, Username: "steam_" + steamID}, nil
	}

	apiURL := fmt.Sprintf("%s?key=%s&steamids=%s", steamProfileAPI, p.APIKey, steamID)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("steam: creating profile request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("steam: fetching profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &OAuthUser{ID: steamID, Username: "steam_" + steamID}, nil
	}

	var result struct {
		Response struct {
			Players []struct {
				SteamID    string `json:"steamid"`
				PersonName string `json:"personaname"`
				AvatarFull string `json:"avatarfull"`
				ProfileURL string `json:"profileurl"`
			} `json:"players"`
		} `json:"response"`
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(body, &result); err != nil || len(result.Response.Players) == 0 {
		return &OAuthUser{ID: steamID, Username: "steam_" + steamID}, nil
	}

	player := result.Response.Players[0]
	return &OAuthUser{
		ID:       player.SteamID,
		Username: player.PersonName,
		Avatar:   player.AvatarFull,
	}, nil
}

func extractRealm(callbackURL string) string {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return callbackURL
	}
	return u.Scheme + "://" + u.Host
}
