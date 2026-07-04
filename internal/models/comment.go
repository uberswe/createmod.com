package models

import (
	"html/template"
	"time"
)

type Comment struct {
	ID              string
	Created         string `json:"-"`
	Published       string
	Author          string `json:"-"`
	AuthorID        string `json:"-"`
	AuthorUsername  string
	AuthorHasAvatar bool `json:"-"`
	AuthorAvatar    string
	Indent          int
	Content         template.HTML
	OriginalContent template.HTML `json:"-"`
	Approved        bool          `json:"-"`
	ParentID        string
	ReplyToAuthor   string
	IsTranslated    bool `json:"-"`
}

type DatabaseComment struct {
	ID        string
	Created   time.Time
	Published string
	Author    string
	Schematic string
	Karma     int
	Approved  bool
	Type      string
	ParentID  string
	Content   string
}
