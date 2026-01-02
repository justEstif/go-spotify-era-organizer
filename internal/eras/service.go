// Package eras provides services for detecting and persisting listening eras.
package eras

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
	"github.com/justestif/go-spotify-era-organizer/internal/db"
)

// Service handles era detection and persistence.
type Service struct {
	db *db.DB
}

// New creates a new era service.
func New(database *db.DB) *Service {
	return &Service{db: database}
}

// DetectResult contains the outcome of era detection.
type DetectResult struct {
	Eras         []db.Era // Detected and persisted eras
	OutlierCount int      // Number of tracks that didn't fit any era
	TotalTracks  int      // Total tracks analyzed
}

// DetectAndPersist runs era detection on a user's tracks and saves results.
// This deletes any existing eras for the user before saving new ones.
// Returns an error if the user has no tracks.
func (s *Service) DetectAndPersist(ctx context.Context, userID string, cfg clustering.TagClusterConfig) (*DetectResult, error) {
	// Load user's tracks with added_at timestamps
	userTracks, tracks, err := s.db.Tracks().GetUserTracksWithAddedAt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("loading user tracks: %w", err)
	}

	if len(tracks) == 0 {
		return &DetectResult{
			Eras:         nil,
			OutlierCount: 0,
			TotalTracks:  0,
		}, nil
	}

	// Build track ID list and addedAt map
	trackIDs := make([]string, len(tracks))
	addedAtMap := make(map[string]db.UserTrack, len(userTracks))
	for i, ut := range userTracks {
		trackIDs[i] = ut.TrackID
		addedAtMap[ut.TrackID] = ut
	}

	// Load tags for all tracks
	tagsMap, err := s.db.Tags().GetForTracks(ctx, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("loading track tags: %w", err)
	}

	// Convert to clustering.Track format
	clusteringTracks := make([]clustering.Track, len(tracks))
	for i, t := range tracks {
		ut := addedAtMap[t.ID]
		tags := tagsMap[t.ID]
		clusteringTracks[i] = toClusteringTrack(t, ut, tags)
	}

	// Run era detection algorithm
	moodEras, outliers := clustering.DetectMoodEras(clusteringTracks, cfg)

	// Delete existing eras for user (fresh detection each time)
	if err := s.db.Eras().DeleteForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("deleting existing eras: %w", err)
	}

	// Persist new eras
	persistedEras := make([]db.Era, 0, len(moodEras))
	for _, moodEra := range moodEras {
		dbEra, eraTrackIDs := toDBEra(moodEra, userID)
		if err := s.db.Eras().Create(ctx, &dbEra, eraTrackIDs); err != nil {
			return nil, fmt.Errorf("creating era %q: %w", dbEra.Name, err)
		}
		persistedEras = append(persistedEras, dbEra)
	}

	return &DetectResult{
		Eras:         persistedEras,
		OutlierCount: len(outliers),
		TotalTracks:  len(tracks),
	}, nil
}

// GetUserEras retrieves all persisted eras for a user.
func (s *Service) GetUserEras(ctx context.Context, userID string) ([]db.Era, error) {
	eras, err := s.db.Eras().GetForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user eras: %w", err)
	}
	return eras, nil
}

// GetEraTracks retrieves all tracks for a specific era.
func (s *Service) GetEraTracks(ctx context.Context, eraID string) ([]db.Track, error) {
	id, err := uuid.Parse(eraID)
	if err != nil {
		return nil, fmt.Errorf("invalid era ID: %w", err)
	}
	tracks, err := s.db.Eras().GetTracks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting era tracks: %w", err)
	}
	return tracks, nil
}

// toClusteringTrack converts database types to a clustering.Track.
func toClusteringTrack(track db.Track, userTrack db.UserTrack, tags []db.TrackTag) clustering.Track {
	clusterTags := make([]clustering.Tag, len(tags))
	for i, t := range tags {
		clusterTags[i] = clustering.Tag{
			Name:  t.TagName,
			Count: t.TagCount,
		}
	}
	return clustering.Track{
		ID:      track.ID,
		Name:    track.Name,
		Artist:  track.Artist,
		AddedAt: userTrack.AddedAt,
		Tags:    clusterTags,
	}
}

// toDBEra converts a clustering.MoodEra to a db.Era and track IDs.
func toDBEra(era clustering.MoodEra, userID string) (db.Era, []string) {
	trackIDs := make([]string, len(era.Tracks))
	for i, t := range era.Tracks {
		trackIDs[i] = t.ID
	}
	return db.Era{
		UserID:    userID,
		Name:      era.Name,
		TopTags:   era.TopTags,
		StartDate: era.StartDate,
		EndDate:   era.EndDate,
	}, trackIDs
}
