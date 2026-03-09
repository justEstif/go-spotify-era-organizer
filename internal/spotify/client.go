// Package spotify provides a wrapper around the Spotify Web API.
package spotify

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zmb3/spotify/v2"
)

// Client wraps the Spotify API client with convenience methods.
type Client struct {
	api        *spotify.Client
	httpClient *http.Client
}

// New creates a new Spotify client wrapper.
// The underlying client should already be authenticated.
// httpClient is the OAuth-authenticated HTTP client used for direct API calls.
func New(api *spotify.Client, httpClient *http.Client) *Client {
	return &Client{api: api, httpClient: httpClient}
}

// UserID returns the current user's Spotify ID.
func (c *Client) UserID(ctx context.Context) (string, error) {
	user, err := c.api.CurrentUser(ctx)
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}
	return user.ID, nil
}
