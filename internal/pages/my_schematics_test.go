package pages

import (
	"testing"

	"createmod/internal/models"
	"createmod/internal/store"
)

func TestBuildMySchematicCard(t *testing.T) {
	cases := []struct {
		name        string
		schem       *store.Schematic
		badge       string
		level       string
		needsAction bool
		hasLink     bool
	}{
		{"published", &store.Schematic{Name: "a", ModerationState: store.ModerationPublished}, "Published", "green", false, false},
		{"published+held", &store.Schematic{Name: "a", ModerationState: store.ModerationPublished, Gallery: []string{"x", "y"}, HeldImages: []string{"y"}}, "Image in review", "blue", false, true},
		{"limited", &store.Schematic{Name: "a", ModerationState: store.ModerationPublishedLimited}, "Link only", "yellow", true, true},
		{"changes_requested", &store.Schematic{Name: "a", ModerationState: store.ModerationChangesRequested}, "Changes requested", "orange", true, true},
		{"rejected_fixable", &store.Schematic{Name: "a", ModerationState: store.ModerationRejectedFixable}, "Rejected", "orange", true, true},
		{"rejected_final", &store.Schematic{Name: "a", ModerationState: store.ModerationRejectedFinal}, "Removed", "red", false, true},
		{"auto_review", &store.Schematic{Name: "a", ModerationState: store.ModerationAutoReview}, "Under review", "blue", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			card := buildMySchematicCard(c.schem, models.Schematic{})
			if card.Badge != c.badge || card.BadgeLevel != c.level {
				t.Errorf("badge = %q/%q, want %q/%q", card.Badge, card.BadgeLevel, c.badge, c.level)
			}
			if card.NeedsAction != c.needsAction {
				t.Errorf("NeedsAction = %v, want %v", card.NeedsAction, c.needsAction)
			}
			if (card.StatusLabel != "" && card.StatusLink != "") != c.hasLink {
				t.Errorf("hasLink = %v (label=%q link=%q), want %v", card.StatusLabel != "" && card.StatusLink != "", card.StatusLabel, card.StatusLink, c.hasLink)
			}
		})
	}
}
