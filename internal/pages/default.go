package pages

import (
	stdctx "context"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/session"
	"createmod/internal/store"
	"github.com/drexedam/gravatar"
	"createmod/internal/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type DefaultData struct {
	IsAuthenticated bool
	Username        string
	UserID          string
	UsernameSlug    string
	Title           string
	Description     string
	Slug            string
	Thumbnail       string
	SubCategory     string
	Categories      []models.SchematicCategory
	Avatar          template.URL
	HasAvatar       bool
	IsContributor   bool
	Language        string
	LangPrefix      string
	CanonicalURL    string
	PrevPageURL     string
	NextPageURL     string
	NoIndex         bool
}

func (d *DefaultData) Populate(e *server.RequestEvent) {
	// Language from URL prefix takes precedence (set by lang middleware)
	if lang := e.Request.Header.Get("X-Createmod-Lang"); lang != "" && isSupportedLanguage(lang) {
		d.Language = lang
	} else {
		// Fallback: cookie / Accept-Language
		d.Language = preferredLanguageFromRequest(e.Request)
	}
	d.LangPrefix = LangToPrefix[d.Language]

	// Populate from PostgreSQL session (set by cookieAuth middleware)
	if sessUser := session.UserFromContext(e.Request.Context()); sessUser != nil {
		d.populateFromSession(e, sessUser)
	}
}

// PopulateWithStore is like Populate but also checks contributor status via the store.
func (d *DefaultData) PopulateWithStore(e *server.RequestEvent, appStore *store.Store) {
	d.Populate(e)
	if d.IsAuthenticated && appStore != nil {
		d.IsContributor = isContributorFromStore(appStore, d.UserID)
	}
}

// populateFromSession fills DefaultData from a PostgreSQL session user.
func (d *DefaultData) populateFromSession(e *server.RequestEvent, user *session.SessionUser) {
	d.IsAuthenticated = true
	caser := cases.Title(language.English)
	d.Username = caser.String(user.Username)
	d.UserID = user.ID
	d.UsernameSlug = strings.ToLower(user.Username)
	if user.Avatar != "" {
		d.Avatar = template.URL(user.Avatar)
	} else {
		url := gravatar.New(user.Email).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		d.Avatar = template.URL(url)
	}
	d.HasAvatar = d.Avatar != ""
	// Contributor status - check has no direct store access here, so left for handler to set
	// TODO: This will be set by handlers with store access
}

// isAuthenticated returns true if the request is authenticated via the PostgreSQL session store.
func isAuthenticated(e *server.RequestEvent) bool {
	return session.UserFromContext(e.Request.Context()) != nil
}

// authenticatedUserID returns the authenticated user's ID from the session.
// Returns empty string if not authenticated.
func authenticatedUserID(e *server.RequestEvent) string {
	if u := session.UserFromContext(e.Request.Context()); u != nil {
		return u.ID
	}
	return ""
}

// authenticatedUserEmail returns the authenticated user's email from the session.
// Returns empty string if not authenticated.
func authenticatedUserEmail(e *server.RequestEvent) string {
	if u := session.UserFromContext(e.Request.Context()); u != nil {
		return u.Email
	}
	return ""
}

// requireAuth checks if the user is authenticated and redirects to /login if not.
// Returns true if the user IS authenticated, false if a redirect was sent.
func requireAuth(e *server.RequestEvent) (bool, error) {
	if isAuthenticated(e) {
		return true, nil
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/login"))
		return false, e.HTML(http.StatusNoContent, "")
	}
	return false, e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/login"))
}


func allCategoriesFromStoreOnly(appStore *store.Store, cacheService *cache.Service) []models.SchematicCategory {
	categories, found := cacheService.GetCategories(cache.AllCategoriesKey)
	if found {
		return categories
	}
	cats, err := appStore.Categories.List(stdctx.Background())
	if err != nil {
		return nil
	}
	var result []models.SchematicCategory
	for _, c := range cats {
		result = append(result, models.SchematicCategory{
			ID:   c.ID,
			Key:  c.Key,
			Name: c.Name,
		})
	}
	cacheService.SetCategories(cache.AllCategoriesKey, result, time.Hour*730)
	return result
}

func isContributorFromStore(appStore *store.Store, userID string) bool {
	if userID == "" || appStore == nil {
		return false
	}
	contrib, err := appStore.Users.IsContributor(stdctx.Background(), userID)
	if err != nil {
		return false
	}
	return contrib
}
