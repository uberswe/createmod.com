// Package textutil holds small, dependency-free helpers for handling user text
// safely — chiefly length caps that never split a multi-byte UTF-8 character.
package textutil

// TruncateRunes caps s to at most n runes without splitting a multi-byte UTF-8
// character. Slicing a string at a byte offset (s[:n]) can cut a rune in half
// and produce invalid UTF-8, which Postgres rejects on write (SQLSTATE 22021,
// "invalid byte sequence for encoding UTF8"). Every length cap on stored user
// text must go through this — non-ASCII text (e.g. Cyrillic) hits it constantly.
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s // fewer bytes than n ⇒ fewer runes than n
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// TruncateRunesEllipsis caps s to n runes (rune-safe) and appends ellipsis only
// when it actually truncated. The ellipsis is caller-supplied so display code
// can keep a single "…" while stored fields use "...".
func TruncateRunesEllipsis(s string, n int, ellipsis string) string {
	t := TruncateRunes(s, n)
	if len(t) < len(s) {
		return t + ellipsis
	}
	return t
}
