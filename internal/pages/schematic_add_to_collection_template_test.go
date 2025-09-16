package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Schematic_Template_Has_AddToCollection_Form(t *testing.T) {
	path := filepath.Join("..", "..", "template", "schematic.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"action=\"/schematics/{{ .Schematic.Name }}/add-to-collection\"",
		"name=\"collection\"",
		"Add to collection",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("schematic.html missing: %s", m)
		}
	}
}
