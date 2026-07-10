CREATE TABLE IF NOT EXISTS schematic_safety (
    schematic_id TEXT PRIMARY KEY,
    checksum TEXT NOT NULL DEFAULT '',
    file_safe BOOLEAN NOT NULL DEFAULT false,
    manifest JSONB NOT NULL DEFAULT '{}',
    pipeline_version INT NOT NULL DEFAULT 0,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_schematic_safety_pipeline ON schematic_safety (pipeline_version);
