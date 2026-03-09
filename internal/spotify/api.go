package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var spotifyBaseURL = "https://api.spotify.com/v1"

// setBaseURL overrides the base URL (for testing only).
func setBaseURL(url string) {
	spotifyBaseURL = url
}

// doAPIRequest performs a raw HTTP request to the Spotify API and decodes the
// JSON response into dest (if dest is non-nil). It uses the OAuth-authenticated
// HTTP client stored on Client.
func (c *Client) doAPIRequest(ctx context.Context, method, path string, body any, dest any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	url := spotifyBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
