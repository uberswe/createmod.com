package models

import "time"

// NewsPostListItem is a minimal view model for listing news posts on /news
// It contains only the fields needed by the template.
type NewsPostListItem struct {
	ID       string
	Title    string
	Excerpt  string
	URL      string
	PostDate time.Time
}
