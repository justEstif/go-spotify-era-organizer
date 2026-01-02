package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"
)

const maxTracksPerRequest = 100

// CreatePlaylist creates a new playlist for the current user.
// Returns the playlist ID.
func (c *Client) CreatePlaylist(ctx context.Context, name, description string, public bool) (string, error) {
	userID, err := c.UserID(ctx)
	if err != nil {
		return "", err
	}

	playlist, err := c.api.CreatePlaylistForUser(ctx, userID, name, description, public, false)
	if err != nil {
		return "", fmt.Errorf("creating playlist: %w", err)
	}

	return playlist.ID.String(), nil
}

// AddTracksToPlaylist adds tracks to a playlist, handling batching for large sets.
// Spotify allows max 100 tracks per request.
func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) error {
	if len(trackIDs) == 0 {
		return nil
	}

	// Convert to spotify.ID
	ids := make([]spotify.ID, len(trackIDs))
	for i, id := range trackIDs {
		ids[i] = spotify.ID(id)
	}

	// Batch in chunks of 100
	for i := 0; i < len(ids); i += maxTracksPerRequest {
		end := min(i+maxTracksPerRequest, len(ids))
		batch := ids[i:end]

		_, err := c.api.AddTracksToPlaylist(ctx, spotify.ID(playlistID), batch...)
		if err != nil {
			return fmt.Errorf("adding tracks (batch %d-%d): %w", i+1, end, err)
		}
	}

	return nil
}
