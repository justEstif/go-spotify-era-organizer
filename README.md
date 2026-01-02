# Spotify Era Organizer

A web application that analyzes your Spotify liked songs, groups them by mood using genre tags, and automatically generates playlists for each "era."

## Problem

Spotify's liked songs grow into an unmanageable list with no context. You naturally go through listening phases with different vibes, but the flat list obscures these patterns.

## Solution

This tool uses Last.fm genre tags to cluster your liked songs by mood using k-means clustering. It then creates private playlists with descriptive names like "rock & indie & alternative: Jan 15 - Feb 3, 2024".

## Development Setup

### Prerequisites

- Go 1.25.5+
- Podman with compose
- [mise](https://mise.jdx.dev/) for tool/env management

### Database

```bash
# Start Postgres
podman compose up -d

# Install tools (including migrate CLI)
mise install

# Run migrations
migrate -path migrations -database "$DATABASE_URL" up

# Stop database
podman compose down

# Reset database (destroy all data)
podman compose down -v
```

### Build

```bash
go build -o spotify-era-organizer ./cmd/spotify-era-organizer
```

## Setup

### 1. Create a Spotify App

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create a new app
3. Add `http://127.0.0.1:8080/callback` as a Redirect URI
4. Note your Client ID and Client Secret

### 2. Set Environment Variables

```bash
export SPOTIFY_ID="your-client-id"
export SPOTIFY_SECRET="your-client-secret"
```

## Usage

Start the web server:

```bash
./spotify-era-organizer
```

Then open your browser to `http://127.0.0.1:8080` and:

1. Click "Login with Spotify" to authenticate
2. View your detected mood eras
3. Create playlists for each era

## How It Works

1. **Authenticate** with Spotify via OAuth (browser-based flow)
2. **Fetch** all your liked songs from Spotify
3. **Fetch genre tags** from Last.fm for each track
4. **Cluster** songs using k-means on tag similarity
5. **Name eras** based on top tags and date ranges
6. **Create** private playlists for each era

## Mood Clustering

Tracks are grouped by their Last.fm genre tags (e.g., "rock", "indie", "electronic"). The k-means algorithm finds natural groupings based on tag similarity, and each cluster is named using its top 3 most prominent tags.

## License

MIT
