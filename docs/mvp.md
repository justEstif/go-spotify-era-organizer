# Spotify Era Organizer

## Problem

Spotify's liked songs grow into an unmanageable scroll of hundreds or thousands of tracks with no temporal context. Users naturally go through "listening eras" — adding bursts of songs during a particular mood or period — but the flat list obscures these patterns. Manually creating playlists by scrolling through liked songs and remembering when you added each track is tedious and error-prone.

## Solution

A CLI tool that analyzes your Spotify liked songs, detects natural clusters based on when songs were added, and automatically generates playlists for each "era." Playlists are named with exact date ranges (e.g., `2024-01-15 to 2024-02-03`) so you can instantly recall what you were listening to during that period. Outliers — songs that don't fit cleanly into any cluster — are skipped rather than forced into awkward groupings.

## Technical Approach

- **CLI built in Go** using Go's standard library
- **Spotify Web API** via `zmb3/spotify/v2` library with `golang.org/x/oauth2` for auth flow, fetching liked songs (`/me/tracks`), and creating playlists
- **Gap-based clustering algorithm**: sort songs by `added_at`, calculate time gaps between consecutive adds, flag gaps exceeding a threshold (configurable, default ~7-14 days) as cluster boundaries
- **Outlier handling**: clusters below a minimum size (e.g., 3 songs) are skipped; optionally dump outliers to a separate "Uncategorized" playlist or report
- **Local token caching** via JSON file for OAuth refresh tokens so users don't re-auth every run
- **Dry-run mode** to preview clusters before creating playlists
- **Single binary distribution** — compile once, run anywhere without runtime dependencies
