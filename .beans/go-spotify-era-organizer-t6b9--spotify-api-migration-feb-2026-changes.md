---
# go-spotify-era-organizer-t6b9
title: Spotify API Migration (Feb 2026 Changes)
status: completed
type: epic
priority: critical
created_at: 2026-03-09T17:49:30Z
updated_at: 2026-03-09T18:03:49Z
---

Migrate to the updated Spotify Web API following the February 2026 changes. Key impacts:

## Deprecated (since Nov 2024)
- **Audio Features** (GET /audio-features) — GONE. Cannot use for clustering.
- **Audio Analysis** (GET /audio-analysis) — GONE.
- **Recommendations** (GET /recommendations) — GONE.
- **Get Several Tracks** (GET /tracks) — REMOVED. Must fetch one at a time.

## February 2026 Changes
- **Dev Mode requires Premium** on the app owner's account
- **Max 5 test users** in dev mode (need extended quota for more)
- **Create Playlist** endpoint changed: POST /users/{id}/playlists → POST /me/playlists
- **Playlist items** renamed: /playlists/{id}/tracks → /playlists/{id}/items
- **Track fields removed**: popularity, available_markets, linked_from
- **Artist fields removed**: popularity, followers
- **User fields removed**: email, country, product, followers, explicit_content
- **Search limit** reduced: max 10 results (was 50), default 5 (was 20)

## Impact on This App
- Audio Features integration feature (#6qz7) is NOT viable — must rely entirely on Last.fm tags + alternative sources
- Playlist creation code needs updating to new endpoints
- zmb3/spotify library may need updates or patches for new endpoints
- Need to check if we depend on any removed fields

## Summary of Changes

- Created `internal/spotify/api.go` with low-level HTTP helper (`doAPIRequest`) for direct Spotify API calls
- Updated `internal/spotify/client.go` to store OAuth `*http.Client` alongside zmb3 client; `New()` now takes both
- Rewrote `internal/spotify/playlist.go` to use new endpoints:
  - `CreatePlaylist` → `POST /me/playlists` (was `/users/{id}/playlists`)
  - `AddTracksToPlaylist` → `POST /playlists/{id}/items` (was `/playlists/{id}/tracks`)
- Updated all 3 call sites in `internal/web/handlers.go` to pass `httpClient` to `spotifyclient.New()`
- Added comprehensive tests in `internal/spotify/playlist_test.go` using httptest
- All checks pass: `go build`, `go vet`, `go test`
