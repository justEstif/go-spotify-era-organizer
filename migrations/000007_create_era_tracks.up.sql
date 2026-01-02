-- Create era_tracks junction table
CREATE TABLE IF NOT EXISTS era_tracks (
    era_id          UUID NOT NULL REFERENCES eras(id) ON DELETE CASCADE,
    track_id        TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    PRIMARY KEY (era_id, track_id)
);

-- Index for fetching tracks in an era
CREATE INDEX idx_era_tracks_era ON era_tracks(era_id);
