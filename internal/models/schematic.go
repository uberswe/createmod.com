package models

import (
	"html/template"
	"time"
)

type Schematic struct {
	ID               string
	Created          time.Time
	Author           *User
	CommentCount     int
	CommentStatus    bool
	Content          string
	HTMLContent      template.HTML
	Excerpt          string
	FeaturedImage    string
	HasGallery       bool
	Gallery          []string
	Title            string
	Name             string
	Video            string
	HasDependencies  bool
	Dependencies     string
	HTMLDependencies template.HTML
	Categories       []SchematicCategory
	Tags             []SchematicTag
	CreatemodVersion string
	MinecraftVersion string
	Views            int
	HasTags          bool
	Rating           string
	HasRating        bool
}
