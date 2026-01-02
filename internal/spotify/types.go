package spotify

import "time"

// FullTrack contains complete track metadata from Spotify.
// Used for syncing to the database with all available fields.
type FullTrack struct {
	ID         string
	Name       string
	Artist     string // Comma-separated artist names
	Album      string
	AlbumID    string
	DurationMs int
	AddedAt    time.Time // When user liked the track
}
