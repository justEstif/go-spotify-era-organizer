# Architecture

Technical design and architecture decisions for Spotify Era Organizer.

## System Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              Web Browser                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                    HTMX + Go Templates                               │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │ │
│  │  │   Home      │  │   Eras      │  │   Sync      │                 │ │
│  │  │   Page      │  │   Page      │  │   Status    │                 │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           Go HTTP Server                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                         internal/web                                 │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │ │
│  │  │ Handlers │  │ Sessions │  │ Templates│  │  Server  │           │ │
│  │  └────┬─────┘  └────┬─────┘  └──────────┘  └──────────┘           │ │
│  └───────┼─────────────┼────────────────────────────────────────────────┘ │
│          │             │                                                  │
│  ┌───────┼─────────────┼────────────────────────────────────────────────┐ │
│  │       ▼             ▼                                                │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │ │
│  │  │   Sync   │  │   Eras   │  │   Tags   │  │Clustering│           │ │
│  │  │ Service  │  │ Service  │  │ Service  │  │  Module  │           │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────┘           │ │
│  │       │             │             │                                  │ │
│  └───────┼─────────────┼─────────────┼──────────────────────────────────┘ │
│          │             │             │                                    │
│  ┌───────┼─────────────┼─────────────┼──────────────────────────────────┐ │
│  │       ▼             ▼             ▼                                  │ │
│  │  ┌────────────────────────────────────────────────────────────────┐ │ │
│  │  │                      internal/db                               │ │ │
│  │  │  Users | Sessions | Tracks | Tags | Eras                       │ │ │
│  │  └────────────────────────────────────────────────────────────────┘ │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
        │                                              │
        ▼                                              ▼
┌───────────────┐                            ┌───────────────┐
│  PostgreSQL   │                            │   External    │
│   Database    │                            │    APIs       │
│               │                            │               │
│ users         │                            │ Spotify API   │
│ sessions      │                            │ Last.fm API   │
│ tracks        │                            │               │
│ user_tracks   │                            └───────────────┘
│ track_tags    │
│ eras          │
│ era_tracks    │
└───────────────┘
```

## Core Components

### Web Layer (`internal/web/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| Server | `server.go` | HTTP server setup, routing, middleware |
| Handlers | `handlers.go` | Request handling, response rendering |
| Sessions | `session.go` | Session management, cookie handling |
| Templates | `templates.go` | Template loading, rendering, functions |

### Service Layer

| Service | Location | Responsibility |
|---------|----------|----------------|
| Sync | `internal/sync/` | Fetches liked songs from Spotify, respects cooldown |
| Eras | `internal/eras/` | Orchestrates era detection and persistence |
| Tags | `internal/tags/` | Fetches and caches Last.fm tags |
| Clustering | `internal/clustering/` | K-means algorithm for era detection |

### Data Layer (`internal/db/`)

| Repository | Entity | Operations |
|------------|--------|------------|
| Users | User profiles | CRUD, last sync tracking |
| Sessions | Auth sessions | Create, validate, delete |
| Tracks | Song metadata | Upsert, batch operations |
| Tags | Last.fm tags | Upsert, cache lookup |
| Eras | Detected eras | CRUD, track associations |

### External Integrations

| Client | Location | API |
|--------|----------|-----|
| Spotify | `internal/spotify/` | OAuth, liked songs, playlists |
| Last.fm | `internal/lastfm/` | Track/artist tags |

## Data Flow

### Authentication Flow

```
Browser                    Server                    Spotify
   │                         │                          │
   │  GET /auth/login        │                          │
   │────────────────────────▶│                          │
   │                         │  Generate state          │
   │                         │  Store server-side       │
   │  302 → Spotify OAuth    │                          │
   │◀────────────────────────│                          │
   │                         │                          │
   │  User authorizes        │                          │
   │─────────────────────────┼─────────────────────────▶│
   │                         │                          │
   │  GET /callback?code=... │                          │
   │────────────────────────▶│                          │
   │                         │  Exchange code for token │
   │                         │─────────────────────────▶│
   │                         │◀─────────────────────────│
   │                         │  Get user profile        │
   │                         │─────────────────────────▶│
   │                         │◀─────────────────────────│
   │                         │  Create session          │
   │  Set-Cookie: session=   │  Trigger initial sync    │
   │◀────────────────────────│                          │
   │  302 → /                │                          │
```

