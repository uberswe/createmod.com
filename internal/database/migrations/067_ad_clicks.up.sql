CREATE TABLE IF NOT EXISTS ad_clicks (
    ad_unit   TEXT NOT NULL,
    dest      TEXT NOT NULL DEFAULT '',
    period    TEXT NOT NULL,
    count     BIGINT NOT NULL DEFAULT 1,
    created   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ad_clicks_unit_dest_period
    ON ad_clicks (ad_unit, dest, period);
