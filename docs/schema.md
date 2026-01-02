# Database Schema

PostgreSQL schema for the Spotify Era Organizer.

## Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐
│     users       │       │    sessions     │
├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──────│ user_id (FK)    │
│ display_name    │       │ id (PK)         │
│ email           │       │ access_token    │
│ created_at      │       │ refresh_token   │
│ updated_at      │       │ token_expiry    │
│ last_sync_at    │       │ created_at      │
└────────┬────────┘       │ expires_at      │
         │                └─────────────────┘
         │
         │ 1:N
         ▼
┌─────────────────┐       ┌─────────────────┐
│   user_tracks   │       │     tracks      │
├─────────────────┤       ├─────────────────┤
│ user_id (PK,FK) │       │ id (PK)         │
│ track_id (PK,FK)│──────►│ name            │
│ added_at        │       │ artist          │
└─────────────────┘       │ album           │
                          │ album_id        │
         ┌────────────────│ duration_ms     │
         │                │ created_at      │
         │                └────────┬────────┘
         │                         │
         │ N:M                     │ 1:N
         ▼                         ▼
┌─────────────────┐       ┌─────────────────┐
│      eras       │       │   track_tags    │
├─────────────────┤       ├─────────────────┤
│ id (PK, UUID)   │       │ track_id (PK,FK)│
│ user_id (FK)    │       │ tag_name (PK)   │
│ name            │       │ tag_count       │
│ top_tags[]      │       │ source          │
│ start_date      │       │ fetched_at      │
│ end_date        │       └─────────────────┘
│ playlist_id     │
│ created_at      │
└────────┬────────┘
         │
         │ 1:N
         ▼
┌─────────────────┐
│   era_tracks    │
├─────────────────┤
│ era_id (PK,FK)  │
│ track_id (PK,FK)│
└─────────────────┘
```

## Tables

### users

Spotify user profile data.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Spotify user ID |
| display_name | TEXT | | User's display name |
| email | TEXT | | User's email |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Account creation |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Last profile update |
| last_sync_at | TIMESTAMPTZ | | Last Spotify library sync |

### sessions

Web authentication sessions with OAuth tokens.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Random 64-char hex session ID |
| user_id | TEXT | FK → users, NOT NULL | Session owner |
| access_token | TEXT | NOT NULL | Spotify OAuth access token |
| refresh_token | TEXT | NOT NULL | Spotify OAuth refresh token |
| token_expiry | TIMESTAMPTZ | NOT NULL | Access token expiry |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Session creation |
| expires_at | TIMESTAMPTZ | NOT NULL | Session expiry (24h default) |

**Indexes:**
- `idx_sessions_user` on (user_id)
- `idx_sessions_expires` on (expires_at)

### tracks

Spotify track metadata (shared across users).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Spotify track ID |
| name | TEXT | NOT NULL | Track title |
| artist | TEXT | NOT NULL | Comma-separated artist names |
| album | TEXT | | Album name |
| album_id | TEXT | | Spotify album ID (for artwork) |
| duration_ms | INTEGER | | Track duration |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | First seen |

### user_tracks

Junction table linking users to their liked tracks.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| user_id | TEXT | PK, FK → users | User who liked track |
| track_id | TEXT | PK, FK → tracks | Liked track |
| added_at | TIMESTAMPTZ | NOT NULL | When user liked track |

**Indexes:**
- `idx_user_tracks_added` on (user_id, added_at DESC)

### track_tags

Last.fm tag cache for tracks.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| track_id | TEXT | PK, FK → tracks | Track ID |
| tag_name | TEXT | PK | Tag name (lowercase) |
| tag_count | INTEGER | NOT NULL | Last.fm popularity count |
| source | TEXT | NOT NULL, CHECK IN ('track', 'artist') | Tag source |
| fetched_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Cache timestamp |

**Indexes:**
- `idx_track_tags_track` on (track_id)
- `idx_track_tags_fetched` on (fetched_at)

**Cache Policy:** Tags older than 30 days should be refreshed.

### eras

Detected mood eras from clustering.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Era ID |
| user_id | TEXT | FK → users, NOT NULL | Era owner |
| name | TEXT | NOT NULL | Era name (e.g., "Rock & Indie: Jan 15 - Feb 3") |
| top_tags | TEXT[] | NOT NULL | Top 3 dominant tags |
| start_date | TIMESTAMPTZ | NOT NULL | Earliest track date |
| end_date | TIMESTAMPTZ | NOT NULL | Latest track date |
| playlist_id | TEXT | | Spotify playlist ID (if created) |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Detection timestamp |

**Indexes:**
- `idx_eras_user` on (user_id)
- `idx_eras_playlist` on (playlist_id) WHERE playlist_id IS NOT NULL

### era_tracks

Junction table linking eras to tracks.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| era_id | UUID | PK, FK → eras | Era ID |
| track_id | TEXT | PK, FK → tracks | Track in era |

**Indexes:**
- `idx_era_tracks_era` on (era_id)

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate).

```bash
# Run all migrations
migrate -path migrations -database "$DATABASE_URL" up

# Rollback last migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Check current version
migrate -path migrations -database "$DATABASE_URL" version
```

## Design Decisions

1. **Spotify IDs as primary keys**: Using Spotify's string IDs directly avoids mapping tables and simplifies sync logic.

2. **Shared tracks table**: Tracks are stored once and linked to users via `user_tracks`. This allows tag caching to benefit all users.

3. **OAuth tokens in sessions**: Tokens are tied to web sessions rather than users. This supports multiple sessions per user and clean session invalidation.

4. **Array type for top_tags**: PostgreSQL arrays are simple and sufficient for a small fixed-size list (3 tags).

5. **UUID for eras**: Using UUIDs prevents enumeration attacks and simplifies distributed ID generation.

6. **Cascade deletes**: All foreign keys cascade on delete to ensure data consistency when users are removed.
