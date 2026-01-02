# Spotify Era Organizer

A web application that analyzes your Spotify liked songs, groups them by mood using Last.fm genre tags, and helps you discover the "listening eras" that define your music journey.

## The Problem

Spotify's liked songs grow into an unmanageable list with no context. You naturally go through listening phases with different vibes, but the flat list obscures these patterns. Manually creating playlists by scrolling through hundreds of tracks and trying to group similar-feeling songs is tedious and error-prone.

## The Solution

This tool fetches your liked songs, enriches them with Last.fm genre tags (e.g., "rock", "indie", "electronic", "chill"), and uses k-means clustering to find natural groupings based on tag similarity. Each cluster becomes an "era" named by its top tags and date range (e.g., "rock & indie & alternative: Jan 15 - Feb 3, 2024").

## Features

- **OAuth Authentication** - Secure login with your Spotify account
- **Automatic Sync** - Fetches all your liked songs from Spotify
- **Tag Enrichment** - Gets genre tags from Last.fm for mood-based clustering
- **Era Detection** - Groups songs into eras using k-means on tag similarity
- **Sync Cooldown** - 1-hour cooldown between syncs to respect API rate limits
- **Responsive UI** - HTMX-powered interface with no JavaScript frameworks

## Quick Start

### Prerequisites

- Go 1.25.5+
- PostgreSQL 16+ (or Podman/Docker)
- [mise](https://mise.jdx.dev/) for tool management (optional but recommended)

### 1. Clone and Build

```bash
git clone https://github.com/justEstif/go-spotify-era-organizer.git
cd go-spotify-era-organizer
go build -o spotify-era-organizer ./cmd/spotify-era-organizer
```

### 2. Set Up Database

```bash
# Start Postgres with Podman/Docker
podman compose up -d

# Install migrate tool (if using mise)
mise install

# Run migrations
export DATABASE_URL="postgres://spotify:spotify@localhost:5432/spotify_era?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up
```

### 3. Configure API Keys

#### Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create a new app
3. Add `http://127.0.0.1:8080/callback` as a Redirect URI
4. Note your Client ID and Client Secret

#### Last.fm API (Optional but Recommended)

1. Go to [Last.fm API](https://www.last.fm/api/account/create)
2. Create an API account
3. Note your API Key

### 4. Set Environment Variables

```bash
export SPOTIFY_ID="your-spotify-client-id"
export SPOTIFY_SECRET="your-spotify-client-secret"
export DATABASE_URL="postgres://spotify:spotify@localhost:5432/spotify_era?sslmode=disable"
export LASTFM_API_KEY="your-lastfm-api-key"  # Optional
```

### 5. Run

```bash
./spotify-era-organizer
```

Open your browser to `http://127.0.0.1:8080`.

## Usage

1. **Connect** - Click "Connect with Spotify" to authenticate
2. **Sync** - Your liked songs are automatically synced on first login
3. **View Eras** - Navigate to the Eras page to see your detected listening eras
4. **Refresh** - Use "Refresh Data" to re-sync your library (1-hour cooldown)

## How It Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Spotify   │────▶│   Sync      │────▶│   Last.fm   │────▶│  Clustering │
│   OAuth     │     │   Tracks    │     │   Tags      │     │   K-means   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                                                                   │
                                                                   ▼
                                                           ┌─────────────┐
                                                           │    Eras     │
                                                           │   Display   │
                                                           └─────────────┘
```

1. **OAuth Flow** - Authenticate with Spotify to get access to your library
2. **Track Sync** - Fetch all liked songs from Spotify's `/me/tracks` endpoint
3. **Tag Enrichment** - Fetch genre tags from Last.fm for each track (cached for 30 days)
4. **K-means Clustering** - Group tracks by tag similarity into clusters
5. **Era Naming** - Name each era using its top 3 tags and date range
6. **Display** - Show eras in a responsive web UI with expandable track lists

## Project Structure

```
├── cmd/spotify-era-organizer/  # Application entrypoint
├── internal/
│   ├── clustering/             # K-means era detection algorithm
│   ├── db/                     # PostgreSQL repositories
│   ├── eras/                   # Era detection service
│   ├── lastfm/                 # Last.fm API client
│   ├── spotify/                # Spotify API client wrapper
│   ├── sync/                   # Library sync service
│   ├── tags/                   # Tag enrichment service
│   └── web/                    # HTTP handlers, templates, sessions
├── migrations/                 # PostgreSQL migrations
├── web/
│   ├── static/css/            # Stylesheets
│   └── templates/             # Go HTML templates
└── docs/                      # Documentation
```

## Configuration

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `SPOTIFY_ID` | Yes | - | Spotify app client ID |
| `SPOTIFY_SECRET` | Yes | - | Spotify app client secret |
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `LASTFM_API_KEY` | No | - | Last.fm API key for tag fetching |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Home page |
| `GET` | `/eras` | Eras list page |
| `GET` | `/auth/login` | Initiate Spotify OAuth |
| `GET` | `/callback` | OAuth callback |
| `POST` | `/auth/logout` | Clear session |
| `POST` | `/api/sync` | Trigger library sync |
| `GET` | `/api/sync/status` | Check sync availability |
| `POST` | `/api/analyze` | Run full analysis pipeline |
| `GET` | `/api/eras` | List eras (JSON) |
| `GET` | `/api/eras/{id}/tracks` | Get era tracks (JSON) |

## Documentation

- [Self-Hosting Guide](docs/self-hosting.md) - Deployment instructions
- [Database Schema](docs/schema.md) - PostgreSQL table structure
- [Architecture](docs/architecture.md) - Technical design decisions
- [Implementation Notes](docs/implementation-notes.md) - Go + HTMX patterns

## Tech Stack

- **Backend**: Go 1.25.5+
- **Database**: PostgreSQL 16+
- **Frontend**: Go templates + HTMX
- **APIs**: Spotify Web API, Last.fm API
- **Libraries**: `zmb3/spotify/v2`, `golang.org/x/oauth2`, `muesli/kmeans`

## License

MIT