### Sync Flow

```
POST /api/sync
     │
     ▼
┌─────────────────┐
│  Check cooldown │──No──▶ 423 Locked
│  (1 hour)       │
└────────┬────────┘
         │ Yes
         ▼
┌─────────────────┐
│  Fetch liked    │
│  songs from     │
│  Spotify API    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Upsert tracks  │
│  to database    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Find tracks    │
│  without tags   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Fetch tags     │
│  from Last.fm   │
│  (new only)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Run k-means    │
│  clustering     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Persist eras   │
│  to database    │
└────────┬────────┘
         │
         ▼
    200 OK + JSON
```

### Era Detection Algorithm

```
Input: List of tracks with tags

1. Build tag vector for each track
   - Extract all unique tags across tracks
   - Create sparse vector: tag → count

2. Normalize vectors
   - TF-IDF weighting
   - L2 normalization

3. K-means clustering
   - Default k=3 clusters
   - Cosine distance metric
   - Max 100 iterations

4. Filter clusters
   - Remove clusters < 3 tracks (outliers)
   - Tracks without tags → outliers

5. Name clusters
   - Top 3 tags by frequency
   - Date range from track added_at

Output: List of eras with track assignments
```

## Design Decisions

### Why Go Templates + HTMX?

- **Simplicity**: No JavaScript build step, no client-side state management
- **Performance**: Server-rendered HTML, minimal JS payload
- **Maintainability**: Single language (Go), single source of truth
- **Progressive enhancement**: Works without JS, enhanced with HTMX

### Why PostgreSQL?

- **ACID compliance**: Reliable data persistence
- **Array types**: Native support for tag lists (`TEXT[]`)
- **UUID support**: Built-in `gen_random_uuid()` for era IDs
- **Mature ecosystem**: Well-tested Go drivers (pgx)

### Why Last.fm Tags?

Spotify deprecated their Audio Features API (energy, valence, etc.) in November 2024. Last.fm tags provide:
- **User-generated labels**: Real human categorization ("chill", "energetic")
- **Genre information**: Musical style classification
- **Wide coverage**: Extensive catalog of tagged tracks

### Session-Based Authentication

OAuth tokens are stored in sessions rather than user records:
- **Multiple devices**: Users can have multiple active sessions
- **Clean invalidation**: Logout clears specific session
- **Token refresh**: Each session maintains its own token lifecycle

### Sync Cooldown

1-hour cooldown between syncs:
- **Rate limiting**: Respect Spotify API limits
- **Battery/bandwidth**: Prevent excessive polling
- **UX**: Clear feedback when sync unavailable

### Tag Caching

Tags are cached for 30 days:
- **API efficiency**: Reduce Last.fm requests
- **Shared cache**: Tags benefit all users with same tracks
- **Lazy refresh**: Stale tags updated on next access

## Security Model

### Authentication

- OAuth 2.0 with Spotify (no password storage)
- Server-side state validation (CSRF protection)
- Secure session cookies (HttpOnly, SameSite)

### Authorization

- All API endpoints require valid session
- User can only access their own data
- Session expires after 24 hours

### Data Protection

- OAuth tokens encrypted at rest (via database encryption)
- No PII logged (user IDs only)
- Session IDs are cryptographically random (64 hex chars)

## Performance Considerations

### Database

- Indexed foreign keys for fast joins
- Batch upserts for track sync
- Connection pooling via pgx

### Caching

- Tag cache reduces Last.fm API calls by ~95%
- Template compilation at startup
- Static assets served with proper headers

### Concurrency

- Tag fetching uses worker pool (5 concurrent requests)
- Background sync on first login
- Context cancellation for long operations

## Limitations

1. **Single instance**: No distributed session store (sessions stored in PostgreSQL)
2. **Spotify only**: No support for other music services (yet)
3. **No real-time**: No WebSocket/SSE for live updates
4. **English tags**: Last.fm tags are predominantly English
