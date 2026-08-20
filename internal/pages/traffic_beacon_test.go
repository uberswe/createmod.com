package pages

import "testing"

func TestPageClass(t *testing.T) {
	cases := map[string]string{
		"/":                          "home",
		"":                           "home",
		"/schematics":                "browse",
		"/schematics/copper-farm":    "schematic",
		"/es/schematics/copper-farm": "schematic", // lang prefix stripped
		"/pt-BR/schematics":          "browse",
		"/search":                    "browse",
		"/tags/redstone":             "browse",
		"/generators/hull":           "generator",
		"/tools/convert":             "tools",
		"/mods/aeronautics":          "mods",
		"/guides/getting-started":    "guides",
		"/author/someone":            "user",
		"/settings/api-keys":         "user",
		"/fr":                        "home", // lang-only path
		"/random-thing":              "other",
	}
	for path, want := range cases {
		if got := PageClass(path); got != want {
			t.Errorf("PageClass(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestSanitizeResolution(t *testing.T) {
	valid := []string{"1920x1080", "1366x1366", "1x1", "99999x99999"}
	for _, v := range valid {
		if sanitizeResolution(v) != v {
			t.Errorf("sanitizeResolution(%q) should keep it", v)
		}
	}
	invalid := []string{"", "1920", "1920X1080", "abcxdef", "1920x1080; DROP", "1920x", "x1080", "123456x1"}
	for _, v := range invalid {
		if got := sanitizeResolution(v); got != "" {
			t.Errorf("sanitizeResolution(%q) = %q, want empty", v, got)
		}
	}
}
