package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
)

// FetchAudioFeatures retrieves audio features for the given tracks.
// Updates tracks in-place with their audio features.
// Batches requests to max 100 tracks per request per Spotify API limits.
// Tracks without available audio features will have nil feature fields.
func (c *Client) FetchAudioFeatures(ctx context.Context, tracks []clustering.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	// Build ID slice and index map for fast lookup
	ids := make([]spotify.ID, len(tracks))
	indexByID := make(map[string]int, len(tracks))
	for i, t := range tracks {
		ids[i] = spotify.ID(t.ID)
		indexByID[t.ID] = i
	}

	total := len(ids)

	// Fetch in batches of 100
	for i := 0; i < total; i += maxTracksPerRequest {
		end := min(i+maxTracksPerRequest, total)
		batch := ids[i:end]

		fmt.Printf("Fetching audio features %d-%d of %d...\n", i+1, end, total)

		features, err := c.api.GetAudioFeatures(ctx, batch...)
		if err != nil {
			return fmt.Errorf("fetching audio features (batch %d-%d): %w", i+1, end, err)
		}

		// Map features back to tracks
		for _, f := range features {
			if f == nil {
				continue // Track has no audio features
			}
			idx, ok := indexByID[f.ID.String()]
			if !ok {
				continue
			}
			applyAudioFeatures(&tracks[idx], f)
		}
	}

	fmt.Printf("Fetched audio features for %d tracks.\n", total)
	return nil
}

// applyAudioFeatures copies audio feature values to a track.
func applyAudioFeatures(t *clustering.Track, f *spotify.AudioFeatures) {
	t.Acousticness = &f.Acousticness
	t.Danceability = &f.Danceability
	t.Energy = &f.Energy
	t.Instrumentalness = &f.Instrumentalness
	t.Liveness = &f.Liveness
	t.Loudness = &f.Loudness
	t.Speechiness = &f.Speechiness
	t.Tempo = &f.Tempo
	t.Valence = &f.Valence
}
