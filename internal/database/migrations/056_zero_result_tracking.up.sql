CREATE TABLE IF NOT EXISTS zero_result_suggestions (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    query      TEXT NOT NULL UNIQUE,
    suggestion TEXT NOT NULL DEFAULT '',
    auto       BOOLEAN NOT NULL DEFAULT true,
    created    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
