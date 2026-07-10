ALTER TABLE collections ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;
-- Backfill already-published collections so they predate any summary window.
UPDATE collections SET published_at = updated WHERE published = true AND published_at IS NULL;
