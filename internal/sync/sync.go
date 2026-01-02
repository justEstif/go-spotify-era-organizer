// Package sync provides services for syncing data between Spotify and PostgreSQL.
package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/spotify"
)

// Common errors.
var (
	// ErrSyncTooRecent is returned when sync is attempted within the cooldown period.
	ErrSyncTooRecent = errors.New("sync attempted too recently")
)

// DefaultSyncCooldown is the default time between allowed syncs (1 hour).
const DefaultSyncCooldown = 1 * time.Hour

// Service handles syncing data from Spotify to the database.
type Service struct {
	db           *db.DB
	syncCooldown time.Duration
}

// Option configures a Service.
type Option func(*Service)

// WithSyncCooldown sets the minimum time between syncs.
func WithSyncCooldown(d time.Duration) Option {
	return func(s *Service) {
		s.syncCooldown = d
	}
}

// New creates a new sync service.
func New(database *db.DB, opts ...Option) *Service {
	s := &Service{
		db:           database,
		syncCooldown: DefaultSyncCooldown,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	TracksCount int
	SyncedAt    time.Time
}

// CanSync checks if enough time has passed since the last sync.
// Returns true if sync is allowed, false otherwise.
// Also returns the time when the next sync will be available.
func (s *Service) CanSync(ctx context.Context, userID string) (bool, time.Time, error) {
	user, err := s.db.Users().Get(ctx, userID)
	if errors.Is(err, db.ErrNotFound) {
		// New user, allow sync
		return true, time.Time{}, nil
	}
	if err != nil {
		return false, time.Time{}, fmt.Errorf("getting user: %w", err)
	}

	if user.LastSyncAt == nil {
		// Never synced, allow
		return true, time.Time{}, nil
	}

	nextSyncTime := user.LastSyncAt.Add(s.syncCooldown)
	if time.Now().Before(nextSyncTime) {
		return false, nextSyncTime, nil
	}

	return true, time.Time{}, nil
}

// SyncLikedSongs fetches all liked songs from Spotify and persists them.
// Returns ErrSyncTooRecent if called within the cooldown period.
// Set force=true to bypass the cooldown check (for first-time sync after login).
func (s *Service) SyncLikedSongs(ctx context.Context, client *spotify.Client, userID string, force bool) (*SyncResult, error) {
	// Check cooldown unless forced
	if !force {
		canSync, nextTime, err := s.CanSync(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !canSync {
			return nil, fmt.Errorf("%w: next sync available at %s", ErrSyncTooRecent, nextTime.Format(time.RFC3339))
		}
	}

	// Fetch all liked songs from Spotify
	spotifyTracks, err := client.FetchAllLikedSongsWithMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching liked songs: %w", err)
	}

	if len(spotifyTracks) == 0 {
		// Update last sync time even if no tracks
		syncTime := time.Now()
		if err := s.db.Users().UpdateLastSync(ctx, userID, syncTime); err != nil {
			return nil, fmt.Errorf("updating last sync: %w", err)
		}
		return &SyncResult{TracksCount: 0, SyncedAt: syncTime}, nil
	}

	// Convert to database types
	dbTracks := make([]db.Track, len(spotifyTracks))
	userTracks := make([]db.UserTrack, len(spotifyTracks))

	for i, st := range spotifyTracks {
		album := st.Album
		albumID := st.AlbumID
		duration := st.DurationMs

		dbTracks[i] = db.Track{
			ID:         st.ID,
			Name:       st.Name,
			Artist:     st.Artist,
			Album:      &album,
			AlbumID:    &albumID,
			DurationMs: &duration,
		}

		userTracks[i] = db.UserTrack{
			UserID:  userID,
			TrackID: st.ID,
			AddedAt: st.AddedAt,
		}
	}

	// Batch upsert tracks
	if err := s.db.Tracks().UpsertBatch(ctx, dbTracks); err != nil {
		return nil, fmt.Errorf("upserting tracks: %w", err)
	}

	// Link tracks to user
	if err := s.db.Tracks().LinkBatchToUser(ctx, userID, userTracks); err != nil {
		return nil, fmt.Errorf("linking tracks to user: %w", err)
	}

	// Update last sync time
	syncTime := time.Now()
	if err := s.db.Users().UpdateLastSync(ctx, userID, syncTime); err != nil {
		return nil, fmt.Errorf("updating last sync: %w", err)
	}

	return &SyncResult{
		TracksCount: len(spotifyTracks),
		SyncedAt:    syncTime,
	}, nil
}

// GetLastSyncTime returns the last sync time for a user.
// Returns nil if the user has never synced.
func (s *Service) GetLastSyncTime(ctx context.Context, userID string) (*time.Time, error) {
	user, err := s.db.Users().Get(ctx, userID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user.LastSyncAt, nil
}
