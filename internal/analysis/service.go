// Package analysis orchestrates the full analysis pipeline:
// sync liked songs → fetch missing tags → detect eras.
package analysis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/eras"
	"github.com/justestif/go-spotify-era-organizer/internal/spotify"
	syncpkg "github.com/justestif/go-spotify-era-organizer/internal/sync"
	"github.com/justestif/go-spotify-era-organizer/internal/tags"
)

// Service orchestrates the full analysis pipeline.
type Service struct {
	db          *db.DB
	syncService *syncpkg.Service
	eraService  *eras.Service
	tagService  *tags.Service // nil if Last.fm not configured
}

// New creates a new analysis service.
func New(database *db.DB, syncSvc *syncpkg.Service, eraSvc *eras.Service, tagSvc *tags.Service) *Service {
	return &Service{
		db:          database,
		syncService: syncSvc,
		eraService:  eraSvc,
		tagService:  tagSvc,
	}
}

// Result contains the outcome of an analysis run.
type Result struct {
	TracksSynced int
	NewTracks    int // Tracks added since last sync (only meaningful for sync operations)
	ErasDetected int
	OutlierCount int
	TotalTracks  int
	SyncedAt     time.Time
	SyncSkipped  bool // True if sync was skipped due to cooldown
}

// RunAnalysis runs the full pipeline: sync → tags → eras.
// If forceSync is false and sync was recent, sync is skipped but tags+eras still run.
func (s *Service) RunAnalysis(ctx context.Context, client *spotify.Client, userID string, forceSync bool) (*Result, error) {
	result := &Result{}

	// Step 1: Sync liked songs
	syncResult, err := s.syncService.SyncLikedSongs(ctx, client, userID, forceSync)
	if err != nil {
		if !errors.Is(err, syncpkg.ErrSyncTooRecent) {
			return nil, fmt.Errorf("syncing liked songs: %w", err)
		}
		result.SyncSkipped = true
		log.Printf("Sync skipped for user %s (recently synced)", userID)
	} else {
		result.TracksSynced = syncResult.TracksCount
		result.NewTracks = syncResult.NewTracks
		result.SyncedAt = syncResult.SyncedAt
		if syncResult.Incremental {
			log.Printf("Incremental sync: %d new tracks for user %s (%d total)", syncResult.NewTracks, userID, syncResult.TracksCount)
		} else {
			log.Printf("Full sync: %d tracks for user %s", syncResult.TracksCount, userID)
		}
	}

	// Step 2: Fetch missing tags
	if err := s.fetchMissingTags(ctx, userID); err != nil {
		log.Printf("Warning: tag fetching failed for user %s: %v", userID, err)
		// Continue — we can still detect eras with whatever tags we have
	}

	// Step 3: Detect and persist eras
	cfg := clustering.DefaultTagClusterConfig()
	eraResult, err := s.eraService.DetectAndPersist(ctx, userID, cfg)
	if err != nil {
		return nil, fmt.Errorf("detecting eras: %w", err)
	}

	result.ErasDetected = len(eraResult.Eras)
	result.OutlierCount = eraResult.OutlierCount
	result.TotalTracks = eraResult.TotalTracks

	log.Printf("Detected %d eras for user %s (%d outliers)", len(eraResult.Eras), userID, eraResult.OutlierCount)

	return result, nil
}

// RunSync syncs liked songs and re-detects eras, respecting the cooldown.
// Returns an error wrapping syncpkg.ErrSyncTooRecent if cooldown hasn't elapsed.
// Also returns the count of new tracks added relative to the previous library size.
func (s *Service) RunSync(ctx context.Context, client *spotify.Client, userID string) (*Result, error) {
	// Check cooldown
	canSync, nextTime, err := s.syncService.CanSync(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checking sync status: %w", err)
	}
	if !canSync {
		return nil, fmt.Errorf("%w: next sync available at %s", syncpkg.ErrSyncTooRecent, nextTime.Format(time.RFC3339))
	}

	// Run the full pipeline with force=false (cooldown already checked)
	// NewTracks is now populated directly by the sync service.
	result, err := s.RunAnalysis(ctx, client, userID, false)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CanSync checks if enough time has passed since the last sync.
func (s *Service) CanSync(ctx context.Context, userID string) (bool, time.Time, error) {
	return s.syncService.CanSync(ctx, userID)
}

// GetLastSyncTime returns the last sync time for a user.
func (s *Service) GetLastSyncTime(ctx context.Context, userID string) (*time.Time, error) {
	return s.syncService.GetLastSyncTime(ctx, userID)
}

// fetchMissingTags fetches Last.fm tags for tracks that don't have any and persists them.
// This is the single source of truth for tag fetching + persistence.
func (s *Service) fetchMissingTags(ctx context.Context, userID string) error {
	if s.tagService == nil {
		log.Printf("Tag service not available, skipping tag fetch for user %s", userID)
		return nil
	}

	// Get all user's tracks
	tracks, err := s.db.Tracks().GetUserTracks(ctx, userID)
	if err != nil {
		return fmt.Errorf("getting user tracks: %w", err)
	}
	if len(tracks) == 0 {
		return nil
	}

	// Find tracks without tags
	trackIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackIDs[i] = t.ID
	}
	missingIDs, err := s.db.Tags().GetTracksWithoutTags(ctx, trackIDs)
	if err != nil {
		return fmt.Errorf("finding tracks without tags: %w", err)
	}
	if len(missingIDs) == 0 {
		return nil
	}

	log.Printf("Fetching tags for %d tracks", len(missingIDs))

	// Build lookup map
	trackMap := make(map[string]db.Track, len(tracks))
	for _, t := range tracks {
		trackMap[t.ID] = t
	}

	// Convert to tag service format
	tagTracks := make([]tags.Track, 0, len(missingIDs))
	for _, id := range missingIDs {
		t, ok := trackMap[id]
		if !ok {
			continue
		}
		tagTracks = append(tagTracks, tags.Track{
			ID:     t.ID,
			Name:   t.Name,
			Artist: t.Artist,
		})
	}

	// Fetch tags concurrently via tag service
	results, err := s.tagService.FetchTagsForTracks(ctx, tagTracks)
	if err != nil {
		return fmt.Errorf("fetching tags: %w", err)
	}

	// Convert and persist
	now := time.Now()
	var dbTags []db.TrackTag
	for _, result := range results {
		if result.Error != nil || len(result.Tags) == 0 {
			continue
		}
		for _, tag := range result.Tags {
			dbTags = append(dbTags, db.TrackTag{
				TrackID:   result.TrackID,
				TagName:   tag.Name,
				TagCount:  tag.Count,
				Source:    string(result.Source),
				FetchedAt: now,
			})
		}
	}

	if len(dbTags) > 0 {
		if err := s.db.Tags().UpsertBatch(ctx, dbTags); err != nil {
			return fmt.Errorf("persisting tags: %w", err)
		}
		log.Printf("Persisted %d tags for user %s", len(dbTags), userID)
	}

	return nil
}
