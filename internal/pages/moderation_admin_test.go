package pages

import (
	"context"
	"testing"

	"createmod/internal/store"
)

func TestApplyModeratorDecision(t *testing.T) {
	ctx := context.Background()

	newStore := func(state string) (*store.Store, *memChecklist, *memSchematics) {
		cl := newMemChecklist()
		sch := &memSchematics{row: &store.Schematic{ID: "s1", Name: "x", ModerationState: state}}
		// ModerationLog + ModerationChats left nil: the engine guards on nil.
		return &store.Store{Schematics: sch, ModerationChecklist: cl}, cl, sch
	}

	t.Run("approve_full publishes and clears checklist", func(t *testing.T) {
		appStore, cl, sch := newStore(store.ModerationFlagged)
		// pre-existing open item that must be resolved on approval
		_ = cl.Create(ctx, &store.ModerationChecklistItem{SchematicID: "s1", Kind: store.ChecklistKindDescription, Status: store.ChecklistStatusOpen})
		ns, os, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", DecisionApproveFull, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if ns != store.ModerationPublished || os != store.ModerationFlagged {
			t.Fatalf("states = %q/%q", ns, os)
		}
		if sch.row.ModerationState != store.ModerationPublished {
			t.Errorf("row state = %q", sch.row.ModerationState)
		}
		if n, _ := cl.CountOpenBySchematic(ctx, "s1"); n != 0 {
			t.Errorf("open items = %d, want 0", n)
		}
	})

	t.Run("publish_notes creates a moderator item per kind", func(t *testing.T) {
		appStore, cl, sch := newStore(store.ModerationFlagged)
		ns, _, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", DecisionPublishNotes, "fix these", []string{"description", "title"})
		if err != nil {
			t.Fatal(err)
		}
		if ns != store.ModerationPublishedLimited || sch.row.ModerationState != store.ModerationPublishedLimited {
			t.Fatalf("state = %q", sch.row.ModerationState)
		}
		open, _ := cl.ListOpenBySchematic(ctx, "s1")
		if len(open) != 2 {
			t.Fatalf("open items = %d, want 2", len(open))
		}
		for _, it := range open {
			if it.Source != store.ChecklistSourceModerator || it.Note != "fix these" {
				t.Errorf("item = %+v", it)
			}
		}
	})

	t.Run("publish_notes with no kinds falls back to one item", func(t *testing.T) {
		appStore, cl, _ := newStore(store.ModerationFlagged)
		if _, _, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", DecisionPublishNotes, "please improve", nil); err != nil {
			t.Fatal(err)
		}
		if n, _ := cl.CountOpenBySchematic(ctx, "s1"); n != 1 {
			t.Errorf("open items = %d, want 1", n)
		}
	})

	t.Run("request_changes -> changes_requested", func(t *testing.T) {
		appStore, _, sch := newStore(store.ModerationFlagged)
		if _, _, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", DecisionRequestChanges, "n", []string{"images"}); err != nil {
			t.Fatal(err)
		}
		if sch.row.ModerationState != store.ModerationChangesRequested {
			t.Errorf("state = %q", sch.row.ModerationState)
		}
	})

	t.Run("reject_fixable and reject_final", func(t *testing.T) {
		appStore, _, sch := newStore(store.ModerationFlagged)
		if _, _, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", DecisionRejectFixable, "fixable", []string{"description"}); err != nil {
			t.Fatal(err)
		}
		if sch.row.ModerationState != store.ModerationRejectedFixable {
			t.Errorf("state = %q, want rejected_fixable", sch.row.ModerationState)
		}
		appStore2, _, sch2 := newStore(store.ModerationFlagged)
		if _, _, err := applyModeratorDecision(ctx, appStore2, "s1", "mod1", DecisionRejectFinal, "severe", nil); err != nil {
			t.Fatal(err)
		}
		if sch2.row.ModerationState != store.ModerationRejectedFinal {
			t.Errorf("state = %q, want rejected_final", sch2.row.ModerationState)
		}
	})

	t.Run("unknown action errors", func(t *testing.T) {
		appStore, _, _ := newStore(store.ModerationFlagged)
		if _, _, err := applyModeratorDecision(ctx, appStore, "s1", "mod1", "bogus", "", nil); err == nil {
			t.Error("want error for unknown action")
		}
	})
}
