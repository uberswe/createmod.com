package pages

import (
	"strings"
	"testing"
)

func Test_NewBreadcrumbs_SingleLevel(t *testing.T) {
	bc := NewBreadcrumbs("en", "Schematics")
	if len(bc) != 2 {
		t.Fatalf("expected 2 items, got %d", len(bc))
	}
	if bc[0].Label != "Home" || bc[0].URL != "/" {
		t.Fatalf("first item should be Home(/), got %q(%q)", bc[0].Label, bc[0].URL)
	}
	if bc[1].Label != "Schematics" || bc[1].URL != "" {
		t.Fatalf("second item should be Schematics(active), got %q(%q)", bc[1].Label, bc[1].URL)
	}
}

func Test_NewBreadcrumbs_TwoLevels(t *testing.T) {
	bc := NewBreadcrumbs("en", "Schematics", "/schematics", "My Build")
	if len(bc) != 3 {
		t.Fatalf("expected 3 items, got %d", len(bc))
	}
	if bc[1].Label != "Schematics" || bc[1].URL != "/schematics" {
		t.Fatalf("second item should be Schematics(/schematics), got %q(%q)", bc[1].Label, bc[1].URL)
	}
	if bc[2].Label != "My Build" || bc[2].URL != "" {
		t.Fatalf("third item should be My Build(active), got %q(%q)", bc[2].Label, bc[2].URL)
	}
}

func Test_NewBreadcrumbs_ThreeLevels(t *testing.T) {
	bc := NewBreadcrumbs("en", "Schematics", "/schematics", "My Build", "/schematics/my-build", "Edit")
	if len(bc) != 4 {
		t.Fatalf("expected 4 items, got %d", len(bc))
	}
	if bc[0].Label != "Home" || bc[0].URL != "/" {
		t.Fatalf("first item wrong: %q(%q)", bc[0].Label, bc[0].URL)
	}
	if bc[1].Label != "Schematics" || bc[1].URL != "/schematics" {
		t.Fatalf("second item wrong: %q(%q)", bc[1].Label, bc[1].URL)
	}
	if bc[2].Label != "My Build" || bc[2].URL != "/schematics/my-build" {
		t.Fatalf("third item wrong: %q(%q)", bc[2].Label, bc[2].URL)
	}
	if bc[3].Label != "Edit" || bc[3].URL != "" {
		t.Fatalf("fourth item should be Edit(active), got %q(%q)", bc[3].Label, bc[3].URL)
	}
}

func Test_NewBreadcrumbs_HomeAlwaysFirst(t *testing.T) {
	bc := NewBreadcrumbs("en", "Rules")
	if len(bc) < 1 {
		t.Fatal("expected at least 1 item")
	}
	if bc[0].URL != "/" {
		t.Fatalf("Home URL should be /, got %q", bc[0].URL)
	}
}

func Test_NewBreadcrumbs_LastItemAlwaysActive(t *testing.T) {
	tests := []struct {
		name  string
		items []string
	}{
		{"one level", []string{"Contact"}},
		{"two levels", []string{"Settings", "/settings", "Password"}},
		{"three levels", []string{"Admin", "/admin", "Schematics", "/admin/schematics", "Edit"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBreadcrumbs("en", tt.items...)
			last := bc[len(bc)-1]
			if last.URL != "" {
				t.Fatalf("last item should have empty URL (active), got %q", last.URL)
			}
		})
	}
}

func Test_BreadcrumbJSONLD_WithBreadcrumbs(t *testing.T) {
	d := DefaultData{
		Language: "en",
		Breadcrumbs: []BreadcrumbItem{
			{Label: "Home", URL: "/"},
			{Label: "Schematics", URL: "/schematics"},
			{Label: "My Build"},
		},
	}
	result := string(d.BreadcrumbJSONLD())
	if result == "" {
		t.Fatal("expected non-empty JSON-LD")
	}
	if !strings.Contains(result, `"@type":"BreadcrumbList"`) {
		t.Fatal("missing BreadcrumbList type")
	}
	if !strings.Contains(result, `"@context":"https://schema.org"`) {
		t.Fatal("missing schema.org context")
	}
	if !strings.Contains(result, `"name":"Home"`) {
		t.Fatal("missing Home item")
	}
	if !strings.Contains(result, `"name":"Schematics"`) {
		t.Fatal("missing Schematics item")
	}
	if !strings.Contains(result, `"name":"My Build"`) {
		t.Fatal("missing My Build item")
	}
	if !strings.Contains(result, `"position":1`) {
		t.Fatal("missing position 1")
	}
	if !strings.Contains(result, `"position":3`) {
		t.Fatal("missing position 3")
	}
	if !strings.Contains(result, `<script type="application/ld+json">`) {
		t.Fatal("missing script tag")
	}
	// Last item should not have "item" since URL is empty
	if !strings.Contains(result, `https://createmod.com/schematics`) {
		t.Fatal("expected full URL for Schematics item")
	}
}

func Test_BreadcrumbJSONLD_Empty(t *testing.T) {
	d := DefaultData{}
	result := string(d.BreadcrumbJSONLD())
	if result != "" {
		t.Fatalf("expected empty string for no breadcrumbs, got %q", result)
	}
}

func Test_BreadcrumbJSONLD_LanguagePrefix(t *testing.T) {
	d := DefaultData{
		Language: "de",
		Breadcrumbs: []BreadcrumbItem{
			{Label: "Startseite", URL: "/"},
			{Label: "Schematics", URL: "/schematics"},
			{Label: "Mein Bau"},
		},
	}
	result := string(d.BreadcrumbJSONLD())
	if !strings.Contains(result, `https://createmod.com/de`) {
		t.Fatal("expected German language prefix in URL")
	}
	if !strings.Contains(result, `https://createmod.com/de/schematics`) {
		t.Fatal("expected /de/schematics in URL")
	}
}
