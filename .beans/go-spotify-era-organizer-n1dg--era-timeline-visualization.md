---
# go-spotify-era-organizer-n1dg
title: Era Timeline Visualization
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:14:02Z
updated_at: 2026-03-09T18:15:35Z
---

Add /timeline page with horizontal chronological timeline of eras, color-coded by mood

## Summary of Changes

- Added `TimelinePageData`, `TimelineEraData`, `TimelineSpan` types in `internal/web/templates.go`
- Added `printf` template function in `defaultFuncs()`
- Added `Timeline` handler in `internal/web/handlers.go` with offset/width computation and color hue hashing
- Added `/timeline` route in `internal/web/server.go`
- Created `web/templates/pages/timeline.html` with horizontal scrollable timeline (desktop) and vertical list (mobile)
- Added 'Timeline View' link to eras page header
