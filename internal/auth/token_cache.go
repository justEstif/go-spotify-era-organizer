// Package auth provides Spotify OAuth2 authentication with token caching.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

const (
	configDirName = "spotify-era-organizer"
	tokenFileName = "token.json"
)

// TokenCache handles persistent storage of OAuth tokens.
type TokenCache struct {
	path string
}

// DefaultTokenCache returns a TokenCache using the default location:
// ~/.config/spotify-era-organizer/token.json
func DefaultTokenCache() (*TokenCache, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config dir: %w", err)
	}

	path := filepath.Join(configDir, configDirName, tokenFileName)
	return &TokenCache{path: path}, nil
}

// NewTokenCache creates a TokenCache with a custom path.
func NewTokenCache(path string) *TokenCache {
	return &TokenCache{path: path}
}

// Path returns the file path where tokens are stored.
func (c *TokenCache) Path() string {
	return c.path
}

// Load reads a cached token from disk.
// Returns (nil, nil) if the token file does not exist.
func (c *TokenCache) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading token file: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parsing token file: %w", err)
	}

	return &token, nil
}

// Save writes the token to disk, creating the parent directory if needed.
func (c *TokenCache) Save(token *oauth2.Token) error {
	if token == nil {
		return errors.New("cannot save nil token")
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding token: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}

	return nil
}

// Delete removes the cached token file.
// Returns nil if the file does not exist.
func (c *TokenCache) Delete() error {
	err := os.Remove(c.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing token file: %w", err)
	}
	return nil
}
