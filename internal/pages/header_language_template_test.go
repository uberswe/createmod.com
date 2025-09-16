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

	// Shows current language code in dropdown trigger
	if !strings.Contains(html, ">en<") {
		t.Fatalf("header should display current language 'en'")
	}

	// Contains language options and return_to param; be tolerant to HTML escaping
	langs := []string{"en", "pt-BR", "pt-PT", "es", "de", "pl", "ru", "zh-Hans"}
	for _, l := range langs {
		if !strings.Contains(html, "/lang?l="+l) {
			t.Fatalf("header language dropdown missing base link for %s", l)
		}
	}
	if !strings.Contains(html, "return_to=") {
		t.Fatalf("header language dropdown should include return_to param")
	}
}
