---
# go-spotify-era-organizer-6qz7
title: Spotify Audio Features Integration
status: scrapped
type: feature
priority: high
created_at: 2026-03-09T17:48:19Z
updated_at: 2026-03-09T17:49:47Z
parent: go-spotify-era-organizer-ul8b
---

Fetch Spotify audio features (danceability, energy, valence, tempo, acousticness) for each track and use them alongside Last.fm tags as clustering dimensions for richer era detection.

## Reasons for Scrapping

Spotify deprecated the Audio Features endpoint (GET /audio-features) in November 2024. It returns 403 for all new apps. This feature is no longer viable. Alternative: rely on Last.fm tags, MusicBrainz data, or third-party audio analysis services like Cyanite.
