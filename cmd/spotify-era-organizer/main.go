// Command spotify-era-organizer runs the Spotify Era Organizer web application.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/justestif/go-spotify-era-organizer/internal/db"
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

	// Connect to database (optional - gracefully degrade if not available)
	var database *db.DB
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		var err error
		database, err = db.New(context.Background(), databaseURL)
		if err != nil {
			return fmt.Errorf("connecting to database: %w", err)
		}
		defer database.Close()
		log.Println("Connected to PostgreSQL database")
	} else {
		log.Println("Warning: DATABASE_URL not set, using in-memory session storage")
		log.Println("Data will not persist across restarts")
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
		DB:           database,
	})
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return server.Run()
}
