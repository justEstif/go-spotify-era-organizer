---
# go-spotify-era-organizer-jqw9
title: Implement Playlist Export to Spotify
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:07:38Z
updated_at: 2026-03-09T18:08:33Z
---

Wire up Create Playlist button to actually create Spotify playlists from eras

## Summary of Changes

- **internal/eras/service.go**: Added `ExportToPlaylist` method that creates a Spotify playlist, adds tracks, and persists the playlist ID. Includes ownership check and duplicate export guard.
- **internal/web/handlers.go**: Added `ExportPlaylist` handler that creates a Spotify client from session token and renders the playlist-link partial.
- **internal/web/server.go**: Added `POST /eras/{id}/playlist` route.
- **web/templates/partials/playlist-link.html**: New HTMX partial with "Open Playlist" link.
- **web/templates/pages/eras.html**: Enabled Create Playlist button with hx-post, hx-target, hx-indicator, and loading spinner.
