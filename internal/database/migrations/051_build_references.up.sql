CREATE TABLE IF NOT EXISTS schematic_references (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    source_type     TEXT NOT NULL DEFAULT '',
    title           TEXT NOT NULL DEFAULT '',
    thumbnail_url   TEXT NOT NULL DEFAULT '',
    author_name     TEXT NOT NULL DEFAULT '',
    last_fetched    TIMESTAMPTZ,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_schematic_references_pair ON schematic_references (schematic_id, url);
CREATE INDEX IF NOT EXISTS idx_schematic_references_schematic ON schematic_references (schematic_id);
