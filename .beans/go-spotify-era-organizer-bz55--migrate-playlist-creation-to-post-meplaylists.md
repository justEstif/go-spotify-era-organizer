---
# go-spotify-era-organizer-bz55
title: Migrate playlist creation to POST /me/playlists
status: completed
type: task
priority: critical
created_at: 2026-03-09T17:49:42Z
updated_at: 2026-03-09T18:04:53Z
parent: go-spotify-era-organizer-t6b9
---

Update internal/spotify/playlist.go: CreatePlaylistForUser is deprecated. Use the new POST /me/playlists endpoint. Also update AddTracksToPlaylist to use /playlists/{id}/items instead of /playlists/{id}/tracks.

## Summary of Changes\nMigrated CreatePlaylist to POST /me/playlists and AddTracksToPlaylist to POST /playlists/{id}/items via new internal/spotify/api.go. Updated all callers.
