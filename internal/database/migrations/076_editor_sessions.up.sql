CREATE TABLE IF NOT EXISTS editor_sessions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT,
    source_kind TEXT NOT NULL,
    source_ref TEXT NOT NULL DEFAULT '',
    ops JSONB NOT NULL DEFAULT '[]',
    cursor INT NOT NULL DEFAULT 0,
    created TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_editor_sessions_updated ON editor_sessions (updated);
