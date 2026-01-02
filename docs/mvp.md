# Spotify Era Organizer

## Problem

Spotify's liked songs grow into an unmanageable scroll of hundreds or thousands of tracks with no context. Users naturally go through "listening eras" with different moods and vibes, but the flat list obscures these patterns. Manually creating playlists by scrolling through liked songs and trying to group similar-feeling tracks is tedious and error-prone.

## Solution

A web application that analyzes your Spotify liked songs, groups them by mood using genre tags, and automatically generates playlists for each "era." Playlists are named with descriptive mood labels and date ranges (e.g., `rock & indie & pop: Jan 15 - Feb 3, 2024`) so you can instantly recall the vibe and time period. Tracks that don't fit cleanly into any cluster are skipped rather than forced into awkward groupings.

## Technical Approach

- **Web application built in Go** using Go's standard library and HTMX for interactivity
- **Spotify Web API** via `zmb3/spotify/v2` library with `golang.org/x/oauth2` for auth flow, fetching liked songs (`/me/tracks`) and creating playlists
- **Last.fm API** for fetching genre tags for each track
- **K-means clustering algorithm** using `muesli/kmeans` library to group tracks by tag similarity
- **Tag-based era naming** using top 3 most prominent tags from each cluster
- **Outlier handling**: clusters below a minimum size (e.g., 3 songs) are skipped; tracks without tags are treated as outliers
- **Session-based authentication** for OAuth tokens
- **Single binary distribution** - compile once, run anywhere without runtime dependencies
