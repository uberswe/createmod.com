package translation

import "testing"

// TestTranslateForUser_Passthrough verifies the safe fallbacks that must never
// call OpenAI: English, empty/whitespace input, and unsupported languages all
// return the original text unchanged. (These paths return before any API call,
// so no key/cache is needed.)
func TestTranslateForUser_Passthrough(t *testing.T) {
	s := New("", nil, nil)
	cases := []struct {
		name, text, lang, want string
	}{
		{"english is passthrough", "Fix your description", "en", "Fix your description"},
		{"empty stays empty", "", "es", ""},
		{"whitespace trims to empty", "   ", "es", ""},
		{"unsupported language passthrough", "Fix your description", "xx", "Fix your description"},
		{"empty target passthrough", "Fix your description", "", "Fix your description"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := s.TranslateForUser(nil, c.text, c.lang); got != c.want {
				t.Errorf("TranslateForUser(%q, %q) = %q, want %q", c.text, c.lang, got, c.want)
			}
		})
	}
}
