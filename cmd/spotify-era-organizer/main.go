// Command spotify-era-organizer runs the Spotify Era Organizer web application.
package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/justestif/go-spotify-era-organizer/internal/web"
	webfs "github.com/justestif/go-spotify-era-organizer/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Validate environment variables
	clientID := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("please set SPOTIFY_ID and SPOTIFY_SECRET environment variables")
	}

	// Create sub-filesystems for templates and static files
	templates, err := fs.Sub(webfs.TemplatesFS, "templates")
	if err != nil {
		return fmt.Errorf("creating templates filesystem: %w", err)
	}

	static, err := fs.Sub(webfs.StaticFS, "static")
	if err != nil {
		return fmt.Errorf("creating static filesystem: %w", err)
	}

	// Create and start server
	server, err := web.NewServer(web.ServerConfig{
		Addr:         web.DefaultAddr,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TemplatesFS:  templates,
		StaticFS:     static,
	})
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return server.Run()
}
