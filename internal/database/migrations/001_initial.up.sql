-- CreateMod.com PostgreSQL Schema
-- Migrated from PocketBase/SQLite
--
-- Design principles:
--   * Keep PocketBase's 15-char alphanumeric IDs as TEXT PRIMARY KEY
--     (avoids breaking file URLs and cross-references)
--   * Convert PocketBase relation fields (JSON arrays in SQLite)
--     to proper junction tables with foreign keys
--   * TIMESTAMPTZ for all timestamps
--   * Soft deletes via nullable deleted TIMESTAMPTZ
--   * JSONB for materials/mods arrays, TEXT[] for gallery

-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE users (
    id              TEXT PRIMARY KEY,  -- 15-char alphanumeric, matches PocketBase
    email           TEXT NOT NULL,
    username        TEXT NOT NULL,
    password_hash   TEXT NOT NULL DEFAULT '',  -- bcrypt hash
    old_password    TEXT NOT NULL DEFAULT '',  -- phpass hash from WordPress migration
    avatar          TEXT NOT NULL DEFAULT '',  -- Gravatar URL
    points          INTEGER NOT NULL DEFAULT 0,
    verified        BOOLEAN NOT NULL DEFAULT false,
    is_admin        BOOLEAN NOT NULL DEFAULT false,
    deleted         TIMESTAMPTZ,              -- soft delete
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_username_lower ON users (LOWER(username));

-- ============================================================
-- EXTERNAL AUTHS (OAuth providers: Discord, GitHub, etc.)
-- ============================================================
CREATE TABLE external_auths (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,    -- 'discord', 'github', etc.
    provider_id     TEXT NOT NULL,    -- ID from the OAuth provider
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_external_auths_provider_id ON external_auths (provider, provider_id);
CREATE INDEX idx_external_auths_user ON external_auths (user_id);

-- ============================================================
-- SESSIONS (replaces PocketBase JWT auth)
-- ============================================================
CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,         -- random token
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

-- ============================================================
-- USER META (generic key-value metadata per user)
-- ============================================================
CREATE TABLE user_meta (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    value           TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_user_meta_user_key ON user_meta (user_id, key);

-- ============================================================
-- SCHEMATIC CATEGORIES
-- ============================================================
CREATE TABLE schematic_categories (
    id              TEXT PRIMARY KEY,
    key             TEXT NOT NULL,
    name            TEXT NOT NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_schematic_categories_key ON schematic_categories (key);

-- ============================================================
-- SCHEMATIC TAGS
-- ============================================================
CREATE TABLE schematic_tags (
    id              TEXT PRIMARY KEY,
    key             TEXT NOT NULL,
    name            TEXT NOT NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_schematic_tags_key ON schematic_tags (key);

-- ============================================================
-- CREATEMOD VERSIONS
-- ============================================================
CREATE TABLE createmod_versions (
    id              TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- MINECRAFT VERSIONS
-- ============================================================
CREATE TABLE minecraft_versions (
    id              TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- SCHEMATICS (main content table)
-- ============================================================
CREATE TABLE schematics (
    id                  TEXT PRIMARY KEY,
    author_id           TEXT REFERENCES users(id),
    name                TEXT NOT NULL DEFAULT '',     -- URL slug
    title               TEXT NOT NULL DEFAULT '',
    description         TEXT NOT NULL DEFAULT '',
    excerpt             TEXT NOT NULL DEFAULT '',
    content             TEXT NOT NULL DEFAULT '',     -- HTML content
    postdate            TIMESTAMPTZ,
    modified            TIMESTAMPTZ,
    detected_language   TEXT NOT NULL DEFAULT '',
    featured_image      TEXT NOT NULL DEFAULT '',     -- filename in S3
    gallery             TEXT[] NOT NULL DEFAULT '{}', -- array of filenames
    schematic_file      TEXT NOT NULL DEFAULT '',     -- NBT filename in S3
    video               TEXT NOT NULL DEFAULT '',     -- YouTube embed URL
    has_dependencies    BOOLEAN NOT NULL DEFAULT false,
    dependencies        TEXT NOT NULL DEFAULT '',     -- HTML
    createmod_version_id TEXT REFERENCES createmod_versions(id),
    minecraft_version_id TEXT REFERENCES minecraft_versions(id),
    views               INTEGER NOT NULL DEFAULT 0,
    downloads           INTEGER NOT NULL DEFAULT 0,
    block_count         INTEGER NOT NULL DEFAULT 0,
    dim_x               INTEGER NOT NULL DEFAULT 0,
    dim_y               INTEGER NOT NULL DEFAULT 0,
    dim_z               INTEGER NOT NULL DEFAULT 0,
    materials           JSONB NOT NULL DEFAULT '[]',  -- [{name, namespace, count}]
    mods                JSONB NOT NULL DEFAULT '[]',  -- ["namespace1", "namespace2"]
    paid                BOOLEAN NOT NULL DEFAULT false,
    featured            BOOLEAN NOT NULL DEFAULT false,
    ai_description      TEXT NOT NULL DEFAULT '',
    moderated           BOOLEAN NOT NULL DEFAULT false,
    moderation_reason   TEXT NOT NULL DEFAULT '',
    blacklisted         BOOLEAN NOT NULL DEFAULT false,
    scheduled_at        TIMESTAMPTZ,
    deleted             TIMESTAMPTZ,
    deleted_at          TIMESTAMPTZ,
    -- Legacy WordPress fields (kept for backward compat)
    old_id              INTEGER,
    status              TEXT NOT NULL DEFAULT '',
    type                TEXT NOT NULL DEFAULT '',
    created             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schematics_author ON schematics (author_id);
CREATE INDEX idx_schematics_name ON schematics (name);
CREATE INDEX idx_schematics_title ON schematics (title);
CREATE INDEX idx_schematics_type_created ON schematics (type, created);
CREATE INDEX idx_schematics_moderated ON schematics (moderated) WHERE deleted IS NULL;
CREATE INDEX idx_schematics_deleted ON schematics (deleted) WHERE deleted IS NULL;

-- ============================================================
-- SCHEMATICS <-> CATEGORIES (junction table)
-- Replaces PocketBase's JSON relation array
-- ============================================================
CREATE TABLE schematics_categories (
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    category_id     TEXT NOT NULL REFERENCES schematic_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (schematic_id, category_id)
);
CREATE INDEX idx_schematics_categories_cat ON schematics_categories (category_id);

-- ============================================================
-- SCHEMATICS <-> TAGS (junction table)
-- ============================================================
CREATE TABLE schematics_tags (
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    tag_id          TEXT NOT NULL REFERENCES schematic_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (schematic_id, tag_id)
);
CREATE INDEX idx_schematics_tags_tag ON schematics_tags (tag_id);

-- ============================================================
-- SCHEMATIC VIEWS (aggregated view counts)
-- ============================================================
CREATE TABLE schematic_views (
    id              TEXT PRIMARY KEY,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    period          TEXT NOT NULL,       -- 'total', 'daily', etc.
    type            TEXT NOT NULL DEFAULT '',
    count           INTEGER NOT NULL DEFAULT 0,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schematic_views_schematic ON schematic_views (schematic_id);
CREATE INDEX idx_schematic_views_period ON schematic_views (schematic_id, period);

-- ============================================================
-- SCHEMATIC RATINGS
-- ============================================================
CREATE TABLE schematic_ratings (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    rating          REAL NOT NULL DEFAULT 0,
    rated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schematic_ratings_schematic ON schematic_ratings (schematic_id);
CREATE UNIQUE INDEX idx_schematic_ratings_user_schematic ON schematic_ratings (user_id, schematic_id);

-- ============================================================
-- SCHEMATIC DOWNLOADS (tracking)
-- ============================================================
CREATE TABLE schematic_downloads (
    id              TEXT PRIMARY KEY,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    user_id         TEXT REFERENCES users(id) ON DELETE SET NULL,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schematic_downloads_schematic ON schematic_downloads (schematic_id);

-- ============================================================
-- SCHEMATIC VERSIONS (version history / snapshots)
-- ============================================================
CREATE TABLE schematic_versions (
    id              TEXT PRIMARY KEY,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    snapshot        TEXT NOT NULL,       -- JSON blob of changed fields
    note            TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_schematic_versions_schematic_version
    ON schematic_versions (schematic_id, version);

-- ============================================================
-- SCHEMATIC FILES (additional files attached to schematics)
-- ============================================================
CREATE TABLE schematic_files (
    id              TEXT PRIMARY KEY,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL DEFAULT '',
    original_name   TEXT NOT NULL DEFAULT '',
    size            BIGINT NOT NULL DEFAULT 0,
    mime_type       TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schematic_files_schematic ON schematic_files (schematic_id);

-- ============================================================
-- SCHEMATIC TRANSLATIONS
-- ============================================================
CREATE TABLE schematic_translations (
    id              TEXT PRIMARY KEY,
    schematic_id    TEXT NOT NULL,       -- text reference, not FK (schematic may be migrated separately)
    language        TEXT NOT NULL,       -- e.g. 'en', 'pt-BR', 'de'
    title           TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_schematic_translations_schematic_lang
    ON schematic_translations (schematic_id, language);
CREATE INDEX idx_schematic_translations_schematic ON schematic_translations (schematic_id);
CREATE INDEX idx_schematic_translations_language ON schematic_translations (language);

-- ============================================================
-- NBT HASHES (duplicate detection)
-- ============================================================
CREATE TABLE nbt_hashes (
    id              TEXT PRIMARY KEY,
    hash            TEXT NOT NULL,
    schematic_id    TEXT REFERENCES schematics(id) ON DELETE CASCADE,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_nbt_hashes_hash ON nbt_hashes (hash);

-- ============================================================
-- COMMENTS
-- ============================================================
CREATE TABLE comments (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    schematic_id    TEXT REFERENCES schematics(id) ON DELETE CASCADE,
    parent_id       TEXT REFERENCES comments(id) ON DELETE SET NULL,
    content         TEXT NOT NULL DEFAULT '',
    published       TIMESTAMPTZ,
    approved        BOOLEAN NOT NULL DEFAULT false,
    type            TEXT NOT NULL DEFAULT 'comment',
    karma           INTEGER NOT NULL DEFAULT 0,
    -- Legacy WordPress fields
    postdate        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL DEFAULT '',    -- legacy author name for imported comments
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_comments_schematic ON comments (schematic_id);
CREATE INDEX idx_comments_author ON comments (author_id);
CREATE INDEX idx_comments_parent ON comments (parent_id);

-- ============================================================
-- GUIDES
-- ============================================================
CREATE TABLE guides (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    title           TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL DEFAULT '',
    upload_link     TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_guides_slug ON guides (slug) WHERE slug != '';

-- ============================================================
-- GUIDE TRANSLATIONS
-- ============================================================
CREATE TABLE guide_translations (
    id              TEXT PRIMARY KEY,
    guide_id        TEXT NOT NULL,
    language        TEXT NOT NULL,
    title           TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_guide_translations_guide_lang
    ON guide_translations (guide_id, language);
CREATE INDEX idx_guide_translations_guide ON guide_translations (guide_id);
CREATE INDEX idx_guide_translations_language ON guide_translations (language);

-- ============================================================
-- COLLECTIONS (user-curated schematic collections)
-- ============================================================
CREATE TABLE collections (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    title           TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    banner_url      TEXT NOT NULL DEFAULT '',
    featured        BOOLEAN NOT NULL DEFAULT false,
    views           INTEGER NOT NULL DEFAULT 0,
    published       BOOLEAN NOT NULL DEFAULT false,
    deleted         TEXT NOT NULL DEFAULT '',     -- soft delete (text, matching PB pattern)
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_collections_slug ON collections (slug) WHERE slug != '';

-- ============================================================
-- COLLECTIONS <-> SCHEMATICS (junction table with ordering)
-- ============================================================
CREATE TABLE collections_schematics (
    collection_id   TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    position        INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, schematic_id)
);
CREATE INDEX idx_collections_schematics_schematic ON collections_schematics (schematic_id);

-- ============================================================
-- COLLECTION TRANSLATIONS
-- ============================================================
CREATE TABLE collection_translations (
    id              TEXT PRIMARY KEY,
    collection_id   TEXT NOT NULL,
    language        TEXT NOT NULL,
    title           TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_collection_translations_collection_lang
    ON collection_translations (collection_id, language);

-- ============================================================
-- ACHIEVEMENTS
-- ============================================================
CREATE TABLE achievements (
    id              TEXT PRIMARY KEY,
    key             TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    icon            TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_achievements_key ON achievements (key);

-- ============================================================
-- USER ACHIEVEMENTS (junction)
-- ============================================================
CREATE TABLE user_achievements (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id  TEXT NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_user_achievements_user_achievement
    ON user_achievements (user_id, achievement_id);

-- ============================================================
-- POINT LOG (audit trail of point transactions)
-- ============================================================
CREATE TABLE point_log (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    points          INTEGER NOT NULL DEFAULT 0,
    reason          TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    earned_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_point_log_user ON point_log (user_id);
CREATE UNIQUE INDEX idx_point_log_user_reason ON point_log (user_id, reason);

-- ============================================================
-- API KEYS
-- ============================================================
CREATE TABLE api_keys (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,       -- text reference to user ID
    key_hash        TEXT NOT NULL,       -- bcrypt/sha256 hash of the key
    label           TEXT NOT NULL DEFAULT '',
    last8           TEXT NOT NULL DEFAULT '',  -- last 8 chars for display
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_user ON api_keys (user_id);
CREATE INDEX idx_api_keys_last8 ON api_keys (last8);

-- ============================================================
-- API KEY USAGE
-- ============================================================
CREATE TABLE api_key_usage (
    id              TEXT PRIMARY KEY,
    api_key_id      TEXT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint        TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_key_usage_api_key ON api_key_usage (api_key_id);

-- ============================================================
-- TEMP UPLOADS (staging before publish)
-- ============================================================
CREATE TABLE temp_uploads (
    id              TEXT PRIMARY KEY,
    token           TEXT NOT NULL,
    filename        TEXT NOT NULL,
    size            BIGINT NOT NULL DEFAULT 0,
    checksum        TEXT NOT NULL DEFAULT '',
    parsed_summary  TEXT NOT NULL DEFAULT '',
    nbt_file        TEXT NOT NULL DEFAULT '',
    block_count     INTEGER NOT NULL DEFAULT 0,
    dim_x           INTEGER NOT NULL DEFAULT 0,
    dim_y           INTEGER NOT NULL DEFAULT 0,
    dim_z           INTEGER NOT NULL DEFAULT 0,
    materials       JSONB NOT NULL DEFAULT '[]',
    mods            JSONB NOT NULL DEFAULT '[]',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_temp_uploads_token ON temp_uploads (token);

-- ============================================================
-- TEMP UPLOAD FILES
-- ============================================================
CREATE TABLE temp_upload_files (
    id              TEXT PRIMARY KEY,
    temp_upload_id  TEXT NOT NULL REFERENCES temp_uploads(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL DEFAULT '',
    original_name   TEXT NOT NULL DEFAULT '',
    size            BIGINT NOT NULL DEFAULT 0,
    mime_type       TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_temp_upload_files_upload ON temp_upload_files (temp_upload_id);

-- ============================================================
-- REPORTS (content moderation reports)
-- ============================================================
CREATE TABLE reports (
    id              TEXT PRIMARY KEY,
    target_type     TEXT NOT NULL,       -- 'schematic', 'comment', etc.
    target_id       TEXT NOT NULL,
    reason          TEXT NOT NULL,
    reporter        TEXT NOT NULL DEFAULT '',  -- user ID of reporter
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reports_target ON reports (target_type, target_id);

-- ============================================================
-- SEARCHES (search query analytics)
-- ============================================================
CREATE TABLE searches (
    id              TEXT PRIMARY KEY,
    query           TEXT NOT NULL DEFAULT '',
    results_count   INTEGER NOT NULL DEFAULT 0,
    user_id         TEXT,
    ip_address      TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_searches_created ON searches (created);

-- ============================================================
-- NEWS (blog posts / announcements)
-- ============================================================
CREATE TABLE news (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    title           TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    excerpt         TEXT NOT NULL DEFAULT '',
    postdate        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL DEFAULT '',     -- slug
    type            TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_news_postdate ON news (postdate);

-- ============================================================
-- PAGES (CMS pages)
-- ============================================================
CREATE TABLE pages (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    title           TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    excerpt         TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL DEFAULT '',     -- slug
    status          TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL DEFAULT '',
    postdate        TIMESTAMPTZ,
    modified        TIMESTAMPTZ,
    menu_order      INTEGER NOT NULL DEFAULT 0,
    comment_count   INTEGER NOT NULL DEFAULT 0,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- CONTACT FORM SUBMISSIONS
-- ============================================================
CREATE TABLE contact_form_submissions (
    id              TEXT PRIMARY KEY,
    author_id       TEXT REFERENCES users(id),
    title           TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL DEFAULT '',
    postdate        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- OUTGOING CLICKS (external link tracking)
-- ============================================================
CREATE TABLE outgoing_clicks (
    id              TEXT PRIMARY KEY,
    url             TEXT NOT NULL DEFAULT '',
    source          TEXT NOT NULL DEFAULT '',
    source_id       TEXT NOT NULL DEFAULT '',
    user_id         TEXT,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_outgoing_clicks_created ON outgoing_clicks (created);

-- ============================================================
-- MOD METADATA (Modrinth/CurseForge mod info)
-- ============================================================
CREATE TABLE mod_metadata (
    id              TEXT PRIMARY KEY,
    namespace       TEXT NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    icon_url        TEXT NOT NULL DEFAULT '',
    modrinth_slug   TEXT NOT NULL DEFAULT '',
    modrinth_url    TEXT NOT NULL DEFAULT '',
    curseforge_id   TEXT NOT NULL DEFAULT '',
    curseforge_url  TEXT NOT NULL DEFAULT '',
    source_url      TEXT NOT NULL DEFAULT '',
    last_fetched    TIMESTAMPTZ,
    manually_set    BOOLEAN NOT NULL DEFAULT false,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_mod_metadata_namespace ON mod_metadata (namespace);

-- ============================================================
-- SEED DATA: Default achievements
-- ============================================================
INSERT INTO achievements (id, key, title, description, icon) VALUES
    ('ach_first_upload', 'first_upload', 'First Upload', 'Upload your first schematic', 'upload'),
    ('ach_first_comment', 'first_comment', 'First Comment', 'Post your first comment', 'message-circle'),
    ('ach_first_guide', 'first_guide', 'First Guide', 'Create your first guide', 'book-open'),
    ('ach_first_collection', 'first_collection', 'First Collection', 'Create your first collection', 'folder');

-- ============================================================
-- TRIGGER: Auto-update 'updated' timestamps
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the updated trigger to all tables with an 'updated' column
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT table_name
        FROM information_schema.columns
        WHERE column_name = 'updated'
          AND table_schema = 'public'
          AND table_name NOT IN ('sessions', 'searches', 'outgoing_clicks', 'schematic_downloads', 'temp_upload_files', 'api_key_usage')
    LOOP
        EXECUTE format(
            'CREATE TRIGGER trg_%I_updated BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_updated_column()',
            tbl, tbl
        );
    END LOOP;
END;
$$;
