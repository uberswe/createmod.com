-- Fix gravatar URLs that use d=404 (returns 404 for users without a Gravatar account).
-- Replace with d=mm (Mystery Man silhouette) so all users get a visible fallback avatar.
UPDATE users
SET avatar = REPLACE(avatar, 'd=404', 'd=mm')
WHERE avatar LIKE '%d=404%';
