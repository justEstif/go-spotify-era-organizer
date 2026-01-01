// Command spotify-era-organizer analyzes Spotify liked songs and creates era-based playlists.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/justestif/go-spotify-era-organizer/internal/auth"
	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
)

// Config holds CLI configuration options.
type Config struct {
	GapDays        int  // Gap threshold in days to split eras
	MinClusterSize int  // Minimum tracks per era
	DryRun         bool // Preview mode (no playlist creation)
}

// parseFlags parses CLI flags and returns configuration.
func parseFlags() Config {
	cfg := Config{}
	flag.IntVar(&cfg.GapDays, "gap", 7, "gap threshold in days to split eras")
	flag.IntVar(&cfg.MinClusterSize, "min-size", 3, "minimum tracks per era")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "preview clusters without creating playlists")
	flag.Parse()
	return cfg
}

// validate checks that configuration values are valid.
func (c Config) validate() error {
	if c.GapDays < 1 {
		return fmt.Errorf("gap must be at least 1 day, got %d", c.GapDays)
	}
	if c.MinClusterSize < 1 {
		return fmt.Errorf("min-size must be at least 1, got %d", c.MinClusterSize)
	}
	return nil
}

// toClusteringConfig converts CLI config to clustering.Config.
func (c Config) toClusteringConfig() clustering.Config {
	return clustering.Config{
		GapThreshold:   time.Duration(c.GapDays) * 24 * time.Hour,
		MinClusterSize: c.MinClusterSize,
	}
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

	client, err := authenticator.Authenticate(ctx)
	if err != nil {
		if errors.Is(err, auth.ErrAuthTimeout) {
			return fmt.Errorf("authentication timed out - please try again")
		}
		return fmt.Errorf("authentication failed: %w", err)
	}

	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("getting user info: %w", err)
	}

	fmt.Printf("Authenticated as: %s\n", user.DisplayName)
	fmt.Println("Authentication successful!")

	// Config is available for use:
	// - cfg.toClusteringConfig() for clustering
	// - cfg.DryRun for preview mode
	_ = cfg.toClusteringConfig() // Will be used by CLI integration (bead dng)

	return nil
}
