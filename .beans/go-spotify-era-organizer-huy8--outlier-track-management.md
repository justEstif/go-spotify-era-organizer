---
# go-spotify-era-organizer-huy8
title: Outlier Track Management
status: completed
type: feature
priority: normal
created_at: 2026-03-09T17:48:05Z
updated_at: 2026-03-09T18:16:00Z
parent: go-spotify-era-organizer-ghgt
---

Display tracks that didn't fit any era cluster (outliers). Allow users to manually assign outliers to existing eras or create a new era from them.

## Summary of Changes

- **Migration**: Added `migrations/000009_outlier_tracks.up.sql` and `.down.sql` with `outlier_tracks` table (composite PK on user_id + track_id, cascading deletes, user index)
- **OutlierRepository** (`internal/db/outliers.go`): SaveForUser, GetForUser, DeleteForUser, Count, AssignToEra — all with proper transactions and ownership verification
- **DB wiring** (`internal/db/db.go`): Added `Outliers()` accessor
- **Era service** (`internal/eras/service.go`): DetectAndPersist now persists outlier track IDs; DetectResult includes OutlierTrackIDs
- **Templates** (`internal/web/templates.go`): ErasPageData now has Outliers and OutlierCount fields
- **Handlers** (`internal/web/handlers.go`): Eras handler loads outliers; new AssignOutlier handler for POST /api/outliers/assign
- **Routes** (`internal/web/server.go`): Added /api/outliers/assign route
- **UI** (`web/templates/pages/eras.html`): Outlier section with scrollable list, per-track era assignment dropdown via HTMX (item disappears on assign), full CSS styling matching dark theme
