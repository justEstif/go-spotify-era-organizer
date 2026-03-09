-- Re-add email column (will be empty; Spotify no longer provides it)
ALTER TABLE users ADD COLUMN email TEXT;
