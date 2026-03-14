package models

import (
	"html/template"
)

type User struct {
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Avatar    template.URL `json:"avatar"`
	HasAvatar bool         `json:"hasAvatar"`
}
