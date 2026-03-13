-- Revert gravatar default from mm back to 404.
UPDATE users
SET avatar = REPLACE(avatar, 'd=mm', 'd=404')
WHERE avatar LIKE '%d=mm%' AND avatar LIKE '%gravatar.com%';
