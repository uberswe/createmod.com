package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Similar_Templates_Have_Sort_Dropdown(t *testing.T) {
	for tmpl, must := range map[string][]string{
		"similar.html": {
			"sim-sort",
			"form-select",
			"data-slug",
			"{{ range .SortKeys }}",
		},
		"similar_tool.html": {
			"sim-sort",
			"form-select",
			"value=\"overall\"",
			"value=\"shape\"",
			"value=\"materials\"",
			"value=\"function\"",
			"value=\"proportions\"",
			"value=\"palette\"",
			"renderResults",
		},
	} {
		path := filepath.Join("..", "..", "template", tmpl)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		s := string(b)
		for _, m := range must {
			if !strings.Contains(s, m) {
				t.Errorf("%s missing: %s", tmpl, m)
			}
		}
	}
}
