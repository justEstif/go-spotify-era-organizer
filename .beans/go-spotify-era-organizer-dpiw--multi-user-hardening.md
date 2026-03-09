---
# go-spotify-era-organizer-dpiw
title: Multi-user Hardening
status: completed
type: feature
priority: normal
created_at: 2026-03-09T18:22:14Z
updated_at: 2026-03-09T18:25:35Z
---

Add per-user rate limiting, concurrent sync guard, and admin status endpoint

## Summary of Changes

### 1. Per-user rate limiter (`internal/ratelimit/ratelimit.go`)
- Token bucket implementation with configurable rate/burst (default: 10 req/s, burst 20)
- Per-key rate limiting with automatic token refill
- Cleanup method removes stale entries (runs every 5 minutes via goroutine)
- Full test coverage in `ratelimit_test.go`

### 2. Rate limit middleware (`internal/web/middleware.go`)
- Extracts user ID from session for authenticated requests
- Falls back to IP-based limiting for unauthenticated requests
- Returns 429 Too Many Requests with Retry-After header
- Wired into all routes via `setupMiddleware()`

### 3. Concurrent sync guard (`internal/sync/sync.go`)
- Added `sync.Map`-based per-user mutex locks
- `acquireUserLock()` uses `TryLock()` for non-blocking acquisition
- New `ErrSyncInProgress` sentinel error
- Handler returns 409 Conflict when sync already running

### 4. Admin status endpoint (`GET /api/admin/status`)
- Returns JSON with total users, active sessions, recent syncs (24h)
- Protected by `Authorization: Bearer <ADMIN_TOKEN>` header
- Returns 404 if `ADMIN_TOKEN` env var not set
- New DB queries: `UserRepository.Count()`, `UserRepository.RecentlyActive()`, `SessionRepository.ActiveCount()`

### 5. ServerConfig updated
- Added `AdminToken` field to `ServerConfig` and `HandlerDeps`

### Files changed
- `internal/ratelimit/ratelimit.go` (new)
- `internal/ratelimit/ratelimit_test.go` (new)
- `internal/web/middleware.go` (new)
- `internal/web/server.go` (modified)
- `internal/web/handlers.go` (modified)
- `internal/sync/sync.go` (modified)
- `internal/db/users.go` (modified)
- `internal/db/sessions.go` (modified)
