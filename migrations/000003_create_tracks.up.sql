-- Create tracks table for Spotify track metadata
CREATE TABLE IF NOT EXISTS tracks (
    id              TEXT PRIMARY KEY,                       -- Spotify track ID
    name            TEXT NOT NULL,
    artist          TEXT NOT NULL,                          -- Comma-separated artist names
    album           TEXT,
    album_id        TEXT,                                   -- For album art lookup
    duration_ms     INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
