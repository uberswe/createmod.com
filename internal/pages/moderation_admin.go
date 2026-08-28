package pages

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"createmod/internal/store"
)

// Moderator decision actions. (#1646)
const (
	DecisionApproveFull    = "approve_full"
	DecisionPublishNotes   = "publish_notes"
	DecisionRequestChanges = "request_changes"
	DecisionRejectFixable  = "reject_fixable"
	DecisionRejectFinal    = "reject_final"
)

var validDecisionKinds = map[string]bool{
	store.ChecklistKindDescription: true,
	store.ChecklistKindTitle:       true,
	store.ChecklistKindImages:      true,
	store.ChecklistKindTags:        true,
	store.ChecklistKindCategory:    true,
}

// applyModeratorDecision transitions a schematic per a moderator's decision and
// records the checklist items, moderation-log entry and chat note that go with
// it. Returns the new and old states. Search reindex + author email are handled
// by the caller (they need job/mailer deps). (#1646)
func applyModeratorDecision(ctx context.Context, appStore *store.Store, schematicID, moderatorID, action, note string, kinds []string) (newState, oldState string, err error) {
	schem, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || schem == nil {
		return "", "", fmt.Errorf("schematic not found")
	}
	oldState = schem.ModerationState

	switch action {
	case DecisionApproveFull:
		schem.ModerationState = store.ModerationPublished
		schem.ModerationReason = ""
		// Surface at the top of Latest on approval, so time spent waiting in the
		// moderation queue doesn't push the schematic down the feed. (#1646)
		now := time.Now()
		schem.CreatedOverride = &now
		resolveAllOpenChecklist(ctx, appStore, schematicID)
	case DecisionPublishNotes:
		schem.ModerationState = store.ModerationPublishedLimited
		schem.ModerationReason = note
		createModeratorChecklist(ctx, appStore, schematicID, kinds, note)
	case DecisionRequestChanges:
		schem.ModerationState = store.ModerationChangesRequested
		schem.ModerationReason = note
		createModeratorChecklist(ctx, appStore, schematicID, kinds, note)
	case DecisionRejectFixable:
		schem.ModerationState = store.ModerationRejectedFixable
		schem.ModerationReason = note
		createModeratorChecklist(ctx, appStore, schematicID, kinds, note)
	case DecisionRejectFinal:
		schem.ModerationState = store.ModerationRejectedFinal
		schem.ModerationReason = note
	default:
		return "", "", fmt.Errorf("unknown decision %q", action)
	}

	if err := appStore.Schematics.Update(ctx, schem); err != nil {
		return "", "", err
	}
	// A moderator decided, so this is now a human review and any pending author
	// request for human review is satisfied. A later fix by the author must
	// therefore come back to a human, never auto-promote. (#1646)
	_ = appStore.Schematics.SetModerationReviewedBy(ctx, schematicID, store.ReviewedByHuman)
	_ = appStore.Schematics.SetHumanReviewRequested(ctx, schematicID, false)
	logModeratorDecision(ctx, appStore, schematicID, moderatorID, oldState, schem.ModerationState, action, note)
	if strings.TrimSpace(note) != "" {
		postModeratorChat(ctx, appStore, schematicID, moderatorID, note)
	}
	return schem.ModerationState, oldState, nil
}

// createModeratorChecklist adds one moderator checklist item per selected kind
// (falling back to a single description item if none were chosen), so the author
// sees exactly what to fix.
func createModeratorChecklist(ctx context.Context, appStore *store.Store, schematicID string, kinds []string, note string) {
	if appStore.ModerationChecklist == nil {
		return
	}
	added := 0
	for _, k := range kinds {
		if !validDecisionKinds[k] {
			continue
		}
		item := &store.ModerationChecklistItem{
			SchematicID: schematicID, Kind: k, Source: store.ChecklistSourceModerator, Note: note,
		}
		if err := appStore.ModerationChecklist.Create(ctx, item); err != nil {
			slog.Error("moderator decision: create checklist item failed", "schematic_id", schematicID, "kind", k, "error", err)
			continue
		}
		added++
	}
	if added == 0 {
		item := &store.ModerationChecklistItem{
			SchematicID: schematicID, Kind: store.ChecklistKindDescription, Source: store.ChecklistSourceModerator, Note: note,
		}
		_ = appStore.ModerationChecklist.Create(ctx, item)
	}
}

func resolveAllOpenChecklist(ctx context.Context, appStore *store.Store, schematicID string) {
	if appStore.ModerationChecklist == nil {
		return
	}
	open, err := appStore.ModerationChecklist.ListOpenBySchematic(ctx, schematicID)
	if err != nil {
		return
	}
	for _, it := range open {
		_ = appStore.ModerationChecklist.Resolve(ctx, it.ID)
	}
}

func logModeratorDecision(ctx context.Context, appStore *store.Store, schematicID, moderatorID, oldState, newState, action, note string) {
	if appStore.ModerationLog == nil {
		return
	}
	reason := "moderator: " + action
	if note != "" {
		reason += " — " + note
	}
	_ = appStore.ModerationLog.Create(ctx, &store.ModerationLogEntry{
		SchematicID: schematicID,
		ActorID:     moderatorID,
		ActorType:   "admin",
		Action:      "decision",
		OldState:    oldState,
		NewState:    newState,
		Reason:      reason,
	})
}

// postModeratorChat posts the moderator's note into the schematic's moderation
// thread (creating it if needed) so the author sees it in the same place.
func postModeratorChat(ctx context.Context, appStore *store.Store, schematicID, moderatorID, note string) {
	if appStore.ModerationChats == nil {
		return
	}
	thread, err := appStore.ModerationChats.GetThreadByContent(ctx, "schematic", schematicID)
	if err != nil || thread == nil {
		thread, err = appStore.ModerationChats.CreateThread(ctx, "schematic", schematicID)
		if err != nil || thread == nil {
			slog.Error("moderator decision: create chat thread failed", "schematic_id", schematicID, "error", err)
			return
		}
	}
	if _, err := appStore.ModerationChats.CreateMessage(ctx, thread.ID, moderatorID, true, note); err != nil {
		slog.Error("moderator decision: post chat message failed", "schematic_id", schematicID, "error", err)
	}
}
