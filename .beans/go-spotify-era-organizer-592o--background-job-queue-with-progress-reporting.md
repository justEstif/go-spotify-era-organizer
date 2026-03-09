---
# go-spotify-era-organizer-592o
title: Background Job Queue with Progress Reporting
status: completed
type: feature
priority: normal
created_at: 2026-03-09T17:49:18Z
updated_at: 2026-03-09T18:27:29Z
parent: go-spotify-era-organizer-mr7q
---

Move sync and analysis to a proper background worker with progress reporting via SSE. Replace the current goroutine-based approach with a persistent job queue that survives restarts.

## Summary\nIn-memory job queue with SSE progress, sync/analyze/recluster handlers, HTMX progress bars.
