-- Moderation overhaul (#1646): per-image holds, a resubmit counter, and a
-- checklist of actionable items a user must resolve to unlock full visibility.
-- The new moderation_state values (published_limited, changes_requested,
-- rejected_fixable, rejected_final) reuse the existing moderation_state TEXT
-- column, so no enum change is needed here.

-- Per-image hold state. Held images are hidden from visitors and shown to the
-- owner as a placeholder; removed images were taken down by a moderator after
-- review. Both are filenames that also appear (or appeared) in gallery /
-- featured_image; holding an image does NOT change the schematic's state.
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS held_images TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS removed_images TEXT[] NOT NULL DEFAULT '{}';

-- Exactly one resubmission is allowed after a rejected_fixable outcome.
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS moderation_resubmit_count INTEGER NOT NULL DEFAULT 0;

-- Actionable checklist items. Auto items are created by failed quality checks;
-- moderator items via the review UI. When a schematic has no open items left it
-- auto-promotes (published_limited / changes_requested -> published).
CREATE TABLE IF NOT EXISTS moderation_checklist_items (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL,                -- description | title | images | tags | category
    source       TEXT NOT NULL DEFAULT 'auto', -- auto | moderator
    note         TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'open', -- open | resolved
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_moderation_checklist_schematic
    ON moderation_checklist_items (schematic_id, status);
