CREATE TABLE IF NOT EXISTS schematic_events (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    event_type      SMALLINT NOT NULL,
    event_value     INTEGER NOT NULL DEFAULT 1,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schematic_events_lookup ON schematic_events (schematic_id, event_type, created);
CREATE INDEX IF NOT EXISTS idx_schematic_events_created ON schematic_events (created);
