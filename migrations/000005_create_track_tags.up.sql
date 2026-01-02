-- Create track_tags table for Last.fm tag cache
CREATE TABLE IF NOT EXISTS track_tags (
    track_id        TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    tag_name        TEXT NOT NULL,
    tag_count       INTEGER NOT NULL,                       -- Last.fm popularity count
    source          TEXT NOT NULL CHECK (source IN ('track', 'artist')),
    fetched_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),     -- For cache invalidation
    PRIMARY KEY (track_id, tag_name)
);

-- Index for fetching all tags for a track
CREATE INDEX idx_track_tags_track ON track_tags(track_id);

-- Index for finding stale cache entries
CREATE INDEX idx_track_tags_fetched ON track_tags(fetched_at);
