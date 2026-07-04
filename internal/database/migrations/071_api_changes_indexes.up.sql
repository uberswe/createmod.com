-- Indexes for GET /api/schematics/changes (store.ChangesSince).
--
-- That query filtered `schematic_versions.created > $1` (no index existed, so a
-- sequential scan of the ever-growing version history on every call) and
-- `schematics.deleted > $1` (the existing idx_schematics_deleted is partial,
-- WHERE deleted IS NULL, so it cannot serve a `deleted IS NOT NULL` range).
-- Both are now index-backed. The endpoint is public and polled by external
-- caches, so keeping it cheap matters.
CREATE INDEX IF NOT EXISTS idx_schematic_versions_created
    ON schematic_versions (created);

CREATE INDEX IF NOT EXISTS idx_schematics_deleted_notnull
    ON schematics (deleted) WHERE deleted IS NOT NULL;
