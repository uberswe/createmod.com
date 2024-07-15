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

type DatabaseSchematic struct {
	ID               string
	Created          string
	Author           *User
	CommentCount     int
	Content          string
	HTMLContent      template.HTML
	Excerpt          string
	FeaturedImage    string
	HasGallery       bool
	Title            string
	Name             string
	Video            string
	HasDependencies  bool
	Dependencies     string
	HTMLDependencies template.HTML
	CreatemodVersion string
	MinecraftVersion string
	Views            int
	HasTags          bool
	Rating           string
	HasRating        bool
}

func (d *DatabaseSchematic) ToSchematic() Schematic {
	created, _ := time.Parse(time.RFC3339, d.Created)
	return Schematic{
		ID:               d.ID,
		Created:          created,
		Author:           d.Author,
		CommentCount:     d.CommentCount,
		Content:          d.Content,
		HTMLContent:      d.HTMLContent,
		Excerpt:          d.Excerpt,
		FeaturedImage:    d.FeaturedImage,
		HasGallery:       d.HasGallery,
		Title:            d.Title,
		Name:             d.Name,
		Video:            d.Video,
		HasDependencies:  d.HasDependencies,
		Dependencies:     d.Dependencies,
		HTMLDependencies: d.HTMLDependencies,
		CreatemodVersion: d.CreatemodVersion,
		MinecraftVersion: d.MinecraftVersion,
		Views:            d.Views,
		HasTags:          d.HasTags,
		Rating:           d.Rating,
		HasRating:        d.HasRating,
	}
}

func DatabaseSchematicsToSchematics(databaseSchematics []DatabaseSchematic) (res []Schematic) {
	res = make([]Schematic, 0)
	for _, dbs := range databaseSchematics {
		res = append(res, dbs.ToSchematic())
	}
	return res
}
