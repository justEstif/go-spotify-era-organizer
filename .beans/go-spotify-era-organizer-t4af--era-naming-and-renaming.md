---
# go-spotify-era-organizer-t4af
title: Era naming and renaming
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:22:15Z
updated_at: 2026-03-09T18:24:57Z
---

Add custom_name column, rename handler, inline editing

## Summary of Changes

### Files created
- `migrations/000010_era_custom_name.up.sql` — adds `custom_name TEXT` column
- `migrations/000010_era_custom_name.down.sql` — drops the column

### Files modified
- `internal/db/models.go` — added `CustomName *string` to Era struct
- `internal/db/eras.go` — updated all queries/scans (Create, Get, GetForUser) to include `custom_name`; added `UpdateCustomName` and `ClearCustomName` methods
- `internal/eras/service.go` — added `RenameEra` method with ownership check
- `internal/web/handlers.go` — added `RenameEra` handler (PUT /eras/{id}/name); updated Eras and Timeline handlers to pass CustomName
- `internal/web/server.go` — added `Put("/eras/{id}/name")` route
- `internal/web/templates.go` — added `CustomName` to EraData and TimelineEraData; added `displayName` template function
- `web/templates/pages/eras.html` — inline-editable era title with contenteditable, pencil hint, auto-name subtitle, hx-put on blur
- `web/templates/pages/timeline.html` — uses `displayName` for era names
