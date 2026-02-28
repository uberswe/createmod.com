package models

import (
	"html/template"
	"time"
)

// NewsPostListItem is a minimal view model for listing news posts on /news
// It contains only the fields needed by the template.
type NewsPostListItem struct {
	ID             string
	Title          string
	Excerpt        string
	FirstParagraph template.HTML // rendered HTML of the first paragraph
	URL            string
	PostDate       time.Time
}
