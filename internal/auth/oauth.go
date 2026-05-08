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

// NewTwitchProvider creates a Twitch OAuth2 provider config.
func NewTwitchProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "twitch",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:read:email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://id.twitch.tv/oauth2/authorize",
				TokenURL: "https://id.twitch.tv/oauth2/token",
			},
		},
		UserURL: "https://api.twitch.tv/helix/users",
		IDField: "id",
	}
}

// NewPatreonProvider creates a Patreon OAuth2 provider config.
func NewPatreonProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "patreon",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"identity", "identity[email]"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.patreon.com/oauth2/authorize",
				TokenURL: "https://www.patreon.com/api/oauth2/token",
			},
		},
		UserURL: "https://www.patreon.com/api/oauth2/v2/identity?fields%5Buser%5D=email,full_name,image_url,vanity",
		IDField: "id",
	}
}

// NewRedditProvider creates a Reddit OAuth2 provider config.
func NewRedditProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "reddit",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"identity"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.reddit.com/api/v1/authorize",
				TokenURL: "https://www.reddit.com/api/v1/access_token",
			},
		},
		UserURL: "https://oauth.reddit.com/api/v1/me",
		IDField: "id",
	}
}

// NewGoogleProvider creates a Google OAuth2 provider config.
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "google",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
		UserURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		IDField: "id",
	}
}

// NewMicrosoftProvider creates a Microsoft OAuth2 provider config.
func NewMicrosoftProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		Name: "microsoft",
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile", "User.Read"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
			},
		},
		UserURL: "https://graph.microsoft.com/v1.0/me",
		IDField: "id",
	}
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

	req, err := http.NewRequestWithContext(ctx, "GET", p.UserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating %s user info request: %w", p.Name, err)
	}

	if p.Name == "twitch" {
		req.Header.Set("Client-Id", p.Config.ClientID)
	}
	if p.Name == "reddit" {
		req.Header.Set("User-Agent", "createmod.com:v1.0 (by /u/createmod)")
	}

	resp, err := client.Do(req)
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
		if user.Email == "" {
			if email, err := fetchGitHubPrimaryEmail(ctx, client); err == nil {
				user.Email = email
			}
		}
	case "twitch":
		if dataArr, ok := data["data"].([]interface{}); ok && len(dataArr) > 0 {
			if u, ok := dataArr[0].(map[string]interface{}); ok {
				user.ID = fmt.Sprintf("%v", u["id"])
				if login, ok := u["login"].(string); ok {
					user.Username = login
				}
				if email, ok := u["email"].(string); ok {
					user.Email = email
				}
				if avatar, ok := u["profile_image_url"].(string); ok {
					user.Avatar = avatar
				}
			}
		}
	case "patreon":
		if d, ok := data["data"].(map[string]interface{}); ok {
			user.ID = fmt.Sprintf("%v", d["id"])
			if attrs, ok := d["attributes"].(map[string]interface{}); ok {
				if email, ok := attrs["email"].(string); ok {
					user.Email = email
				}
				if name, ok := attrs["full_name"].(string); ok {
					user.Username = name
				}
				if vanity, ok := attrs["vanity"].(string); ok && vanity != "" {
					user.Username = vanity
				}
				if img, ok := attrs["image_url"].(string); ok {
					user.Avatar = img
				}
			}
		}
	case "reddit":
		user.ID = fmt.Sprintf("%v", data["id"])
		if name, ok := data["name"].(string); ok {
			user.Username = name
		}
		if icon, ok := data["icon_img"].(string); ok {
			user.Avatar = icon
		}
	case "google":
		user.ID = fmt.Sprintf("%v", data["id"])
		if name, ok := data["name"].(string); ok {
			user.Username = name
		}
		if email, ok := data["email"].(string); ok {
			user.Email = email
		}
		if picture, ok := data["picture"].(string); ok {
			user.Avatar = picture
		}
	case "microsoft":
		user.ID = fmt.Sprintf("%v", data["id"])
		if name, ok := data["displayName"].(string); ok {
			user.Username = name
		}
		if email, ok := data["mail"].(string); ok && email != "" {
			user.Email = email
		} else if upn, ok := data["userPrincipalName"].(string); ok {
			user.Email = upn
		}
	default:
		user.ID = fmt.Sprintf("%v", data[p.IDField])
	}

	return user, nil
}
