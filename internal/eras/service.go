// Package eras provides services for detecting and persisting listening eras.
package eras

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
	"github.com/justestif/go-spotify-era-organizer/internal/db"
	spotifyclient "github.com/justestif/go-spotify-era-organizer/internal/spotify"
)

// ErrAlreadyExported is returned when an era already has a playlist.
var ErrAlreadyExported = errors.New("era already exported to playlist")

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
	Eras            []db.Era // Detected and persisted eras
	OutlierCount    int      // Number of tracks that didn't fit any era
	OutlierTrackIDs []string // IDs of tracks that didn't fit any era
	TotalTracks     int      // Total tracks analyzed
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

	// Extract outlier track IDs and persist them
	outlierTrackIDs := make([]string, len(outliers))
	for i, o := range outliers {
		outlierTrackIDs[i] = o.ID
	}
	if err := s.db.Outliers().SaveForUser(ctx, userID, outlierTrackIDs); err != nil {
		return nil, fmt.Errorf("saving outlier tracks: %w", err)
	}

	return &DetectResult{
		Eras:            persistedEras,
		OutlierCount:    len(outliers),
		OutlierTrackIDs: outlierTrackIDs,
		TotalTracks:     len(tracks),
	}, nil
}

// RenameEra sets or clears a custom name for an era.
// An empty newName clears the custom name, reverting to the auto-generated name.
func (s *Service) RenameEra(ctx context.Context, userID, eraID, newName string) error {
	id, err := uuid.Parse(eraID)
	if err != nil {
		return fmt.Errorf("invalid era ID: %w", err)
	}

	// Get era and verify ownership
	era, err := s.db.Eras().Get(ctx, id)
	if err != nil {
		return fmt.Errorf("getting era: %w", err)
	}
	if era.UserID != userID {
		return fmt.Errorf("era does not belong to user")
	}

	if newName == "" {
		return s.db.Eras().ClearCustomName(ctx, id)
	}
	return s.db.Eras().UpdateCustomName(ctx, id, newName)
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

// ExportToPlaylist creates a Spotify playlist from an era's tracks.
// Returns the playlist ID. Returns ErrAlreadyExported if the era already has a playlist.
func (s *Service) ExportToPlaylist(ctx context.Context, client *spotifyclient.Client, userID, eraID string) (string, error) {
	id, err := uuid.Parse(eraID)
	if err != nil {
		return "", fmt.Errorf("invalid era ID: %w", err)
	}

	// Get the era and verify ownership
	era, err := s.db.Eras().Get(ctx, id)
	if err != nil {
		return "", fmt.Errorf("getting era: %w", err)
	}
	if era.UserID != userID {
		return "", fmt.Errorf("era does not belong to user")
	}

	// Check if already exported
	if era.PlaylistID != nil {
		return *era.PlaylistID, ErrAlreadyExported
	}

	// Get tracks for the era
	tracks, err := s.db.Eras().GetTracks(ctx, id)
	if err != nil {
		return "", fmt.Errorf("getting era tracks: %w", err)
	}

	// Build description
	tagStr := strings.Join(era.TopTags, ", ")
	description := fmt.Sprintf("Generated by Spotify Era Organizer • %d tracks • %s", len(tracks), tagStr)

	// Create playlist
	playlistID, err := client.CreatePlaylist(ctx, era.Name, description, false)
	if err != nil {
		return "", fmt.Errorf("creating playlist: %w", err)
	}

	// Add tracks
	trackIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackIDs[i] = t.ID
	}
	if err := client.AddTracksToPlaylist(ctx, playlistID, trackIDs); err != nil {
		return "", fmt.Errorf("adding tracks to playlist: %w", err)
	}

	// Persist playlist ID
	if err := s.db.Eras().UpdatePlaylistID(ctx, id, playlistID); err != nil {
		return "", fmt.Errorf("saving playlist ID: %w", err)
	}

	return playlistID, nil
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
