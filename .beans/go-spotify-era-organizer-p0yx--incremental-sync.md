---
# go-spotify-era-organizer-p0yx
title: Incremental Sync
status: completed
type: feature
priority: high
created_at: 2026-03-09T17:48:18Z
updated_at: 2026-03-09T18:09:43Z
parent: go-spotify-era-organizer-ul8b
---

Only fetch newly liked songs since the last sync timestamp instead of re-fetching the entire library. Store a sync cursor/timestamp and use it on subsequent syncs to reduce API calls.

## Summary of Changes\nAdded FetchLikedSongsSince, refactored sync into fullSync/incrementalSync. Subsequent syncs only fetch new tracks.
