CREATE TABLE IF NOT EXISTS schematic_videos (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id  TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    video_url     TEXT NOT NULL,
    video_type    TEXT NOT NULL DEFAULT 'showcase',
    title         TEXT NOT NULL DEFAULT '',
    position      INTEGER NOT NULL DEFAULT 0,
    created       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_schematic_videos_schematic ON schematic_videos (schematic_id);

-- Migrate existing single video field into new table
INSERT INTO schematic_videos (schematic_id, video_url, video_type, position)
SELECT id, video, 'showcase', 0
FROM schematics WHERE video != '' AND video IS NOT NULL
ON CONFLICT DO NOTHING;
