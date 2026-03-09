# Spotify API Migration Notes

**Date:** March 2026

This document summarizes Spotify Web API changes affecting this project and how we handle them.

## November 2024 Deprecations

Spotify deprecated several endpoints that now return **403 Forbidden** for new apps:

| Endpoint | Status | Impact on This Project |
|----------|--------|----------------------|
| `GET /audio-features` | Deprecated (403) | **None** — we use Last.fm tags for clustering, not audio features |
| `GET /audio-analysis` | Deprecated (403) | **None** — not used |
| `GET /recommendations` | Deprecated (403) | **None** — not used |
| `GET /tracks` (batch) | Removed | **None** — we fetch tracks via `/me/tracks` (saved tracks endpoint) |

## February 2026 Changes

### Dev Mode Restrictions

- **Spotify Premium required** on the app owner's account to use dev mode
- **Max 5 test users** in dev mode (reduced from 25)
- Only **1 dev mode Client ID** per developer account
- Must apply for **extended quota mode** for wider distribution

### Endpoint Changes

- Playlist endpoints renamed: `/playlists/{id}/tracks` → `/playlists/{id}/items`
  - The `zmb3/spotify/v2` library may still use old paths; bypass it for playlist operations if needed

### Removed Response Fields

- `popularity` — removed from track, artist, and album objects
- `email` — removed from user profile
- `country` — removed from user profile
- Various other fields (see Spotify changelog)

### Search Limit

- Maximum results per search request reduced from 50 to **10**

## What We Changed

- **No code changes needed for clustering** — we already use Last.fm tags instead of Spotify audio features
- **Documentation updated** to reflect Premium requirement and user limits
- **README** updated with Known Limitations section
- **AGENTS.md** updated with deprecated endpoint notes

## What to Watch For

- If `zmb3/spotify/v2` library updates to support new playlist endpoints, update our dependency
- If implementing playlist export, use `/playlists/{id}/items` path directly
- Search functionality must respect the 10-result limit; paginate if more results needed

## References

- [Spotify Web API Changelog](https://developer.spotify.com/blog/2024-11-27-changes-to-the-web-api)
- [Spotify Quota Modes](https://developer.spotify.com/documentation/web-api/concepts/quota-modes)
- [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
