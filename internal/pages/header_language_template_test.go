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

	// Language switcher uses /lang endpoint to set the cm_lang cookie before redirecting.
	// Go's html/template URL-normalizes href values, encoding / as %2f in query params.
	langExpected := []string{
		"/lang?l=en&return_to=%2fvideos",
		"/lang?l=de&return_to=%2fvideos",
		"/lang?l=es&return_to=%2fvideos",
		"/lang?l=pl&return_to=%2fvideos",
		"/lang?l=pt-BR&return_to=%2fvideos",
		"/lang?l=pt-PT&return_to=%2fvideos",
		"/lang?l=ru&return_to=%2fvideos",
		"/lang?l=zh-Hans&return_to=%2fvideos",
	}
	for _, expectedHref := range langExpected {
		if !strings.Contains(html, expectedHref) {
			t.Fatalf("header language dropdown missing link %q", expectedHref)
		}
	}
}
