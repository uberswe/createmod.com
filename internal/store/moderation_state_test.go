package store

import "testing"

// TestVisibilityGates locks in the two-gate model from the moderation overhaul
// (#1646): IsPublicState = listed/indexed; IsViewableState = reachable by direct
// link. published_limited must be viewable but NOT listed; owner-only states
// (changes_requested, rejected_*) must be neither.
func TestVisibilityGates(t *testing.T) {
	cases := []struct {
		state    string
		listed   bool // IsPublicState
		viewable bool // IsViewableState
	}{
		{ModerationPublished, true, true},
		{ModerationApproved, true, true},
		{ModerationPublishedLimited, false, true}, // link works, unlisted
		{ModerationChangesRequested, false, false},
		{ModerationRejectedFixable, false, false},
		{ModerationRejectedFinal, false, false},
		{ModerationAutoReview, false, false},
		{ModerationFlagged, false, false},
		{ModerationRejected, false, false},
		{ModerationDeleted, false, false},
	}
	for _, c := range cases {
		if got := IsPublicState(c.state); got != c.listed {
			t.Errorf("IsPublicState(%q) = %v, want %v", c.state, got, c.listed)
		}
		if got := IsViewableState(c.state); got != c.viewable {
			t.Errorf("IsViewableState(%q) = %v, want %v", c.state, got, c.viewable)
		}
		// A listed schematic must always be viewable.
		if c.listed && !c.viewable {
			t.Errorf("%q listed but not viewable — inconsistent test case", c.state)
		}
	}
}
