---
# go-spotify-era-organizer-pcg5
title: Background job queue with SSE progress reporting
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:22:23Z
updated_at: 2026-03-09T18:26:42Z
---

Add internal/jobs package with in-memory queue, SSE progress endpoint, and update UI to show progress bars during sync/analysis operations.

## Summary of Changes

### New files
- `internal/jobs/queue.go` — In-memory job queue with submit, subscribe, progress, duplicate prevention, auto-cleanup
- `internal/jobs/handlers.go` — Job handlers wrapping analysis/sync services with Spotify auth token from metadata
- `internal/jobs/queue_test.go` — Tests with race detector: submit, fail, duplicate prevention, subscribe, unregistered handler
- `web/templates/partials/job-progress.html` — SSE-connected progress bar partial with auto-reload on completion

### Modified files
- `internal/web/handlers.go` — Added SubmitSyncJob, SubmitAnalyzeJob, JobProgress (SSE), ActiveJobs handlers; added jobQueue to deps
- `internal/web/server.go` — Created and wired job queue with registered handlers; added 4 new routes
- `web/templates/layouts/base.html` — Added htmx-ext-sse script
- `web/templates/pages/eras.html` — "Analyze My Library" button now uses /api/jobs/analyze with progress bar target
- `web/templates/partials/sync-status.html` — Sync button now uses /api/jobs/sync with progress bar target

### Routes added
- POST /api/jobs/sync — Submit background sync job
- POST /api/jobs/analyze — Submit background analysis job  
- GET /api/jobs/{id}/progress — SSE progress stream
- GET /api/jobs/active — List user's active jobs

### Key design decisions
- Old /api/analyze and /api/sync endpoints preserved for backwards compat
- SSE uses native EventSource in inline script (more reliable than htmx-ext-sse for progress bars)
- Completed jobs auto-cleaned after 1 hour on next submit
- Duplicate prevention returns existing job ID (not an error to the client)
