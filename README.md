# Spotify Era Organizer

A CLI tool that analyzes your Spotify liked songs, groups them by mood using audio features, and automatically generates playlists for each "era."

## Problem

Spotify's liked songs grow into an unmanageable list with no context. You naturally go through listening phases with different vibes, but the flat list obscures these patterns.

## Solution

This tool uses Spotify's audio features (energy, valence, danceability, acousticness) to cluster your liked songs by mood using k-means clustering. It then creates private playlists with descriptive names like "Upbeat Party: Jan 15 - Feb 3, 2024" or "Chill & Happy (Acoustic): Mar 1 - Apr 10, 2024".

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

This shows detected mood eras without creating any playlists.

### Create playlists

```bash
spotify-era-organizer
```

Creates playlists for each mood era detected in your liked songs.

### Options

| Flag         | Default | Description                                     |
| ------------ | ------- | ----------------------------------------------- |
| `--dry-run`  | `false` | Preview eras without creating playlists         |
| `--clusters` | `3`     | Number of mood clusters to create               |
| `--min-size` | `3`     | Minimum tracks required per era                 |
| `--limit`    | `0`     | Maximum playlists to create (0 = unlimited)     |

### Examples

```bash
# Preview all detected mood eras
spotify-era-organizer --dry-run

# Create 5 mood-based playlists
spotify-era-organizer --clusters=5

# Require at least 10 tracks per era
spotify-era-organizer --min-size=10

# Create only the first 3 playlists
spotify-era-organizer --limit=3
```

## How It Works

1. **Authenticate** with Spotify (opens browser for OAuth)
2. **Fetch** all your liked songs
3. **Fetch audio features** for each track (energy, valence, danceability, acousticness)
4. **Cluster** songs using k-means on audio features:
   - Groups tracks with similar "vibes" together
   - Uses 4 key features: energy, valence, danceability, acousticness
5. **Name eras** based on mood quadrants:
   - High Energy + High Valence = "Upbeat Party"
   - High Energy + Low Valence = "Intense & Dark"
   - Low Energy + High Valence = "Chill & Happy"
   - Low Energy + Low Valence = "Reflective & Melancholy"
   - High Acousticness adds "(Acoustic)" modifier
6. **Create** private playlists for each era

## Mood Quadrants

| Energy | Valence | Mood Name                  |
| ------ | ------- | -------------------------- |
| High   | High    | Upbeat Party               |
| High   | Low     | Intense & Dark             |
| Low    | High    | Chill & Happy              |
| Low    | Low     | Reflective & Melancholy    |

If acousticness > 60%, "(Acoustic)" is appended to the name.

## Output Example

```
Authenticated as: Your Name

Fetching liked songs...
Found 500 liked songs.

Fetching audio features...
Fetching audio features 1-100 of 500...
Fetching audio features 101-200 of 500...
...
Fetched audio features for 500 tracks.

Analyzing moods and clustering tracks...

Found 3 mood eras from 500 tracks (15 outliers skipped)

Era 1: Upbeat Party: Nov 15, 2024 - Dec 1, 2024 (180 tracks)
  Mood: Energy=78% Valence=72% Danceability=68%
  * "Dance Song" - Artist
  * "Party Track" - Another Artist
  * "Feel Good Hit" - Third Artist
  ... and 177 more

Era 2: Chill & Happy (Acoustic): Sep 1, 2024 - Oct 20, 2024 (165 tracks)
  Mood: Energy=35% Valence=65% Danceability=45%
  * "Acoustic Ballad" - Singer
  * "Mellow Tune" - Band
  * "Soft Song" - Artist
  ... and 162 more

Era 3: Reflective & Melancholy: Jul 1, 2024 - Aug 15, 2024 (140 tracks)
  Mood: Energy=28% Valence=32% Danceability=38%
  * "Sad Song" - Artist
  * "Melancholic Track" - Band
  * "Introspective" - Singer
  ... and 137 more

Creating playlists...
Created playlist 1/3: "Upbeat Party: Nov 15, 2024 - Dec 1, 2024" (180 tracks)
Created playlist 2/3: "Chill & Happy (Acoustic): Sep 1, 2024 - Oct 20, 2024" (165 tracks)
Created playlist 3/3: "Reflective & Melancholy: Jul 1, 2024 - Aug 15, 2024" (140 tracks)

Done! Created 3 playlists.
```

## License

MIT
