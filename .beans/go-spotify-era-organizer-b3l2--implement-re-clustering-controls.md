---
# go-spotify-era-organizer-b3l2
title: Implement re-clustering controls
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:07:46Z
updated_at: 2026-03-09T18:08:55Z
---

Add API endpoint and UI controls for adjusting clustering parameters (num_clusters, min_cluster_size)

## Summary of Changes

- Added `Recluster` handler in `internal/web/handlers.go` — parses `num_clusters` (2-10) and `min_cluster_size` (1-10) from form, validates, calls `eraService.DetectAndPersist` with custom config
- Added `POST /api/recluster` route in `internal/web/server.go`
- Added collapsible controls panel with range sliders in `web/templates/pages/eras.html` (HTMX form, reloads page on success)
- Added glass-morphism CSS for the controls panel matching existing dark theme
