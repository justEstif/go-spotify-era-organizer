// Package clustering implements tag-based mood clustering for music tracks.
package clustering

import (
	"time"
)

// Track represents a song with its metadata and tags.
type Track struct {
	ID      string
	Name    string
	Artist  string
	AddedAt time.Time
	Tags    []Tag // Tags from Last.fm (empty if not fetched or unavailable)
}

// Tag represents a music tag with popularity count.
type Tag struct {
	Name  string
	Count int
}
