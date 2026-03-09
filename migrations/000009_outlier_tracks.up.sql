CREATE TABLE IF NOT EXISTS outlier_tracks (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_id TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    detection_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id)
);
CREATE INDEX idx_outlier_tracks_user ON outlier_tracks(user_id);
