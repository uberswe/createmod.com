CREATE TABLE comment_translations (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    comment_id  TEXT NOT NULL,
    language    TEXT NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_comment_translations_comment_lang ON comment_translations (comment_id, language);
CREATE INDEX idx_comment_translations_comment ON comment_translations (comment_id);
CREATE INDEX idx_comment_translations_language ON comment_translations (language);
