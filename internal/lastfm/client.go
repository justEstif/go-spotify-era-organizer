package lastfm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	baseURL   = "http://ws.audioscrobbler.com/2.0/"
	userAgent = "spotify-era-organizer/1.0"
)

// Last.fm API error codes.
const (
	errCodeInvalidParams = 6
	errCodeInvalidAPIKey = 10
	errCodeRateLimited   = 29
)

// Sentinel errors.
var (
	// ErrRateLimited is returned when the API rate limit is exceeded after retries.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrInvalidAPIKey is returned when the API key is invalid.
	ErrInvalidAPIKey = errors.New("invalid API key")
)

// Client is a Last.fm API client with caching and rate limiting.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string

	// In-memory cache: key = "track:{artist}:{track}" or "artist:{artist}"
	cache   map[string][]Tag
	cacheMu sync.RWMutex
}

// NewClient creates a new Last.fm API client from the provided configuration.
func NewClient(cfg *Config) *Client {
	return &Client{
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		cache:   make(map[string][]Tag),
	}
}

// GetTags fetches tags for a track, falling back to artist tags if track has none.
// Results are cached in memory. Returns an empty slice (not nil) if no tags are found.
func (c *Client) GetTags(ctx context.Context, artist, track string) ([]Tag, error) {
	// Try track tags first
	tags, err := c.getTrackTags(ctx, artist, track)
	if err != nil {
		return nil, err
	}

	if len(tags) > 0 {
		return tags, nil
	}

	// Fallback to artist tags
	return c.getArtistTags(ctx, artist)
}

// getTrackTags fetches tags for a specific track (with caching).
func (c *Client) getTrackTags(ctx context.Context, artist, track string) ([]Tag, error) {
	cacheKey := fmt.Sprintf("track:%s:%s", artist, track)

	// Check cache
	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		c.cacheMu.RUnlock()
		return cached, nil
	}
	c.cacheMu.RUnlock()

	// Build request params
	params := url.Values{
		"method":      {"track.getTopTags"},
		"artist":      {artist},
		"track":       {track},
		"autocorrect": {"1"},
		"format":      {"json"},
		"api_key":     {c.apiKey},
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fetching track tags: %w", err)
	}

	var resp trackTagsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing track tags response: %w", err)
	}

	tags := resp.TopTags.Tag
	if tags == nil {
		tags = []Tag{}
	}

	// Cache result
	c.cacheMu.Lock()
	c.cache[cacheKey] = tags
	c.cacheMu.Unlock()

	return tags, nil
}

// getArtistTags fetches tags for an artist (with caching).
func (c *Client) getArtistTags(ctx context.Context, artist string) ([]Tag, error) {
	cacheKey := fmt.Sprintf("artist:%s", artist)

	// Check cache
	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		c.cacheMu.RUnlock()
		return cached, nil
	}
	c.cacheMu.RUnlock()

	// Build request params
	params := url.Values{
		"method":      {"artist.getTopTags"},
		"artist":      {artist},
		"autocorrect": {"1"},
		"format":      {"json"},
		"api_key":     {c.apiKey},
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fetching artist tags: %w", err)
	}

	var resp artistTagsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing artist tags response: %w", err)
	}

	tags := resp.TopTags.Tag
	if tags == nil {
		tags = []Tag{}
	}

	// Cache result
	c.cacheMu.Lock()
	c.cache[cacheKey] = tags
	c.cacheMu.Unlock()

	return tags, nil
}

// doRequest performs an HTTP GET request with retry on rate limit.
// Retries up to 3 times with exponential backoff (1s, 2s, 4s).
func (c *Client) doRequest(ctx context.Context, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + "?" + params.Encode()

	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= len(delays); attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delays[attempt-1]):
			}
		}

		body, err := c.doSingleRequest(ctx, reqURL)
		if err == nil {
			return body, nil
		}

		// Check if we should retry
		if errors.Is(err, ErrRateLimited) {
			lastErr = err
			continue
		}

		// Non-retryable error
		return nil, err
	}

	return nil, lastErr
}

// doSingleRequest performs a single HTTP request.
func (c *Client) doSingleRequest(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Check for API error in response
	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != 0 {
		switch apiErr.Error {
		case errCodeRateLimited:
			return nil, ErrRateLimited
		case errCodeInvalidAPIKey:
			return nil, ErrInvalidAPIKey
		default:
			return nil, fmt.Errorf("API error %d: %s", apiErr.Error, apiErr.Message)
		}
	}

	return body, nil
}
