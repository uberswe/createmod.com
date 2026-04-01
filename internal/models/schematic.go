package models

import (
	"html/template"
	"time"
)

type Schematic struct {
	ID                   string             `json:"id"`
	Created              time.Time          `json:"created"`
	CreatedFormatted     string             `json:"createdFormatted"`
	CreatedHumanReadable string             `json:"createdHumanReadable"`
	Author               *User              `json:"author"`
	CommentCount         int                `json:"commentCount"`
	CommentStatus        bool               `json:"commentStatus"`
	Content              string             `json:"content"`
	HTMLContent          template.HTML      `json:"htmlContent"`
	Excerpt              string             `json:"excerpt"`
	FeaturedImage        string             `json:"featuredImage"`
	HasGallery           bool               `json:"hasGallery"`
	Gallery              []string           `json:"gallery"`
	Title                string             `json:"title"`
	Name                 string             `json:"name"`
	Video                string             `json:"video"`
	HasDependencies      bool               `json:"hasDependencies"`
	Dependencies         string             `json:"dependencies"`
	HTMLDependencies     template.HTML      `json:"htmlDependencies"`
	Categories           []SchematicCategory `json:"categories"`
	CategoryId           string             `json:"categoryId"`
	Tags                 []SchematicTag     `json:"tags"`
	CreatemodVersion     string             `json:"createmodVersion"`
	MinecraftVersion     string             `json:"minecraftVersion"`
	Views                int                `json:"views"`
	Downloads            int                `json:"downloads"`
	HasTags              bool               `json:"hasTags"`
	Rating               string             `json:"rating"`
	HasRating            bool               `json:"hasRating"`
	SchematicFile        string             `json:"schematicFile"`
	RatingCount          int                `json:"ratingCount"`
	AIDescription        string             `json:"aiDescription"`
	Paid                 bool               `json:"paid"`
	Featured             bool               `json:"featured"`
	Materials            string             `json:"materials"`
	ExternalURL          string             `json:"externalURL"`
	BlockCount           int                `json:"blockCount"`
	DimX                 int                `json:"dimX"`
	DimY                 int                `json:"dimY"`
	DimZ                 int                `json:"dimZ"`
	Mods                 []string           `json:"mods"`
	DetectedLanguage     string             `json:"-"`
	ModerationState      string             `json:"-"`
}

// ModerationChatMessage holds a single message in the moderation discussion thread.
type ModerationChatMessage struct {
	ID           string
	AuthorName   string
	AuthorAvatar string
	IsModerator  bool
	Body         string
	Created      string // human-readable formatted time
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
	AvgRating        *float64
	SchematicFile    string
	AIDescription    string
	Featured         bool
}
