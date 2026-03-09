---
# go-spotify-era-organizer-ig2a
title: Audit and fix removed Spotify API fields
status: in-progress
type: task
created_at: 2026-03-09T18:02:20Z
updated_at: 2026-03-09T18:02:20Z
---

Remove usage of spotifyUser.Email and drop email column from users table, since Spotify removed email from GET /me response in Feb 2026.
