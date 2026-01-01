// Command spotify-era-organizer analyzes Spotify liked songs and creates mood-based playlists.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/justestif/go-spotify-era-organizer/internal/auth"
	"github.com/justestif/go-spotify-era-organizer/internal/spotify"
)

// Config holds CLI configuration options.
type Config struct {
	NumClusters    int  // Number of mood-based clusters to create
	MinClusterSize int  // Minimum tracks per era
	DryRun         bool // Preview mode (no playlist creation)
	Limit          int  // Maximum playlists to create (0 = unlimited)
}

// parseFlags parses CLI flags and returns configuration.
func parseFlags() Config {
	cfg := Config{}
	flag.IntVar(&cfg.NumClusters, "clusters", 3, "number of mood-based clusters to create")
	flag.IntVar(&cfg.MinClusterSize, "min-size", 3, "minimum tracks per era")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "preview clusters without creating playlists")
	flag.IntVar(&cfg.Limit, "limit", 0, "maximum playlists to create (0 = unlimited)")
	flag.Parse()
	return cfg
}

// validate checks that configuration values are valid.
func (c Config) validate() error {
	if c.NumClusters < 1 {
		return fmt.Errorf("clusters must be at least 1, got %d", c.NumClusters)
	}
	if c.MinClusterSize < 1 {
		return fmt.Errorf("min-size must be at least 1, got %d", c.MinClusterSize)
	}
	if c.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", c.Limit)
	}
	return nil
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	ctx := context.Background()

	authenticator, err := auth.New()
	if err != nil {
		if errors.Is(err, auth.ErrMissingCredentials) {
			return fmt.Errorf("please set SPOTIFY_ID and SPOTIFY_SECRET environment variables")
		}
		return fmt.Errorf("creating authenticator: %w", err)
	}

	apiClient, err := authenticator.Authenticate(ctx)
	if err != nil {
		if errors.Is(err, auth.ErrAuthTimeout) {
			return fmt.Errorf("authentication timed out - please try again")
		}
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Wrap with our client for convenience methods
	client := spotify.New(apiClient)

	user, err := apiClient.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("getting user info: %w", err)
	}

	fmt.Printf("Authenticated as: %s\n", user.DisplayName)

	// Fetch all liked songs
	fmt.Println("\nFetching liked songs...")
	tracks, err := client.FetchAllLikedSongs(ctx)
	if err != nil {
		return fmt.Errorf("fetching liked songs: %w", err)
	}

	if len(tracks) == 0 {
		fmt.Println("No liked songs found.")
		return nil
	}

	fmt.Printf("Found %d liked songs.\n", len(tracks))

	// DEPRECATED: Spotify Audio Features API
	// The Audio Features API was deprecated by Spotify in November 2024 for new apps.
	// Mood-based clustering is being migrated to use Last.fm tags instead.
	// See: https://github.com/justestif/go-spotify-era-organizer/issues (V3 Epic)
	fmt.Println()
	fmt.Println("=================================================================")
	fmt.Println("NOTICE: Mood-based clustering is temporarily unavailable.")
	fmt.Println()
	fmt.Println("Spotify deprecated the Audio Features API in November 2024.")
	fmt.Println("We are migrating to Last.fm tags for mood detection.")
	fmt.Println()
	fmt.Println("Please use the web UI (coming soon) or check back for updates.")
	fmt.Println("=================================================================")

	return nil
}
