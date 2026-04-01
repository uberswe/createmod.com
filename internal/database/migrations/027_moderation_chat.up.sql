CREATE TABLE moderation_threads (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    content_type TEXT NOT NULL DEFAULT 'schematic',
    content_id   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'open',
    created      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(content_type, content_id)
);

CREATE TABLE moderation_messages (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    thread_id    TEXT NOT NULL REFERENCES moderation_threads(id) ON DELETE CASCADE,
    author_id    TEXT NOT NULL REFERENCES users(id),
    is_moderator BOOLEAN NOT NULL DEFAULT FALSE,
    body         TEXT NOT NULL CHECK (char_length(body) <= 2000),
    created      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mod_messages_thread ON moderation_messages(thread_id, created);
