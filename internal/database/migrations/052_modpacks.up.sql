CREATE TABLE IF NOT EXISTS modpacks (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    modrinth_id   TEXT NOT NULL UNIQUE,
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    icon_url      TEXT NOT NULL DEFAULT '',
    modrinth_url  TEXT NOT NULL DEFAULT '',
    downloads     INTEGER NOT NULL DEFAULT 0,
    last_fetched  TIMESTAMPTZ,
    created       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schematics_modpacks (
    schematic_id  TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    modpack_id    TEXT NOT NULL REFERENCES modpacks(id) ON DELETE CASCADE,
    PRIMARY KEY (schematic_id, modpack_id)
);
CREATE INDEX IF NOT EXISTS idx_schematics_modpacks_modpack ON schematics_modpacks (modpack_id);
