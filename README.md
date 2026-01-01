# Spotify Era Organizer

A CLI tool that analyzes your Spotify liked songs, detects natural listening "eras" based on when songs were added, and automatically generates playlists for each era.

## Problem

Spotify's liked songs grow into an unmanageable list with no temporal context. You naturally go through listening phases, but the flat list obscures these patterns.

## Solution

This tool clusters your liked songs by their `added_at` timestamps, detecting gaps that indicate the end of one "era" and the start of another. It then creates private playlists named with exact date ranges (e.g., `2024-01-15 to 2024-02-03`).

## Installation

```bash
go install github.com/justestif/go-spotify-era-organizer/cmd/spotify-era-organizer@latest
```

Or build from source:

```bash
git clone https://github.com/justestif/go-spotify-era-organizer.git
cd go-spotify-era-organizer
go build -o spotify-era-organizer ./cmd/spotify-era-organizer
```

## Setup

### 1. Create a Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create a new app
3. Add `http://localhost:8080/callback` as a Redirect URI
4. Note your Client ID and Client Secret

### 2. Set Environment Variables

```bash
export SPOTIFY_ID="your-client-id"
export SPOTIFY_SECRET="your-client-secret"
```

## Usage

### Preview eras (dry-run)

```bash
spotify-era-organizer --dry-run
```

This shows detected eras without creating any playlists.

### Create playlists

```bash
spotify-era-organizer
```

Creates up to 5 playlists for your most recent listening eras (default).

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Preview eras without creating playlists |
| `--limit` | `5` | Maximum playlists to create (0 = unlimited) |
| `--gap` | `7` | Gap threshold in days to split eras |
| `--min-size` | `3` | Minimum tracks required per era |

### Examples

```bash
# Preview all detected eras
spotify-era-organizer --dry-run --limit=0

# Create playlists for the 2 most recent eras
spotify-era-organizer --limit=2

# Use a 14-day gap threshold (longer eras)
spotify-era-organizer --gap=14

# Require at least 5 tracks per era
spotify-era-organizer --min-size=5

# Create all playlists (no limit)
spotify-era-organizer --limit=0
```

## How It Works

1. **Authenticate** with Spotify (opens browser for OAuth)
2. **Fetch** all your liked songs with their `added_at` timestamps
3. **Cluster** songs using gap-based temporal clustering:
   - Sort songs by add date
   - Split into eras when gaps exceed threshold (default: 7 days)
   - Filter out small clusters (default: < 3 tracks)
4. **Create** private playlists for each era (most recent first)

## Output Example

```
Authenticated as: Your Name

Fetching liked songs...
Fetched 500 tracks total.

Detecting eras...
Showing 5 of 12 eras (use --limit=0 for all)

Found 5 eras from 500 tracks (23 outliers skipped)

Era 1: 2024-11-15 to 2024-12-01 (45 tracks)
  • "Song Name" - Artist
  • "Another Song" - Another Artist
  • "Third Song" - Third Artist
  ... and 42 more

Era 2: 2024-10-01 to 2024-10-20 (32 tracks)
...

Creating playlists...
Created playlist 1/5: "2024-11-15 to 2024-12-01" (45 tracks)
Created playlist 2/5: "2024-10-01 to 2024-10-20" (32 tracks)
...

Done! Created 5 playlists.
```

## License

MIT
