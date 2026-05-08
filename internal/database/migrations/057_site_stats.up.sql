CREATE TABLE search_term_moderation (
    query       TEXT PRIMARY KEY,
    is_clean    BOOLEAN NOT NULL DEFAULT true,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_term_moderation_checked ON search_term_moderation (checked_at);
