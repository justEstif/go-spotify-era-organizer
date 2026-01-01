// Package clustering implements mood-based clustering using audio features.
package clustering

import (
	"time"
)

// Track represents a song with its metadata and audio features.
type Track struct {
	ID      string
	Name    string
	Artist  string
	AddedAt time.Time
	// Audio features (nil if not fetched or unavailable)
	Acousticness     *float32
	Danceability     *float32
	Energy           *float32
	Instrumentalness *float32
	Liveness         *float32
	Loudness         *float32
	Speechiness      *float32
	Tempo            *float32
	Valence          *float32
}
