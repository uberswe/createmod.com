package pages

import (
	"testing"

	"createmod/internal/store"
)

func TestComputeOwnerModeration(t *testing.T) {
	descItem := []store.ModerationChecklistItem{{Kind: store.ChecklistKindDescription, Source: store.ChecklistSourceAuto, Note: "add detail", Status: store.ChecklistStatusOpen}}

	t.Run("limited+held stacks two banners", func(t *testing.T) {
		s := &store.Schematic{Name: "x", ModerationState: store.ModerationPublishedLimited, Gallery: []string{"a", "b"}, HeldImages: []string{"b"}}
		om := computeOwnerModeration(s, descItem)
		if len(om.Banners) != 2 {
			t.Fatalf("banners = %d, want 2", len(om.Banners))
		}
		if om.Banners[0].Level != "warn" {
			t.Errorf("state banner level = %q, want warn", om.Banners[0].Level)
		}
		if om.Banners[1].Level != "info" {
			t.Errorf("held banner level = %q, want info", om.Banners[1].Level)
		}
		if !om.HasChecklist || len(om.Checklist) != 1 {
			t.Fatalf("checklist = %d, want 1", len(om.Checklist))
		}
		if om.Checklist[0].SourceLabel != "Auto-check" || om.Checklist[0].CTALabel != "Fix description" {
			t.Errorf("checklist row = %+v", om.Checklist[0])
		}
	})

	t.Run("clean published shows one green banner, no checklist", func(t *testing.T) {
		s := &store.Schematic{Name: "x", ModerationState: store.ModerationPublished}
		om := computeOwnerModeration(s, nil)
		if len(om.Banners) != 1 || om.Banners[0].Level != "ok" {
			t.Fatalf("banners = %+v, want one ok", om.Banners)
		}
		if !om.FullyPublished || om.HasChecklist {
			t.Errorf("FullyPublished=%v HasChecklist=%v", om.FullyPublished, om.HasChecklist)
		}
	})

	t.Run("final rejection sets appeal-only", func(t *testing.T) {
		s := &store.Schematic{Name: "x", ModerationState: store.ModerationRejectedFinal, ModerationReason: "nope"}
		om := computeOwnerModeration(s, nil)
		if len(om.Banners) != 1 || om.Banners[0].Level != "bad" {
			t.Fatalf("banners = %+v, want one bad", om.Banners)
		}
		if !om.IsFinalReject || !om.AppealOnly {
			t.Errorf("IsFinalReject=%v AppealOnly=%v", om.IsFinalReject, om.AppealOnly)
		}
	})

	t.Run("moderator source pill", func(t *testing.T) {
		s := &store.Schematic{Name: "x", ModerationState: store.ModerationChangesRequested}
		items := []store.ModerationChecklistItem{{Kind: store.ChecklistKindTitle, Source: store.ChecklistSourceModerator, Note: "fix title", Status: store.ChecklistStatusOpen}}
		om := computeOwnerModeration(s, items)
		if om.Checklist[0].SourceLevel != "moderator" || om.Checklist[0].SourceLabel != "Moderator" {
			t.Errorf("row = %+v, want moderator", om.Checklist[0])
		}
	})

	t.Run("published with only a removed image shows removal banner", func(t *testing.T) {
		s := &store.Schematic{Name: "x", ModerationState: store.ModerationPublished, RemovedImages: []string{"c"}}
		om := computeOwnerModeration(s, nil)
		if len(om.Banners) != 1 || om.Banners[0].Level != "bad" {
			t.Fatalf("banners = %+v, want one bad removal", om.Banners)
		}
	})
}
