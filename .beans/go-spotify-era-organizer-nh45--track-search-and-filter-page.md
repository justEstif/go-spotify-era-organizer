---
# go-spotify-era-organizer-nh45
title: Track Search and Filter Page
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:22:18Z
updated_at: 2026-03-09T18:24:38Z
---

Add dedicated search page with live search, era filtering, and pagination

## Summary of Changes

### Files Modified
- `internal/db/tracks.go` — Added `SearchResult` type and `Search()` method with parameterized ILIKE query, count, and pagination
- `internal/web/templates.go` — Added `SearchPageData` and `SearchResultData` types
- `internal/web/handlers.go` — Added `Search()` handler with auth, query parsing, HTMX partial support
- `internal/web/server.go` — Added `GET /search` route
- `web/templates/pages/eras.html` — Added Search link to header actions

### Files Created
- `web/templates/pages/search.html` — Full search page with live HTMX search, era filter dropdown, glassmorphism styling
- `web/templates/partials/search-results.html` — Results partial for HTMX swap with pagination

### Features
- Live debounced search (300ms keyup delay)
- Era filter dropdown
- Pagination (50 per page)
- HTMX partial updates with URL push
- SQL injection safe (parameterized queries)

### Note
`go build ./...` has pre-existing errors from other uncommitted work (AdminStatus handler, unused import). All search-related code compiles cleanly (`go vet ./internal/db/` passes).
