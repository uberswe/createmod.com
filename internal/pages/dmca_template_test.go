package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_DMCA_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "dmca.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"DMCA Takedown Requests",
		`id="dmca-form"`,
		`name="name"`,
		`name="company"`,
		`name="email"`,
		`name="copyright_holder"`,
		`name="work"`,
		`name="urls"`,
		`name="details"`,
		`name="good_faith"`,
		`name="accuracy"`,
		`name="signature"`,
		"'/api/dmca'",
		"512(c)(3)",
		"512(f)",
		`id="dmca-success"`,
		`id="dmca-error"`,
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("dmca.html missing: %s", m)
		}
	}
}

func Test_Footer_Has_DMCA_Link(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "footer.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(b), `"/dmca"`) {
		t.Fatal("footer.html missing DMCA link")
	}
}

func Test_SearchFilters_Have_Submit_Button(t *testing.T) {
	for _, name := range []string{"search_filters.html", "mod_filters.html"} {
		path := filepath.Join("..", "..", "template", "include", name)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		s := string(b)
		if !strings.Contains(s, `btn-filter-search`) || !strings.Contains(s, `type="submit"`) {
			t.Fatalf("%s missing clickable search submit button", name)
		}
	}
}
