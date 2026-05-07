-- Clear video fields that are not valid YouTube video links.
-- Kept patterns:
--   youtube.com/watch?v=ID  (including m. and www. subdomains)
--   youtube.com/embed/ID
--   youtube.com/shorts/ID
--   youtu.be/ID
--   bare 11-character video ID
UPDATE schematics
SET video = ''
WHERE video != ''
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/watch\?'
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/embed/[A-Za-z0-9_-]'
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/shorts/[A-Za-z0-9_-]'
  AND video !~ '^https?://(www\.)?youtu\.be/[A-Za-z0-9_-]'
  AND video !~ '^[A-Za-z0-9_-]{11}$';

UPDATE collections
SET video = ''
WHERE video != ''
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/watch\?'
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/embed/[A-Za-z0-9_-]'
  AND video !~ '^https?://(www\.|m\.)?youtube\.com/shorts/[A-Za-z0-9_-]'
  AND video !~ '^https?://(www\.)?youtu\.be/[A-Za-z0-9_-]'
  AND video !~ '^[A-Za-z0-9_-]{11}$';
