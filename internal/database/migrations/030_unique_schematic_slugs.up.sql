-- Fix existing duplicate schematic slugs by appending random suffixes to newer duplicates.
-- For each set of duplicates, keep the oldest (by created) unchanged and rename the rest.
DO $$
DECLARE
    rec RECORD;
    new_name TEXT;
    suffix TEXT;
    attempts INT;
    chars TEXT := 'abcdefghijklmnopqrstuvwxyz0123456789';
BEGIN
    FOR rec IN
        SELECT s.id, s.name, s.created
        FROM schematics s
        INNER JOIN (
            SELECT name
            FROM schematics
            WHERE moderation_state != 'deleted'
            GROUP BY name
            HAVING COUNT(*) > 1
        ) dups ON s.name = dups.name
        WHERE s.moderation_state != 'deleted'
        ORDER BY s.name, s.created ASC
    LOOP
        -- Skip the oldest schematic in each duplicate group (it keeps its name).
        IF NOT EXISTS (
            SELECT 1 FROM schematics s2
            WHERE s2.name = rec.name
              AND s2.moderation_state != 'deleted'
              AND s2.created < rec.created
              AND s2.id != rec.id
        ) THEN
            CONTINUE;
        END IF;

        -- Try appending a single random character, then two, etc.
        attempts := 0;
        LOOP
            attempts := attempts + 1;
            IF attempts > 30 THEN
                -- Fallback: append the schematic ID to guarantee uniqueness
                new_name := rec.name || '-' || rec.id;
                EXIT;
            END IF;
            suffix := '';
            FOR i IN 1..((attempts - 1) / 26 + 1) LOOP
                suffix := suffix || substr(chars, floor(random() * 36 + 1)::int, 1);
            END LOOP;
            new_name := rec.name || '-' || suffix;
            -- Check if this name is already taken
            IF NOT EXISTS (
                SELECT 1 FROM schematics
                WHERE name = new_name AND moderation_state != 'deleted'
            ) THEN
                EXIT;
            END IF;
        END LOOP;

        UPDATE schematics SET name = new_name WHERE id = rec.id;
    END LOOP;
END $$;

-- Now that duplicates are resolved, add a unique partial index on non-deleted schematics.
CREATE UNIQUE INDEX idx_schematics_name_unique ON schematics (name) WHERE moderation_state != 'deleted';
