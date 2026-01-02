package db

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Spotify user profile.
type User struct {
	ID          string
	DisplayName string
	Email       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastSyncAt  *time.Time // nullable
}

// Session represents an authenticated web session.
type Session struct {
	ID           string
	UserID       string
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Track represents a Spotify track.
type Track struct {
	ID         string
	Name       string
	Artist     string
	Album      *string // nullable
	AlbumID    *string // nullable
	DurationMs *int    // nullable
	CreatedAt  time.Time
}

// UserTrack represents a user's liked track with timestamp.
type UserTrack struct {
	UserID  string
	TrackID string
	AddedAt time.Time
}

// TrackTag represents a Last.fm tag for a track.
type TrackTag struct {
	TrackID   string
	TagName   string
	TagCount  int
	Source    string // "track" or "artist"
	FetchedAt time.Time
}

// Era represents a detected mood era.
type Era struct {
	ID         uuid.UUID
	UserID     string
	Name       string
	TopTags    []string
	StartDate  time.Time
	EndDate    time.Time
	PlaylistID *string // nullable - Spotify playlist ID if created
	CreatedAt  time.Time
}

// EraTrack represents a track belonging to an era.
type EraTrack struct {
	EraID   uuid.UUID
	TrackID string
}
