package pages

import (
	stdctx "context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/session"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/drexedam/gravatar"
	"createmod/internal/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"html/template"
	"net/http"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// BreadcrumbItem represents a single item in the breadcrumb trail.
type BreadcrumbItem struct {
	Label string // display text (translated by the handler)
	URL   string // empty string = active/current page (last item)
}

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
	Breadcrumbs     []BreadcrumbItem
}

// NewBreadcrumbs builds a breadcrumb trail starting with Home.
// Arguments are pairs of (label, url) followed by a final label for the active page.
// Example: NewBreadcrumbs("en", "Schematics", "/schematics", "My Build")
// produces: [Home(/), Schematics(/schematics), My Build(active)]
func NewBreadcrumbs(lang string, items ...string) []BreadcrumbItem {
	home := i18n.T(lang, "Home")
	if home == "" || home == "Home" {
		home = "Home"
	}
	crumbs := []BreadcrumbItem{{Label: home, URL: "/"}}

	// Process pairs: (label, url), (label, url), ..., final label
	for i := 0; i < len(items); i++ {
		if i+1 < len(items) {
			// This is a (label, url) pair
			crumbs = append(crumbs, BreadcrumbItem{Label: items[i], URL: items[i+1]})
			i++ // skip URL
		} else {
			// Last item: active page (no URL)
			crumbs = append(crumbs, BreadcrumbItem{Label: items[i]})
		}
	}
	return crumbs
}

// BreadcrumbJSONLD returns a <script type="application/ld+json"> block with
// Schema.org BreadcrumbList markup. Returns empty template.HTML if no breadcrumbs.
func (d *DefaultData) BreadcrumbJSONLD() template.HTML {
	if len(d.Breadcrumbs) == 0 {
		return ""
	}
	type listItem struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item,omitempty"`
	}
	type breadcrumbList struct {
		Context  string     `json:"@context"`
		Type     string     `json:"@type"`
		ItemList []listItem `json:"itemListElement"`
	}

	bl := breadcrumbList{
		Context:  "https://schema.org",
		Type:     "BreadcrumbList",
		ItemList: make([]listItem, 0, len(d.Breadcrumbs)),
	}
	for i, bc := range d.Breadcrumbs {
		li := listItem{
			Type:     "ListItem",
			Position: i + 1,
			Name:     bc.Label,
		}
		if bc.URL != "" {
			li.Item = fmt.Sprintf("https://createmod.com%s", PrefixedPath(d.Language, bc.URL))
		}
		bl.ItemList = append(bl.ItemList, li)
	}
	data, err := json.Marshal(bl)
	if err != nil {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, data))
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

// safeRedirectPath validates a return_to URL parameter to prevent open redirects.
// Returns the path if it is a safe, relative path; otherwise returns fallback.
func safeRedirectPath(returnTo, fallback string) string {
	returnTo = strings.TrimSpace(returnTo)
	if returnTo == "" {
		return fallback
	}
	// Must start with /
	if !strings.HasPrefix(returnTo, "/") {
		return fallback
	}
	// Block protocol-relative URLs (e.g. //evil.com)
	if strings.HasPrefix(returnTo, "//") {
		return fallback
	}
	// Block URLs with scheme-like patterns after the slash
	if idx := strings.Index(returnTo, ":"); idx > 0 && idx < strings.Index(returnTo+"?", "?") {
		// Check if the colon comes before any slash after the first character
		afterFirst := returnTo[1:]
		slashIdx := strings.Index(afterFirst, "/")
		colonIdx := strings.Index(afterFirst, ":")
		if colonIdx >= 0 && (slashIdx < 0 || colonIdx < slashIdx) {
			return fallback
		}
	}
	return returnTo
}

// safeFilenameRe matches only ASCII letters, digits, hyphens, and underscores.
var safeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

// sanitizeFilename produces a URL-safe, ASCII-only filename from a user-provided
// filename. It preserves the file extension and replaces non-ASCII / special
// characters with underscores. If the base name ends up empty after sanitization
// (e.g. a purely non-Latin filename), a random hex string is used instead.
func sanitizeFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace spaces with underscores, then strip everything non-ASCII-safe
	base = strings.ReplaceAll(base, " ", "_")
	base = safeFilenameRe.ReplaceAllString(base, "")

	// Collapse multiple underscores and trim leading/trailing underscores
	for strings.Contains(base, "__") {
		base = strings.ReplaceAll(base, "__", "_")
	}
	base = strings.Trim(base, "_")

	// If nothing remains, generate a random name
	if base == "" {
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		base = hex.EncodeToString(buf)
	}

	return base + ext
}

// sanitizeContentDispositionFilename strips characters that could cause header injection
// in Content-Disposition filenames.
func sanitizeContentDispositionFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "download"
	}
	// Remove characters that could break the header or enable injection
	replacer := strings.NewReplacer(
		"\"", "",
		"\\", "",
		"\r", "",
		"\n", "",
		"\x00", "",
	)
	filename = replacer.Replace(filename)
	if filename == "" {
		return "download"
	}
	return filename
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

// adminRecipients returns mail.Address entries for all admin users. Falls back
// to the mailer's configured sender address if no admins are found in the DB.
func adminRecipients(appStore *store.Store, mailService interface{ DefaultFrom() mail.Address }) []mail.Address {
	if appStore != nil {
		emails, err := appStore.Users.ListAdminEmails(stdctx.Background())
		if err == nil && len(emails) > 0 {
			addrs := make([]mail.Address, len(emails))
			for i, e := range emails {
				addrs[i] = mail.Address{Address: e}
			}
			return addrs
		}
	}
	// Fallback: use the sender address itself
	from := mailService.DefaultFrom()
	if from.Address != "" {
		return []mail.Address{from}
	}
	return nil
}
