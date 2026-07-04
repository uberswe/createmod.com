package pages

import (
	"testing"
	"time"
)

func Test_ChangesCursor_RoundTrip(t *testing.T) {
	at := time.Date(2026, 7, 4, 12, 30, 45, 123456789, time.UTC)
	// name deliberately contains the separator to prove it survives.
	token := encodeChangesCursor(at, "some|weird-name", "updated")

	gotAt, gotName, gotKind, ok := decodeChangesCursor(token)
	if !ok {
		t.Fatalf("decode failed for token %q", token)
	}
	if !gotAt.Equal(at) {
		t.Errorf("at: got %v want %v", gotAt, at)
	}
	if gotName != "some|weird-name" {
		t.Errorf("name: got %q want %q", gotName, "some|weird-name")
	}
	if gotKind != "updated" {
		t.Errorf("kind: got %q want %q", gotKind, "updated")
	}
}

func Test_ChangesCursor_BackwardsCompatRFC3339(t *testing.T) {
	// The original cursor format was a bare RFC3339Nano timestamp; it must still
	// be accepted (as a position at the start of that timestamp).
	raw := "2026-07-04T12:30:45.123456789Z"
	gotAt, gotName, gotKind, ok := decodeChangesCursor(raw)
	if !ok {
		t.Fatalf("decode failed for legacy cursor %q", raw)
	}
	want, _ := time.Parse(time.RFC3339Nano, raw)
	if !gotAt.Equal(want) {
		t.Errorf("at: got %v want %v", gotAt, want)
	}
	if gotName != "" || gotKind != "" {
		t.Errorf("legacy cursor should have empty name/kind, got %q/%q", gotName, gotKind)
	}
}

func Test_ChangesCursor_Invalid(t *testing.T) {
	if _, _, _, ok := decodeChangesCursor("!!!not-valid!!!"); ok {
		t.Errorf("expected decode to fail for garbage cursor")
	}
}
