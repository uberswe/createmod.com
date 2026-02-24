package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Schematic_Template_Paid_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "schematic.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	// Expect Paid badge and Get Schematic conditional text
	if !strings.Contains(s, "badge") || !strings.Contains(s, "Paid") {
		t.Fatalf("schematic.html should contain a Paid badge marker")
	}
	if !strings.Contains(s, "Get Schematic") {
		t.Fatalf("schematic.html should contain 'Get Schematic' for paid items")
	}
}

func Test_Include_Cards_Paid_Badge(t *testing.T) {
	// schematic_card.html is the base card that contains Paid badge markup.
	// schematic_card_full.html and others delegate to it via {{template}}.
	p := filepath.Join("..", "..", "template", "include", "schematic_card.html")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	if !strings.Contains(s, "Paid") || !strings.Contains(s, "badge") {
		t.Fatalf("%s should contain a Paid badge marker", p)
	}
	// Verify full card delegates to schematic_card.html
	fp := filepath.Join("..", "..", "template", "include", "schematic_card_full.html")
	fb, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("read %s: %v", fp, err)
	}
	fs := string(fb)
	if !strings.Contains(fs, `schematic_card.html`) {
		t.Fatalf("%s should reference schematic_card.html", fp)
	}
}
