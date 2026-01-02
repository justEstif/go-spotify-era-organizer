-- Create sessions table for web auth sessions
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,                       -- Random 64-char hex session ID
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token    TEXT NOT NULL,                          -- Spotify OAuth access token
    refresh_token   TEXT NOT NULL,                          -- Spotify OAuth refresh token
    token_expiry    TIMESTAMPTZ NOT NULL,                   -- When access token expires
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL                    -- Session expiry (default 24h)
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
