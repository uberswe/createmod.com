package pages

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"createmod/internal/store"
)

func openDescItem() []store.ModerationChecklistItem {
	return []store.ModerationChecklistItem{{
		Kind: store.ChecklistKindDescription, Source: store.ChecklistSourceAuto,
		Note: "Add more detail so people can find your build.", Status: store.ChecklistStatusOpen,
		CreatedAt: time.Unix(0, 0),
	}}
}

func TestComputePublishOutcome(t *testing.T) {
	base := func() *UploadPendingData {
		return &UploadPendingData{SchematicName: "steam-train", SchematicURL: "/schematics/steam-train"}
	}
	cases := []struct {
		name      string
		schem     *store.Schematic
		items     []store.ModerationChecklistItem
		outcome   string
		heroLevel string
		listState string // "Latest & search" rail row state
		linkState string
	}{
		{"nil-pending", nil, nil, "pending", "pending", "", ""},
		{"auto_review", &store.Schematic{ModerationState: store.ModerationAutoReview}, nil, "pending", "pending", "", ""},
		{"full", &store.Schematic{ModerationState: store.ModerationPublished}, nil, "full", "ok", "ok", "ok"},
		{"held", &store.Schematic{ModerationState: store.ModerationPublished, Gallery: []string{"a", "b"}, HeldImages: []string{"b"}}, nil, "held", "ok", "ok", "ok"},
		{"limited", &store.Schematic{ModerationState: store.ModerationPublishedLimited}, openDescItem(), "limited", "warn", "warn", "ok"},
		{"limited_held", &store.Schematic{ModerationState: store.ModerationPublishedLimited, Gallery: []string{"a", "b"}, HeldImages: []string{"b"}}, openDescItem(), "limited_held", "warn", "warn", "ok"},
		{"rejected_fixable", &store.Schematic{ModerationState: store.ModerationRejectedFixable, ModerationReason: "nope"}, nil, "rejected_fixable", "bad", "bad", "bad"},
		{"rejected_final", &store.Schematic{ModerationState: store.ModerationRejectedFinal, ModerationReason: "severe"}, nil, "rejected_final", "bad", "bad", "bad"},
		{"flagged", &store.Schematic{ModerationState: store.ModerationFlagged}, nil, "flagged", "pending", "bad", "bad"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := base()
			computePublishOutcome(d, c.schem, c.items)
			if d.Outcome != c.outcome {
				t.Errorf("Outcome = %q, want %q", d.Outcome, c.outcome)
			}
			if d.HeroLevel != c.heroLevel {
				t.Errorf("HeroLevel = %q, want %q", d.HeroLevel, c.heroLevel)
			}
			if d.HeroTitle == "" {
				t.Error("HeroTitle is empty")
			}
			if c.linkState != "" {
				var link, list string
				for _, r := range d.Appears {
					switch r.Name {
					case "Direct link":
						link = r.State
					case "Latest & search":
						list = r.State
					}
				}
				if link != c.linkState {
					t.Errorf("Direct link state = %q, want %q", link, c.linkState)
				}
				if list != c.listState {
					t.Errorf("Latest & search state = %q, want %q", list, c.listState)
				}
			}
			// Sanity: a limited outcome must carry an actionable description item.
			if c.outcome == "limited" {
				found := false
				for _, it := range d.OpenItems {
					if it.Kind == store.ChecklistKindDescription && !it.NoAction && it.CTALabel != "" {
						found = true
					}
				}
				if !found {
					t.Error("limited outcome missing actionable description item")
				}
			}
			// A held outcome must carry a no-action image note.
			if c.outcome == "held" || c.outcome == "limited_held" {
				found := false
				for _, it := range d.OpenItems {
					if it.NoAction {
						found = true
					}
				}
				if !found {
					t.Error("held outcome missing no-action image note")
				}
			}
		})
	}
}

// TestRenderPublishResultFixtures renders the fragment for the main outcomes to
// HTML files (when CM_RENDER_FIXTURES=1) so the result page can be eyeballed in
// a browser. Always parses the template, so a broken template fails the test.
func TestRenderPublishResultFixtures(t *testing.T) {
	tmpl, err := template.ParseFiles("../../template/upload_result.html")
	if err != nil {
		t.Fatalf("parse upload_result.html: %v", err)
	}
	out := os.Getenv("CM_FIXTURE_DIR")
	scenarios := map[string]*store.Schematic{
		"full":         {ModerationState: store.ModerationPublished},
		"held":         {ModerationState: store.ModerationPublished, Gallery: []string{"a", "b"}, HeldImages: []string{"b"}},
		"limited":      {ModerationState: store.ModerationPublishedLimited},
		"limited_held": {ModerationState: store.ModerationPublishedLimited, Gallery: []string{"a", "b"}, HeldImages: []string{"b"}},
		"rejected":     {ModerationState: store.ModerationRejectedFinal, ModerationReason: "This image is not a Minecraft build."},
		"pending":      {ModerationState: store.ModerationAutoReview},
	}
	for name, schem := range scenarios {
		d := &UploadPendingData{SchematicName: "steam-train", SchematicURL: "/schematics/steam-train", SchematicFullURL: "https://createmod.com/schematics/steam-train"}
		var items []store.ModerationChecklistItem
		if schem.ModerationState == store.ModerationPublishedLimited {
			items = openDescItem()
		}
		computePublishOutcome(d, schem, items)
		// Always execute the template so a field rename or bad pipeline fails the
		// test; write a fixture file too when CM_FIXTURE_DIR is set.
		w := io.Discard
		if out != "" {
			f, err := os.Create(filepath.Join(out, name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			w = f
		}
		if err := tmpl.Execute(w, d); err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}
	}
}
