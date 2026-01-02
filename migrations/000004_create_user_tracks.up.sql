-- Create user_tracks junction table for liked songs
CREATE TABLE IF NOT EXISTS user_tracks (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_id        TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    added_at        TIMESTAMPTZ NOT NULL,                   -- When user liked the track
    PRIMARY KEY (user_id, track_id)
);

-- Index for fetching user's liked songs sorted by date
CREATE INDEX idx_user_tracks_added ON user_tracks(user_id, added_at DESC);
