# Spotify Era Organizer

## Problem

Spotify's liked songs grow into an unmanageable scroll of hundreds or thousands of tracks with no context. Users naturally go through "listening eras" with different moods and vibes, but the flat list obscures these patterns. Manually creating playlists by scrolling through liked songs and trying to group similar-feeling tracks is tedious and error-prone.

## Solution

A CLI tool that analyzes your Spotify liked songs, groups them by mood using audio features, and automatically generates playlists for each "era." Playlists are named with descriptive mood labels and date ranges (e.g., `Upbeat Party: Jan 15 - Feb 3, 2024`) so you can instantly recall the vibe and time period. Tracks that don't fit cleanly into any cluster are skipped rather than forced into awkward groupings.

## Technical Approach

- **CLI built in Go** using Go's standard library
- **Spotify Web API** via `zmb3/spotify/v2` library with `golang.org/x/oauth2` for auth flow, fetching liked songs (`/me/tracks`), audio features (`/audio-features`), and creating playlists
- **K-means clustering algorithm** using `muesli/kmeans` library to group tracks by audio features:
  - Energy (intensity and activity)
  - Valence (musical positivity)
  - Danceability (how suitable for dancing)
  - Acousticness (acoustic vs electronic)
- **Mood-based era naming** using a quadrant system based on energy/valence with acousticness modifier
- **Outlier handling**: clusters below a minimum size (e.g., 3 songs) are skipped; tracks without audio features are treated as outliers
- **Local token caching** via JSON file for OAuth refresh tokens so users don't re-auth every run
- **Dry-run mode** to preview clusters before creating playlists
- **Single binary distribution** - compile once, run anywhere without runtime dependencies
