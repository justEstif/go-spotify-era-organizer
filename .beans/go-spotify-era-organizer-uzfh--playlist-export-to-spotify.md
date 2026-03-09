---
# go-spotify-era-organizer-uzfh
title: Playlist Export to Spotify
status: completed
type: feature
priority: high
created_at: 2026-03-09T17:48:05Z
updated_at: 2026-03-09T18:09:43Z
parent: go-spotify-era-organizer-ghgt
---

Add a button on each era to create a Spotify playlist from its tracks. Wire up the existing internal/spotify/playlist.go to the UI. Include playlist naming (based on era name/tags) and a success confirmation.

## Summary of Changes\nAdded ExportToPlaylist service, POST /eras/{id}/playlist handler, playlist-link HTMX partial. Create Playlist button now functional.
