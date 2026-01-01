// Command spotify-era-organizer analyzes Spotify liked songs and creates era-based playlists.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/justestif/go-spotify-era-organizer/internal/auth"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	return nil
}
