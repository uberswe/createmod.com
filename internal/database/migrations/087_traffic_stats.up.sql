-- Raw view/download hit counts aggregated by (day, event type, user agent,
-- country, resolution, page class) for bot-traffic analysis. Written by an
-- in-memory per-pod aggregator that flushes deltas periodically, so this stays
-- small (one row per distinct bucket per day) even under a bot flood. Retained
-- ~30 days by a periodic cleanup job.
--
-- event_type:  'view'    server-side, every HTML page (catches no-JS bots; no resolution)
--              'view_js' the client JS beacon (only JS clients; carries resolution)
--              'download'
-- resolution:  'WxH' from window.screen (view_js only); '' otherwise — '' is itself a bot signal
-- page_class:  coarse page group (home/browse/schematic/generator/...)
-- day is 'YYYY-MM-DD' UTC text to avoid date-array encoding quirks; ISO text sorts chronologically.
CREATE TABLE IF NOT EXISTS traffic_stats (
    day         TEXT   NOT NULL,
    event_type  TEXT   NOT NULL,
    user_agent  TEXT   NOT NULL,
    country     TEXT   NOT NULL DEFAULT '',
    resolution  TEXT   NOT NULL DEFAULT '',
    page_class  TEXT   NOT NULL DEFAULT '',
    count       BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, event_type, user_agent, country, resolution, page_class)
);

-- Supports the retention delete and day-range analytics queries.
CREATE INDEX IF NOT EXISTS idx_traffic_stats_day ON traffic_stats (day);
