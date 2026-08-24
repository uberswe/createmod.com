package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

// ChecklistRecheckArgs re-evaluates a schematic's open moderation checklist
// after the author edits it (e.g. improved the description). If the relevant
// automated check now passes, the item is resolved; when no open items remain a
// published_limited/changes_requested schematic auto-promotes to fully
// published — reindexed and the author emailed — with no moderator step. (#1646)
type ChecklistRecheckArgs struct {
	SchematicID string `json:"schematic_id"`
}

func (ChecklistRecheckArgs) Kind() string { return "moderation_checklist_recheck" }

// ChecklistRecheckWorker runs the recheck + auto-promote flow.
type ChecklistRecheckWorker struct {
	river.WorkerDefaults[ChecklistRecheckArgs]
	deps Deps
}

func (w *ChecklistRecheckWorker) Work(ctx context.Context, job *river.Job[ChecklistRecheckArgs]) error {
	id := job.Args.SchematicID
	if w.deps.Store == nil {
		return nil
	}
	schem, err := w.deps.Store.Schematics.GetByID(ctx, id)
	if err != nil || schem == nil {
		slog.Warn("checklist recheck: schematic not found", "schematic_id", id, "error", err)
		return nil
	}
	// Only limited/changes-requested schematics have something to promote.
	if schem.ModerationState != store.ModerationPublishedLimited &&
		schem.ModerationState != store.ModerationChangesRequested {
		return nil
	}

	// Re-run the description quality check; resolve the auto description item(s)
	// when it passes. (Moderator-authored items are resolved from the review UI,
	// not here.)
	if w.deps.Moderation != nil {
		qual, qErr := w.deps.Moderation.CheckSchematicQuality(schem.Title, schem.Description)
		if qErr != nil {
			slog.Warn("checklist recheck: quality check unavailable", "schematic_id", id, "error", qErr)
		} else if qual.Approved && w.deps.Store.ModerationChecklist != nil {
			if n, rErr := w.deps.Store.ModerationChecklist.ResolveOpenByKind(ctx, id, store.ChecklistKindDescription); rErr != nil {
				slog.Error("checklist recheck: resolve failed", "schematic_id", id, "error", rErr)
			} else if n > 0 {
				slog.Info("checklist recheck: resolved description items", "schematic_id", id, "count", n)
			}
		}
	}

	promoted, err := pages.PromoteIfChecklistResolved(ctx, w.deps.Store, id)
	if err != nil {
		slog.Error("checklist recheck: promote failed", "schematic_id", id, "error", err)
		return err
	}
	if !promoted {
		return nil
	}

	slog.Info("checklist recheck: schematic auto-promoted to published", "schematic_id", id)

	// Side effects of reaching full visibility: index it and tell the author.
	if w.deps.Cache != nil {
		pages.RefreshIndexCache(w.deps.Cache, w.deps.Store, []int{7})
	}
	if w.deps.MeiliClient != nil {
		upsertSchematicToMeili(ctx, w.deps, id)
	}
	pages.SendSchematicLiveEmail(ctx, w.deps.Mail, w.deps.Store, id)
	return nil
}
