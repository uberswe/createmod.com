ALTER TABLE schematics ADD COLUMN short_code TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_schematics_short_code ON schematics(short_code) WHERE short_code != '';

DO $$
DECLARE
    r RECORD;
    alphabet TEXT := 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';
    code TEXT;
    i INT;
    attempts INT;
    code_exists BOOLEAN;
BEGIN
    FOR r IN SELECT id FROM schematics WHERE short_code = '' AND moderation_state != 'deleted' LOOP
        attempts := 0;
        LOOP
            code := '';
            FOR i IN 1..5 LOOP
                code := code || substr(alphabet, floor(random() * length(alphabet) + 1)::int, 1);
            END LOOP;
            SELECT EXISTS(SELECT 1 FROM schematics WHERE short_code = code) INTO code_exists;
            EXIT WHEN NOT code_exists OR attempts > 50;
            attempts := attempts + 1;
        END LOOP;
        IF NOT code_exists THEN
            UPDATE schematics SET short_code = code WHERE id = r.id;
        END IF;
    END LOOP;
END $$;
