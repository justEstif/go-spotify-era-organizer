---
# go-spotify-era-organizer-h6li
title: Implement incremental sync for liked songs
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:07:32Z
updated_at: 2026-03-09T18:08:42Z
---

Modify SyncLikedSongs to support incremental fetching

## Summary of Changes

### internal/spotify/tracks.go
- Added `FetchLikedSongsSince(ctx, since)` — fetches liked songs page-by-page, stopping when it hits a track with `AddedAt <= since`

### internal/sync/sync.go
- Updated `SyncResult` with `NewTracks`, `Incremental` fields
- Refactored `SyncLikedSongs` to check last sync time and delegate to `incrementalSync` or `fullSync`
- Extracted `fullSync`, `incrementalSync`, and `persistTracks` helper methods
- Incremental sync only fetches/persists new tracks, then reports total count

### internal/analysis/service.go
- `RunAnalysis` now uses `syncResult.NewTracks` directly instead of calculating before/after diff
- `RunSync` simplified — removed pre-sync track counting logic
- Added incremental vs full sync logging
