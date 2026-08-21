package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateRunesValidUTF8 guards the schematic-save bug: byte-slicing a
// UTF-8 string (s[:n]) can cut a multi-byte rune in half, producing invalid
// UTF-8 that Postgres rejects with SQLSTATE 22021. Every result must stay valid.
func TestTruncateRunesValidUTF8(t *testing.T) {
	cyrillic := strings.Repeat("я", 200)    // 2 bytes/rune → 400 bytes, byte 180 is mid-rune
	emdash := strings.Repeat("—", 100)      // 3 bytes/rune → boundary lands mid-rune
	mixed := strings.Repeat("aя—b", 100)    // mixed widths
	ellipsisChar := strings.Repeat("…", 90) // 3 bytes/rune

	for _, in := range []string{cyrillic, emdash, mixed, ellipsisChar} {
		for _, n := range []int{1, 179, 180, 200, 181, 256} {
			got := TruncateRunes(in, n)
			if !utf8.ValidString(got) {
				t.Errorf("TruncateRunes(%d runes, n=%d) produced invalid UTF-8", utf8.RuneCountInString(in), n)
			}
			if utf8.RuneCountInString(got) > n {
				t.Errorf("TruncateRunes n=%d returned %d runes, want <= n", n, utf8.RuneCountInString(got))
			}
			if e := TruncateRunesEllipsis(in, n, "…"); !utf8.ValidString(e) {
				t.Errorf("TruncateRunesEllipsis(n=%d) produced invalid UTF-8", n)
			}
		}
	}
}

func TestTruncateRunesBehaviour(t *testing.T) {
	if got := TruncateRunes("hello", 10); got != "hello" {
		t.Errorf("short ASCII: got %q", got)
	}
	if got := TruncateRunes("абвгд", 3); got != "абв" {
		t.Errorf("cyrillic cap: got %q (%d runes)", got, utf8.RuneCountInString(got))
	}
	if got := TruncateRunes("hi", 0); got != "" {
		t.Errorf("n=0: got %q", got)
	}
	// Ellipsis only when actually truncated, and the caller's ellipsis is used.
	if got := TruncateRunesEllipsis("hi", 5, "..."); got != "hi" {
		t.Errorf("no-trunc ellipsis: got %q", got)
	}
	if got := TruncateRunesEllipsis("абвгд", 3, "..."); got != "абв..." {
		t.Errorf("trunc ellipsis: got %q", got)
	}
	if got := TruncateRunesEllipsis("абвгд", 3, "…"); got != "абв…" {
		t.Errorf("trunc custom ellipsis: got %q", got)
	}
}
