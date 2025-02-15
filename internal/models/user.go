package models

import (
	"html/template"
)

type User struct {
	ID        string
	Username  string
	Avatar    template.URL
	HasAvatar bool
}
