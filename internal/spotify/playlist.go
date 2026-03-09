package spotify

import (
	"context"
	"fmt"
)

const maxTracksPerRequest = 100

// createPlaylistRequest is the JSON body for POST /me/playlists.
type createPlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

// createPlaylistResponse is the JSON response from POST /me/playlists.
type createPlaylistResponse struct {
	ID string `json:"id"`
}

// addItemsRequest is the JSON body for POST /playlists/{id}/items.
type addItemsRequest struct {
	URIs []string `json:"uris"`
}

// CreatePlaylist creates a new playlist for the current user.
// Returns the playlist ID.
func (c *Client) CreatePlaylist(ctx context.Context, name, description string, public bool) (string, error) {
	reqBody := createPlaylistRequest{
		Name:        name,
		Description: description,
		Public:      public,
	}

	var resp createPlaylistResponse
	if err := c.doAPIRequest(ctx, "POST", "/me/playlists", reqBody, &resp); err != nil {
		return "", fmt.Errorf("creating playlist: %w", err)
	}

	return resp.ID, nil
}

// AddTracksToPlaylist adds tracks to a playlist, handling batching for large sets.
// Spotify allows max 100 tracks per request.
func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) error {
	if len(trackIDs) == 0 {
		return nil
	}

	// Batch in chunks of 100
	for i := 0; i < len(trackIDs); i += maxTracksPerRequest {
		end := min(i+maxTracksPerRequest, len(trackIDs))
		batch := trackIDs[i:end]

		uris := make([]string, len(batch))
		for j, id := range batch {
			uris[j] = "spotify:track:" + id
		}

		reqBody := addItemsRequest{URIs: uris}
		path := fmt.Sprintf("/playlists/%s/items", playlistID)

		if err := c.doAPIRequest(ctx, "POST", path, reqBody, nil); err != nil {
			return fmt.Errorf("adding tracks (batch %d-%d): %w", i+1, end, err)
		}
	}

	return nil
}
