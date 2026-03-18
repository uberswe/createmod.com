package pages

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"createmod/internal/cache"
	"createmod/internal/server"
	"createmod/internal/store"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	DcNS    string     `xml:"xmlns:dc,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	AtomLink      atomLink  `xml:"atom link"`
	Items         []rssItem `xml:"item"`
}

type atomLink struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	Creator     string  `xml:"dc:creator,omitempty"`
	PubDate     string  `xml:"pubDate"`
	GUID        rssGUID `xml:"guid"`
}

// relURLRe matches href="..." and src="..." attribute values in HTML.
var relURLRe = regexp.MustCompile(`((?:href|src)\s*=\s*")([^"]+)`)

// absifyURLs rewrites relative URLs in HTML content to absolute URLs.
// URLs that already start with http://, https://, //, /, mailto:, or # are left unchanged.
func absifyURLs(html, baseURL string) string {
	return relURLRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := relURLRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		prefix, val := parts[1], parts[2]
		if len(val) == 0 ||
			val[0] == '/' ||
			val[0] == '#' ||
			len(val) > 7 && (val[:7] == "http://" || val[:8] == "https://") ||
			len(val) > 2 && val[:2] == "//" ||
			len(val) > 7 && val[:7] == "mailto:" {
			return match
		}
		return prefix + baseURL + "/" + val
	})
}

const rssFeedCacheKey = "rss_feed"

// RSSFeedHandler serves an RSS 2.0 feed of the latest approved schematics.
func RSSFeedHandler(appStore *store.Store, cacheService *cache.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Check cache first
		if cached, ok := cacheService.Get(rssFeedCacheKey); ok {
			if data, ok := cached.([]byte); ok {
				e.Response.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
				e.Response.Header().Set("Cache-Control", "public, max-age=3600")
				_, err := e.Response.Write(data)
				return err
			}
		}

		// Fetch latest 50 approved schematics
		ctx := e.Request.Context()
		schematics, err := appStore.Schematics.ListApproved(ctx, 50, 0)
		if err != nil {
			slog.Error("RSS feed: failed to list schematics", "error", err)
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to generate feed"}
		}

		// Build RSS items
		items := make([]rssItem, 0, len(schematics))
		for _, s := range schematics {
			pubDate := s.Created.UTC().Format(time.RFC1123Z)
			if s.Postdate != nil {
				pubDate = s.Postdate.UTC().Format(time.RFC1123Z)
			}

			description := s.Excerpt
			if description == "" && len(s.Description) > 0 {
				description = s.Description
			}
			description = absifyURLs(description, "https://createmod.com")

			// Look up author username
			authorName := ""
			if s.AuthorID != "" {
				if u, uErr := appStore.Users.GetUserByID(ctx, s.AuthorID); uErr == nil && u != nil {
					authorName = u.Username
				}
			}

			items = append(items, rssItem{
				Title:       s.Title,
				Link:        "https://createmod.com/schematics/" + s.Name,
				Description: description,
				Creator:     authorName,
				PubDate:     pubDate,
				GUID: rssGUID{
					IsPermaLink: "false",
					Value:       s.ID,
				},
			})
		}

		lastBuild := time.Now().UTC().Format(time.RFC1123Z)
		if len(schematics) > 0 {
			t := schematics[0].Created
			if schematics[0].Postdate != nil {
				t = *schematics[0].Postdate
			}
			lastBuild = t.UTC().Format(time.RFC1123Z)
		}

		feed := rssFeed{
			Version: "2.0",
			AtomNS:  "http://www.w3.org/2005/Atom",
			DcNS:    "http://purl.org/dc/elements/1.1/",
			Channel: rssChannel{
				Title:         "CreateMod.com - Latest Schematics",
				Link:          "https://createmod.com",
				Description:   "The latest community-built schematics for Minecraft's Create mod",
				Language:      "en",
				LastBuildDate: lastBuild,
				AtomLink: atomLink{
					Href: "https://createmod.com/feed.xml",
					Rel:  "self",
					Type: "application/rss+xml",
				},
				Items: items,
			},
		}

		xmlData, err := renderRSSFeed(feed)
		if err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to marshal feed"}
		}

		// Cache for 1 hour
		cacheService.SetWithTTL(rssFeedCacheKey, xmlData, 1*time.Hour)
		return writeRSSResponse(e, xmlData)
	}
}

// renderRSSFeed marshals an rssFeed to XML with proper namespace prefixes.
func renderRSSFeed(feed rssFeed) ([]byte, error) {
	data, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}
	xmlData := append([]byte(xml.Header), data...)

	// Fix Go encoding/xml namespace output to use prefixed form for validators.
	xmlData = bytes.ReplaceAll(xmlData, []byte(`<link xmlns="http://www.w3.org/2005/Atom"`), []byte(`<atom:link`))
	xmlData = bytes.ReplaceAll(xmlData, []byte(`></link>`), []byte(` />`))
	xmlData = bytes.ReplaceAll(xmlData, []byte(`<creator xmlns="http://purl.org/dc/elements/1.1/">`), []byte(`<dc:creator>`))
	xmlData = bytes.ReplaceAll(xmlData, []byte(`</creator>`), []byte(`</dc:creator>`))
	return xmlData, nil
}

// writeRSSResponse writes RSS XML data to the response with appropriate headers.
func writeRSSResponse(e *server.RequestEvent, xmlData []byte) error {
	e.Response.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	e.Response.Header().Set("Cache-Control", "public, max-age=3600")
	_, err := e.Response.Write(xmlData)
	return err
}

// AuthorFeedHandler serves an RSS 2.0 feed of a specific author's schematics.
// URL pattern: GET /author/{username}/feed
func AuthorFeedHandler(appStore *store.Store, cacheService *cache.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			return &server.APIError{Status: http.StatusNotFound, Message: "Not found"}
		}

		cacheKey := "rss_author_" + username
		if cached, ok := cacheService.Get(cacheKey); ok {
			if data, ok := cached.([]byte); ok {
				return writeRSSResponse(e, data)
			}
		}

		ctx := e.Request.Context()
		user, err := appStore.Users.GetUserByUsername(ctx, username)
		if err != nil || user == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "Author not found"}
		}

		schematics, err := appStore.Schematics.ListByAuthor(ctx, user.ID, 50, 0)
		if err != nil {
			slog.Error("author RSS feed: failed to list schematics", "error", err, "username", username)
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to generate feed"}
		}

		items := make([]rssItem, 0, len(schematics))
		for _, s := range schematics {
			pubDate := s.Created.UTC().Format(time.RFC1123Z)
			if s.Postdate != nil {
				pubDate = s.Postdate.UTC().Format(time.RFC1123Z)
			}
			description := s.Excerpt
			if description == "" && len(s.Description) > 0 {
				description = s.Description
			}
			description = absifyURLs(description, "https://createmod.com")

			items = append(items, rssItem{
				Title:       s.Title,
				Link:        "https://createmod.com/schematics/" + s.Name,
				Description: description,
				Creator:     user.Username,
				PubDate:     pubDate,
				GUID:        rssGUID{IsPermaLink: "false", Value: s.ID},
			})
		}

		lastBuild := time.Now().UTC().Format(time.RFC1123Z)
		if len(schematics) > 0 {
			t := schematics[0].Created
			if schematics[0].Postdate != nil {
				t = *schematics[0].Postdate
			}
			lastBuild = t.UTC().Format(time.RFC1123Z)
		}

		selfURL := fmt.Sprintf("https://createmod.com/author/%s/feed", username)
		feed := rssFeed{
			Version: "2.0",
			AtomNS:  "http://www.w3.org/2005/Atom",
			DcNS:    "http://purl.org/dc/elements/1.1/",
			Channel: rssChannel{
				Title:         fmt.Sprintf("CreateMod.com - Schematics by %s", user.Username),
				Link:          fmt.Sprintf("https://createmod.com/author/%s", username),
				Description:   fmt.Sprintf("Latest schematics by %s on CreateMod.com", user.Username),
				Language:      "en",
				LastBuildDate: lastBuild,
				AtomLink:      atomLink{Href: selfURL, Rel: "self", Type: "application/rss+xml"},
				Items:         items,
			},
		}

		xmlData, err := renderRSSFeed(feed)
		if err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to marshal feed"}
		}
		cacheService.SetWithTTL(cacheKey, xmlData, 1*time.Hour)
		return writeRSSResponse(e, xmlData)
	}
}

// SchematicFeedHandler serves an RSS 2.0 feed of comments on a specific schematic.
// URL pattern: GET /schematics/{name}/feed
func SchematicFeedHandler(appStore *store.Store, cacheService *cache.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return &server.APIError{Status: http.StatusNotFound, Message: "Not found"}
		}

		cacheKey := "rss_schematic_" + name
		if cached, ok := cacheService.Get(cacheKey); ok {
			if data, ok := cached.([]byte); ok {
				return writeRSSResponse(e, data)
			}
		}

		ctx := e.Request.Context()
		schematic, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || schematic == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "Schematic not found"}
		}

		comments, err := appStore.Comments.ListBySchematic(ctx, schematic.ID)
		if err != nil {
			slog.Error("schematic RSS feed: failed to list comments", "error", err, "name", name)
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to generate feed"}
		}

		items := make([]rssItem, 0, len(comments))
		for _, c := range comments {
			pubDate := c.Created.UTC().Format(time.RFC1123Z)
			if c.Published != nil {
				pubDate = c.Published.UTC().Format(time.RFC1123Z)
			}
			title := fmt.Sprintf("Comment by %s", c.AuthorUsername)
			if c.AuthorUsername == "" {
				title = "Comment"
			}
			items = append(items, rssItem{
				Title:       title,
				Link:        fmt.Sprintf("https://createmod.com/schematics/%s#comment-%s", name, c.ID),
				Description: c.Content,
				Creator:     c.AuthorUsername,
				PubDate:     pubDate,
				GUID:        rssGUID{IsPermaLink: "false", Value: c.ID},
			})
		}

		lastBuild := time.Now().UTC().Format(time.RFC1123Z)
		if len(comments) > 0 {
			last := comments[len(comments)-1]
			t := last.Created
			if last.Published != nil {
				t = *last.Published
			}
			lastBuild = t.UTC().Format(time.RFC1123Z)
		}

		selfURL := fmt.Sprintf("https://createmod.com/schematics/%s/feed", name)
		feed := rssFeed{
			Version: "2.0",
			AtomNS:  "http://www.w3.org/2005/Atom",
			DcNS:    "http://purl.org/dc/elements/1.1/",
			Channel: rssChannel{
				Title:         fmt.Sprintf("CreateMod.com - Comments on %s", schematic.Title),
				Link:          fmt.Sprintf("https://createmod.com/schematics/%s", name),
				Description:   fmt.Sprintf("Latest comments on %s", schematic.Title),
				Language:      "en",
				LastBuildDate: lastBuild,
				AtomLink:      atomLink{Href: selfURL, Rel: "self", Type: "application/rss+xml"},
				Items:         items,
			},
		}

		xmlData, err := renderRSSFeed(feed)
		if err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to marshal feed"}
		}
		cacheService.SetWithTTL(cacheKey, xmlData, 1*time.Hour)
		return writeRSSResponse(e, xmlData)
	}
}

// pingWebSub notifies a WebSub hub that the feed has new content.
func pingWebSub(hubURL, feedURL string) {
	resp, err := http.PostForm(hubURL, url.Values{
		"hub.mode": {"publish"},
		"hub.url":  {feedURL},
	})
	if err != nil {
		slog.Warn("WebSub ping failed", "hub", hubURL, "error", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		slog.Warn("WebSub ping unexpected status", "hub", hubURL, "status", resp.StatusCode)
	}
}

// pingPingomatic notifies Pingomatic that the site has new content via XML-RPC.
func pingPingomatic(siteTitle, siteURL, feedURL string) {
	body := `<?xml version="1.0"?>
<methodCall>
  <methodName>weblogUpdates.ping</methodName>
  <params>
    <param><value>` + siteTitle + `</value></param>
    <param><value>` + siteURL + `</value></param>
    <param><value>` + feedURL + `</value></param>
  </params>
</methodCall>`
	resp, err := http.Post("https://rpc.pingomatic.com/", "text/xml", bytes.NewBufferString(body))
	if err != nil {
		slog.Warn("Pingomatic ping failed", "error", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("Pingomatic ping unexpected status", "status", resp.StatusCode)
	}
}

// PingFeedServicesAsync pings WebSub hubs and Pingomatic in background goroutines.
// Only call this in production (non-dev) environments.
func PingFeedServicesAsync() {
	feedURL := "https://createmod.com/feed.xml"
	go pingWebSub("https://pubsubhubbub.appspot.com/", feedURL)
	go pingWebSub("https://pubsubhubbub.superfeedr.com/", feedURL)
	go pingPingomatic("CreateMod.com", "https://createmod.com", feedURL)
}

