-- Create eras table for detected mood eras
CREATE TABLE IF NOT EXISTS eras (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,                          -- e.g., "Rock & Indie: Jan 15 - Feb 3, 2024"
    top_tags        TEXT[] NOT NULL,                        -- Top 3 dominant tags
    start_date      TIMESTAMPTZ NOT NULL,                   -- Earliest track add date
    end_date        TIMESTAMPTZ NOT NULL,                   -- Latest track add date
    playlist_id     TEXT,                                   -- Spotify playlist ID (if created)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fetching user's eras
CREATE INDEX idx_eras_user ON eras(user_id);

-- Index for finding eras by playlist
CREATE INDEX idx_eras_playlist ON eras(playlist_id) WHERE playlist_id IS NOT NULL;
