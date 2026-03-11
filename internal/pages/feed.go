package pages

import (
	"createmod/internal/cache"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/xml"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	AtomLink      atomLink  `xml:"atom:link"`
	Items         []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
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
				Author:      authorName,
				PubDate:     pubDate,
				GUID:        s.ID,
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
			Atom:    "http://www.w3.org/2005/Atom",
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

		data, err := xml.MarshalIndent(feed, "", "  ")
		if err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "Failed to marshal feed"}
		}

		// Prepend XML declaration
		xmlData := append([]byte(xml.Header), data...)

		// Cache for 1 hour
		cacheService.SetWithTTL(rssFeedCacheKey, xmlData, 1*time.Hour)

		e.Response.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		e.Response.Header().Set("Cache-Control", "public, max-age=3600")
		_, writeErr := e.Response.Write(xmlData)
		return writeErr
	}
}

// pingWebSub notifies Google's WebSub hub that the feed has new content.
func pingWebSub(feedURL string) {
	resp, err := http.PostForm("https://pubsubhubbub.appspot.com/", url.Values{
		"hub.mode": {"publish"},
		"hub.url":  {feedURL},
	})
	if err != nil {
		slog.Warn("WebSub ping failed", "error", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		slog.Warn("WebSub ping unexpected status", "status", resp.StatusCode)
	}
}

// PingWebSubAsync pings the WebSub hub in a background goroutine.
// Only call this in production (non-dev) environments.
func PingWebSubAsync() {
	go pingWebSub("https://createmod.com/feed.xml")
}

