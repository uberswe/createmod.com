package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// OAuthProvider defines an OAuth2 provider configuration.
type OAuthProvider struct {
	Name     string
	Config   oauth2.Config
	UserURL  string // API endpoint to fetch user info
	IDField  string // JSON field name for user ID
}

// OAuthUser represents a user fetched from an OAuth provider.
type OAuthUser struct {
	ID       string
	Username string
	Email    string
	Avatar   string
}

// NewDiscordProvider creates a Discord OAuth2 provider config.
func NewDiscordProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "discord",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		},
		UserURL: "https://discord.com/api/users/@me",
		IDField: "id",
	}
}

// NewGitHubProvider creates a GitHub OAuth2 provider config.
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "github",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
		},
		UserURL: "https://api.github.com/user",
		IDField: "id",
	}
}

// fetchGitHubPrimaryEmail calls GitHub's /user/emails endpoint and returns the
// user's verified primary email. Requires the `user:email` scope (which we
// already request). The endpoint returns an array of {email, primary,
// verified, visibility}.
func fetchGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching github emails: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching github emails: status %d", resp.StatusCode)
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("parsing github emails: %w", err)
	}
	// Prefer the primary verified address; fall back to any verified address.
	for _, em := range emails {
		if em.Primary && em.Verified && em.Email != "" {
			return em.Email, nil
		}
	}
	for _, em := range emails {
		if em.Verified && em.Email != "" {
			return em.Email, nil
		}
	}
	return "", fmt.Errorf("no verified github email found")
}

// AuthURL returns the OAuth2 authorization URL with the given state parameter.
func (p *OAuthProvider) AuthURL(state string) string {
	return p.Config.AuthCodeURL(state)
}

// Exchange exchanges an authorization code for a token.
func (p *OAuthProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.Config.Exchange(ctx, code)
}

// FetchUser fetches the user info from the provider using the given token.
func (p *OAuthProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*OAuthUser, error) {
	client := p.Config.Client(ctx, token)
	resp, err := client.Get(p.UserURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s user info: %w", p.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s user info: status %d", p.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s user response: %w", p.Name, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parsing %s user response: %w", p.Name, err)
	}

	user := &OAuthUser{}

	switch p.Name {
	case "discord":
		user.ID = fmt.Sprintf("%v", data["id"])
		user.Username = fmt.Sprintf("%v", data["username"])
		if email, ok := data["email"].(string); ok {
			user.Email = email
		}
		if avatar, ok := data["avatar"].(string); ok && avatar != "" {
			user.Avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, avatar)
		}
	case "github":
		user.ID = fmt.Sprintf("%v", data["id"])
		if login, ok := data["login"].(string); ok {
			user.Username = login
		}
		if email, ok := data["email"].(string); ok {
			user.Email = email
		}
		if avatar, ok := data["avatar_url"].(string); ok {
			user.Avatar = avatar
		}
		// GitHub returns email: null when the user's primary email is set to
		// private. Fall back to the /user/emails endpoint to fetch the
		// verified primary email so we can create an account.
		if user.Email == "" {
			if email, err := fetchGitHubPrimaryEmail(ctx, client); err == nil {
				user.Email = email
			}
		}
	default:
		user.ID = fmt.Sprintf("%v", data[p.IDField])
	}

	return user, nil
}
