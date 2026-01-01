// Package lastfm provides Last.fm API integration for fetching track tags.
package lastfm

import (
	"errors"
	"os"
)

// ErrMissingAPIKey is returned when LASTFM_API_KEY is not set.
var ErrMissingAPIKey = errors.New("missing LASTFM_API_KEY environment variable")

// Config holds Last.fm API configuration.
type Config struct {
	APIKey string
}

// LoadConfig reads Last.fm configuration from environment variables.
// Returns ErrMissingAPIKey if LASTFM_API_KEY is not set.
func LoadConfig() (*Config, error) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	return &Config{APIKey: apiKey}, nil
}
