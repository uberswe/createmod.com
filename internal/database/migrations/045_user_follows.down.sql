ALTER TABLE users DROP COLUMN IF EXISTS following_count;
ALTER TABLE users DROP COLUMN IF EXISTS follower_count;
DROP TABLE IF EXISTS user_follows;
