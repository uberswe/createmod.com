package pages

import (
	"context"
	"log/slog"

	"createmod/internal/store"
)

// CreateAutoChecklistItem adds an automated checklist item of the given kind if
// the schematic has no open item of that kind already (so re-running a check
// doesn't pile up duplicates). Returns true if a new item was created. (#1646)
func CreateAutoChecklistItem(ctx context.Context, appStore *store.Store, schematicID, kind, note string) bool {
	if appStore == nil || appStore.ModerationChecklist == nil {
		return false
	}
	open, err := appStore.ModerationChecklist.ListOpenBySchematic(ctx, schematicID)
	if err != nil {
		slog.Error("moderation checklist: list open failed", "schematic_id", schematicID, "error", err)
		return false
	}
	for _, it := range open {
		if it.Kind == kind && it.Source == store.ChecklistSourceAuto {
			return false // already tracked
		}
	}
	item := &store.ModerationChecklistItem{
		SchematicID: schematicID,
		Kind:        kind,
		Source:      store.ChecklistSourceAuto,
		Note:        note,
	}
	if err := appStore.ModerationChecklist.Create(ctx, item); err != nil {
		slog.Error("moderation checklist: create failed", "schematic_id", schematicID, "kind", kind, "error", err)
		return false
	}
	return true
}

// EnterPublishedLimited moves a schematic into published_limited (reachable by
// link, kept out of Latest/search) and records an auto description checklist
// item. Used when the text-quality check fails but nothing violates policy —
// the schematic still publishes, just limited. No-op if already non-viewable or
// already limited/changes_requested. Returns true if it transitioned. (#1646)
func EnterPublishedLimited(ctx context.Context, appStore *store.Store, schem *store.Schematic, note string) bool {
	if appStore == nil || schem == nil {
		return false
	}
	switch schem.ModerationState {
	case store.ModerationPublishedLimited, store.ModerationChangesRequested:
		// Already limited; just make sure the checklist item exists.
		CreateAutoChecklistItem(ctx, appStore, schem.ID, store.ChecklistKindDescription, note)
		return false
	}
	old := schem.ModerationState
	schem.ModerationState = store.ModerationPublishedLimited
	schem.ModerationReason = note
	if err := appStore.Schematics.Update(ctx, schem); err != nil {
		slog.Error("moderation: failed to enter published_limited", "schematic_id", schem.ID, "error", err)
		return false
	}
	// This is the automated quality outcome, so mark it as a system review: it
	// drives the "automated review" email note and the request-human-review CTA.
	_ = appStore.Schematics.SetModerationReviewedBy(ctx, schem.ID, store.ReviewedBySystem)
	CreateAutoChecklistItem(ctx, appStore, schem.ID, store.ChecklistKindDescription, note)
	logModerationState(ctx, appStore, schem.ID, old, schem.ModerationState, "quality check failed, published with limits: "+note)
	return true
}

// PromoteIfChecklistResolved promotes a schematic to fully published when it has
// no open checklist items left and is currently in a promotable state
// (published_limited or changes_requested). Pure state/DB work — callers handle
// the side effects (search reindex, "now live" email). Returns true if it
// promoted. (#1646)
func PromoteIfChecklistResolved(ctx context.Context, appStore *store.Store, schematicID string) (bool, error) {
	if appStore == nil || appStore.ModerationChecklist == nil {
		return false, nil
	}
	schem, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || schem == nil {
		return false, err
	}
	if schem.ModerationState != store.ModerationPublishedLimited &&
		schem.ModerationState != store.ModerationChangesRequested {
		return false, nil
	}
	openCount, err := appStore.ModerationChecklist.CountOpenBySchematic(ctx, schematicID)
	if err != nil {
		return false, err
	}
	if openCount > 0 {
		return false, nil
	}
	old := schem.ModerationState
	schem.ModerationState = store.ModerationPublished
	schem.ModerationReason = ""
	if err := appStore.Schematics.Update(ctx, schem); err != nil {
		return false, err
	}
	logModerationState(ctx, appStore, schematicID, old, schem.ModerationState, "auto-promoted: all checklist items resolved")
	return true, nil
}

// logModerationState writes a system state-change entry, ignoring errors (audit
// log is best-effort).
func logModerationState(ctx context.Context, appStore *store.Store, schematicID, oldState, newState, reason string) {
	if appStore == nil || appStore.ModerationLog == nil {
		return
	}
	_ = appStore.ModerationLog.Create(ctx, &store.ModerationLogEntry{
		SchematicID: schematicID,
		ActorType:   "system",
		Action:      "state_change",
		OldState:    oldState,
		NewState:    newState,
		Reason:      reason,
	})
}
