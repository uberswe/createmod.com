package pages

import (
	"strings"
	"testing"
)

func Test_Header_Language_Dropdown_And_Links(t *testing.T) {
	d := DefaultData{
		IsAuthenticated: false,
		Language:        "en",
		Slug:            "/videos",
		Title:           "Videos",
	}
	html := renderTemplate(t, "template/include/header.html", d)

	// Shows current language flag in dropdown trigger
	if !strings.Contains(html, "title=\"en\"") {
		t.Fatalf("header should display current language flag with title='en'")
	}

	// Language switcher now uses direct subdirectory links instead of /lang?l=...
	// For English, links go to root paths; for other languages, they get prefixed.
	// With Slug="/videos", English link should be "/videos" and German should be "/de/videos".
	langExpected := map[string]string{
		"en":      "/videos",
		"de":      "/de/videos",
		"es":      "/es/videos",
		"pl":      "/pl/videos",
		"pt-BR":   "/pt-br/videos",
		"pt-PT":   "/pt-pt/videos",
		"ru":      "/ru/videos",
		"zh-Hans": "/zh/videos",
	}
	for lang, expectedHref := range langExpected {
		if !strings.Contains(html, expectedHref) {
			t.Fatalf("header language dropdown missing link for %s (expected href containing %q)", lang, expectedHref)
		}
	}
}
