package spotify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
)

// FetchAllLikedSongs retrieves all tracks from the user's library.
// Returns tracks as clustering.Track with artists joined by ", ".
// Progress is logged to stdout during fetch.
func (c *Client) FetchAllLikedSongs(ctx context.Context) ([]clustering.Track, error) {
	var tracks []clustering.Track

	// Fetch first page (limit 50 is max per request)
	page, err := c.api.CurrentUsersTracks(ctx, spotify.Limit(50))
	if err != nil {
		return nil, fmt.Errorf("fetching liked songs: %w", err)
	}

	for {
		for _, saved := range page.Tracks {
			track := convertTrack(saved)
			tracks = append(tracks, track)
		}

		fmt.Printf("Fetched %d tracks...\n", len(tracks))

		err = c.api.NextPage(ctx, page)
		if errors.Is(err, spotify.ErrNoMorePages) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fetching next page: %w", err)
		}
	}

	fmt.Printf("Fetched %d tracks total.\n", len(tracks))
	return tracks, nil
}

// convertTrack converts a Spotify SavedTrack to clustering.Track.
func convertTrack(saved spotify.SavedTrack) clustering.Track {
	// Join artist names
	artists := make([]string, len(saved.Artists))
	for i, a := range saved.Artists {
		artists[i] = a.Name
	}

	// Parse AddedAt timestamp, use zero value on failure
	addedAt, _ := time.Parse(time.RFC3339, saved.AddedAt)

	return clustering.Track{
		ID:      saved.ID.String(),
		Name:    saved.Name,
		Artist:  strings.Join(artists, ", "),
		AddedAt: addedAt,
	}
}
