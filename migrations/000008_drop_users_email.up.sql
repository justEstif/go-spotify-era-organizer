-- Drop email column: Spotify removed email from GET /me response (Feb 2026)
ALTER TABLE users DROP COLUMN IF EXISTS email;
