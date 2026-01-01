// Command spotify-era-organizer analyzes Spotify liked songs and creates mood-based playlists.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/justestif/go-spotify-era-organizer/internal/auth"
	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
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

// toMoodConfig converts CLI config to clustering.MoodConfig.
func (c Config) toMoodConfig() clustering.MoodConfig {
	return clustering.MoodConfig{
		NumClusters:    c.NumClusters,
		MinClusterSize: c.MinClusterSize,
	}
}

// extractTrackIDs extracts Spotify track IDs from a slice of tracks.
func extractTrackIDs(tracks []clustering.Track) []string {
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.ID
	}
	return ids
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

	// Fetch audio features for mood-based clustering
	fmt.Println("\nFetching audio features...")
	if err := client.FetchAudioFeatures(ctx, tracks); err != nil {
		return fmt.Errorf("fetching audio features: %w", err)
	}

	// Detect mood-based eras
	fmt.Println("\nAnalyzing moods and clustering tracks...")
	moodCfg := cfg.toMoodConfig()
	eras, outliers := clustering.DetectMoodEras(tracks, moodCfg)

	// Apply limit if set
	totalEras := len(eras)
	if cfg.Limit > 0 && len(eras) > cfg.Limit {
		eras = eras[:cfg.Limit]
		fmt.Printf("Showing %d of %d eras (use --limit=0 for all)\n", cfg.Limit, totalEras)
	}

	// Display summary
	fmt.Println()
	fmt.Print(clustering.FormatMoodEraSummary(eras, outliers))

	if cfg.DryRun {
		fmt.Println("\nDry-run mode: no playlists created.")
		return nil
	}

	// Create playlists for each era
	if len(eras) == 0 {
		fmt.Println("\nNo eras to create playlists for.")
		return nil
	}

	fmt.Println("\nCreating playlists...")
	for i, era := range eras {
		// Use the mood-based era name directly (includes date range)
		playlistName := era.Name

		// Create the playlist (private, no description)
		playlistID, err := client.CreatePlaylist(ctx, playlistName, "", false)
		if err != nil {
			return fmt.Errorf("creating playlist %q: %w", playlistName, err)
		}

		// Add tracks to the playlist
		trackIDs := extractTrackIDs(era.Tracks)
		if err := client.AddTracksToPlaylist(ctx, playlistID, trackIDs); err != nil {
			return fmt.Errorf("adding tracks to playlist %q: %w", playlistName, err)
		}

		fmt.Printf("Created playlist %d/%d: %q (%d tracks)\n", i+1, len(eras), playlistName, len(era.Tracks))
	}

	fmt.Printf("\nDone! Created %d playlists.\n", len(eras))
	return nil
}
