package models

import (
	"html/template"
	"time"
)

type Comment struct {
	ID             string
	Created        string
	Author         string
	AuthorUsername string
	Content        template.HTML
	Approved       bool
	ParentID       string
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
