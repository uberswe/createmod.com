-- Case-insensitive emails. Application code lowercases on write and looks
-- up on LOWER(email); this migration normalizes existing rows, including
-- automatically merging case-duplicate accounts (the same address
-- registered in different casings), then enforces uniqueness on
-- LOWER(email).
--
-- Merge policy per duplicate group of ACTIVE accounts:
--   keeper = most schematics, then oldest account.
--   Other accounts: authorship (schematics, comments, collections, guides)
--   moves to the keeper; OAuth links move unless the keeper already has
--   that provider; ratings move unless the keeper already rated that
--   schematic (conflicting ones are soft-deleted); sessions are dropped;
--   the account is soft-deleted and its email rewritten to a traceable
--   unique alias so the keeper can own the canonical lowercase address.

DO $$
DECLARE
    grp RECORD;
    keeper_id TEXT;
    dup RECORD;
BEGIN
    FOR grp IN
        SELECT LOWER(email) AS email_lower
        FROM users
        WHERE deleted IS NULL
        GROUP BY LOWER(email)
        HAVING COUNT(*) > 1
    LOOP
        SELECT u.id INTO keeper_id
        FROM users u
        WHERE LOWER(u.email) = grp.email_lower AND u.deleted IS NULL
        ORDER BY (SELECT COUNT(*) FROM schematics s WHERE s.author_id = u.id AND s.deleted IS NULL) DESC,
                 u.created ASC
        LIMIT 1;

        FOR dup IN
            SELECT id FROM users
            WHERE LOWER(email) = grp.email_lower AND deleted IS NULL AND id <> keeper_id
        LOOP
            UPDATE schematics  SET author_id = keeper_id WHERE author_id = dup.id;
            UPDATE comments    SET author_id = keeper_id WHERE author_id = dup.id;
            UPDATE collections SET author_id = keeper_id WHERE author_id = dup.id;
            UPDATE guides      SET author_id = keeper_id WHERE author_id = dup.id;

            UPDATE external_auths ea SET user_id = keeper_id
            WHERE ea.user_id = dup.id
              AND NOT EXISTS (
                  SELECT 1 FROM external_auths k
                  WHERE k.user_id = keeper_id AND k.provider = ea.provider
              );

            UPDATE schematic_ratings r SET user_id = keeper_id
            WHERE r.user_id = dup.id
              AND NOT EXISTS (
                  SELECT 1 FROM schematic_ratings k
                  WHERE k.user_id = keeper_id AND k.schematic_id = r.schematic_id
              );
            UPDATE schematic_ratings SET deleted = NOW()
            WHERE user_id = dup.id AND deleted IS NULL;

            DELETE FROM sessions WHERE user_id = dup.id;

            UPDATE users SET
                deleted = NOW(),
                updated = NOW(),
                email = CASE
                    WHEN position('@' in email) > 1 THEN
                        LOWER(substring(email from 1 for position('@' in email) - 1))
                        || '+merged-' || id || '@'
                        || LOWER(substring(email from position('@' in email) + 1))
                    ELSE 'merged-' || id || '@invalid.local'
                END
            WHERE id = dup.id;
        END LOOP;
    END LOOP;
END $$;

-- Soft-deleted accounts whose email still collides (case-insensitively)
-- with a different account get the same alias treatment.
UPDATE users u SET email = CASE
        WHEN position('@' in email) > 1 THEN
            LOWER(substring(email from 1 for position('@' in email) - 1))
            || '+merged-' || id || '@'
            || LOWER(substring(email from position('@' in email) + 1))
        ELSE 'merged-' || id || '@invalid.local'
    END
WHERE u.deleted IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM users o
      WHERE o.id <> u.id AND LOWER(o.email) = LOWER(u.email)
  );

-- All remaining emails are collision-free: normalize to lowercase.
UPDATE users SET email = LOWER(email) WHERE email <> LOWER(email);

-- Rating aggregates may have changed where conflicting duplicate ratings
-- were soft-deleted; recompute (same statement as migration 074).
UPDATE schematics SET
    avg_rating = COALESCE(sub.avg_r, 0),
    rating_count = COALESCE(sub.cnt, 0)
FROM (
    SELECT schematic_id, AVG(rating)::REAL AS avg_r, COUNT(*)::INTEGER AS cnt
    FROM schematic_ratings
    WHERE deleted IS NULL AND rating BETWEEN 1 AND 5
    GROUP BY schematic_id
) sub
WHERE schematics.id = sub.schematic_id;

-- Enforce case-insensitive uniqueness from here on.
DROP INDEX IF EXISTS idx_users_email_lower;
CREATE UNIQUE INDEX idx_users_email_lower ON users (LOWER(email));
