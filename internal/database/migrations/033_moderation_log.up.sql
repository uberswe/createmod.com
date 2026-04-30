CREATE TABLE IF NOT EXISTS moderation_log (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    actor_id     TEXT NOT NULL DEFAULT '',
    actor_type   TEXT NOT NULL DEFAULT 'system',
    action       TEXT NOT NULL,
    old_state    TEXT NOT NULL DEFAULT '',
    new_state    TEXT NOT NULL DEFAULT '',
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_log_schematic ON moderation_log (schematic_id, created_at DESC);
