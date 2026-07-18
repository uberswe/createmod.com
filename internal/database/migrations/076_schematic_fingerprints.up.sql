CREATE TABLE IF NOT EXISTS schematic_fingerprints (
    schematic_id TEXT PRIMARY KEY,
    fp JSONB NOT NULL DEFAULT '{}',
    version INT NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_schematic_fingerprints_version ON schematic_fingerprints (version);
