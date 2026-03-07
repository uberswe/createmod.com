package pages

import "testing"

func Test_safeRedirectPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		want     string
	}{
		{"empty returns fallback", "", "/", "/"},
		{"simple relative path", "/schematics", "/", "/schematics"},
		{"root path", "/", "/", "/"},
		{"path with query", "/search?q=test", "/", "/search?q=test"},
		{"protocol relative URL blocked", "//evil.com/steal", "/", "/"},
		{"absolute URL blocked", "https://evil.com", "/", "/"},
		{"no leading slash blocked", "evil.com/steal", "/", "/"},
		{"javascript scheme blocked", "javascript:alert(1)", "/", "/"},
		{"data scheme blocked", "data:text/html,<h1>hi</h1>", "/", "/"},
		{"deep relative path allowed", "/admin/schematics/123", "/", "/admin/schematics/123"},
		{"path with fragment allowed", "/schematics#comments", "/", "/schematics#comments"},
		{"custom fallback used", "", "/login", "/login"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeRedirectPath(tt.input, tt.fallback)
			if got != tt.want {
				t.Errorf("safeRedirectPath(%q, %q) = %q, want %q", tt.input, tt.fallback, got, tt.want)
			}
		})
	}
}

func Test_sanitizeContentDispositionFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal filename", "schematic.nbt", "schematic.nbt"},
		{"empty returns default", "", "download"},
		{"strips quotes", "file\".nbt", "file.nbt"},
		{"strips backslash", "file\\.nbt", "file.nbt"},
		{"strips newline", "file\r\n.nbt", "file.nbt"},
		{"strips null byte", "file\x00.nbt", "file.nbt"},
		{"preserves spaces", "my schematic.nbt", "my schematic.nbt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeContentDispositionFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeContentDispositionFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
