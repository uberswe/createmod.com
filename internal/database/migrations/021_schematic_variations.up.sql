CREATE TABLE IF NOT EXISTS schematic_variations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    replacements JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schematic_variations_schematic ON schematic_variations(schematic_id);
CREATE INDEX IF NOT EXISTS idx_schematic_variations_user ON schematic_variations(user_id);
