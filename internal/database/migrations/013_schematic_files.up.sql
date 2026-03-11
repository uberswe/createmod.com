CREATE TABLE IF NOT EXISTS schematic_files (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    filename TEXT NOT NULL DEFAULT '',
    original_name TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    created TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_schematic_files_schematic_id ON schematic_files(schematic_id);
