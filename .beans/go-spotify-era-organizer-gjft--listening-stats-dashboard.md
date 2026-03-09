---
# go-spotify-era-organizer-gjft
title: Listening Stats Dashboard
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:14:02Z
updated_at: 2026-03-09T18:15:55Z
---

Add /stats page with summary cards, top tags, top artists, monthly activity charts

## Summary of Changes

- Created `internal/db/stats.go` with `StatsRepository` — queries for top tags, top artists, tracks per month, totals, and date range
- Wired `Stats()` method on `DB` in `internal/db/db.go`
- Added `StatsPageData`, `TagStat`, `ArtistStat`, `MonthlyStat` types in `internal/web/templates.go`
- Added `Stats` handler in `internal/web/handlers.go` with auth guard, data computation (percentages relative to max)
- Added `GET /stats` route in `internal/web/server.go`
- Created `web/templates/pages/stats.html` with summary cards, horizontal bar charts (tags/artists), vertical monthly activity chart, empty state, responsive CSS
- Added Stats nav link to eras page header
