package clustering

import (
	"testing"
	"time"
)

func TestDetectMoodEras(t *testing.T) {
	// Helper to create a track with audio features
	makeTrack := func(id string, energy, valence, danceability, acousticness float32) Track {
		return Track{
			ID:           id,
			Name:         "Track " + id,
			Artist:       "Artist",
			AddedAt:      time.Now(),
			Energy:       &energy,
			Valence:      &valence,
			Danceability: &danceability,
			Acousticness: &acousticness,
		}
	}

	// Helper to create a track without audio features
	makeTrackNoFeatures := func(id string) Track {
		return Track{
			ID:      id,
			Name:    "Track " + id,
			Artist:  "Artist",
			AddedAt: time.Now(),
		}
	}

	tests := []struct {
		name         string
		tracks       []Track
		cfg          MoodConfig
		wantEras     int
		wantOutliers int
	}{
		{
			name:         "empty input",
			tracks:       nil,
			cfg:          DefaultMoodConfig(),
			wantEras:     0,
			wantOutliers: 0,
		},
		{
			name: "fewer tracks than clusters",
			tracks: []Track{
				makeTrack("1", 0.8, 0.7, 0.6, 0.2),
				makeTrack("2", 0.3, 0.4, 0.5, 0.8),
			},
			cfg:          MoodConfig{NumClusters: 5, MinClusterSize: 1},
			wantEras:     0,
			wantOutliers: 2,
		},
		{
			name: "tracks without features become outliers",
			tracks: []Track{
				makeTrackNoFeatures("1"),
				makeTrackNoFeatures("2"),
				makeTrackNoFeatures("3"),
			},
			cfg:          DefaultMoodConfig(),
			wantEras:     0,
			wantOutliers: 3,
		},
		{
			name: "basic clustering with 2 clusters",
			tracks: []Track{
				// High energy cluster
				makeTrack("1", 0.9, 0.8, 0.7, 0.1),
				makeTrack("2", 0.85, 0.75, 0.65, 0.15),
				makeTrack("3", 0.88, 0.82, 0.72, 0.12),
				// Low energy cluster
				makeTrack("4", 0.2, 0.3, 0.4, 0.9),
				makeTrack("5", 0.25, 0.35, 0.45, 0.85),
				makeTrack("6", 0.22, 0.32, 0.42, 0.88),
			},
			cfg:          MoodConfig{NumClusters: 2, MinClusterSize: 2},
			wantEras:     2,
			wantOutliers: 0,
		},
		{
			name: "mixed tracks with some missing features",
			tracks: []Track{
				makeTrack("1", 0.9, 0.8, 0.7, 0.1),
				makeTrack("2", 0.85, 0.75, 0.65, 0.15),
				makeTrack("3", 0.88, 0.82, 0.72, 0.12),
				makeTrackNoFeatures("4"),
				makeTrackNoFeatures("5"),
			},
			cfg:          MoodConfig{NumClusters: 1, MinClusterSize: 2},
			wantEras:     1,
			wantOutliers: 2, // The two tracks without features
		},
		{
			name: "small clusters become outliers",
			tracks: []Track{
				// Main cluster
				makeTrack("1", 0.9, 0.8, 0.7, 0.1),
				makeTrack("2", 0.85, 0.75, 0.65, 0.15),
				makeTrack("3", 0.88, 0.82, 0.72, 0.12),
				makeTrack("4", 0.87, 0.79, 0.69, 0.13),
				// Single outlier track (very different)
				makeTrack("5", 0.1, 0.1, 0.1, 0.9),
			},
			cfg:          MoodConfig{NumClusters: 2, MinClusterSize: 3},
			wantEras:     1,
			wantOutliers: 1, // The single track cluster becomes an outlier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eras, outliers := DetectMoodEras(tt.tracks, tt.cfg)

			if len(eras) != tt.wantEras {
				t.Errorf("got %d eras, want %d", len(eras), tt.wantEras)
			}

			if len(outliers) != tt.wantOutliers {
				t.Errorf("got %d outliers, want %d", len(outliers), tt.wantOutliers)
			}

			// Verify all tracks are accounted for
			totalTracks := len(outliers)
			for _, era := range eras {
				totalTracks += len(era.Tracks)
			}
			if totalTracks != len(tt.tracks) {
				t.Errorf("total tracks = %d, want %d", totalTracks, len(tt.tracks))
			}
		})
	}
}

func TestDefaultMoodConfig(t *testing.T) {
	cfg := DefaultMoodConfig()

	if cfg.NumClusters != 3 {
		t.Errorf("NumClusters = %d, want 3", cfg.NumClusters)
	}

	if cfg.MinClusterSize != 3 {
		t.Errorf("MinClusterSize = %d, want 3", cfg.MinClusterSize)
	}
}

func TestMoodEraHasDateRange(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tracks := []Track{
		{
			ID:           "1",
			Name:         "First",
			Artist:       "Artist",
			AddedAt:      baseTime,
			Energy:       ptr(0.8),
			Valence:      ptr(0.7),
			Danceability: ptr(0.6),
			Acousticness: ptr(0.2),
		},
		{
			ID:           "2",
			Name:         "Second",
			Artist:       "Artist",
			AddedAt:      baseTime.Add(24 * time.Hour),
			Energy:       ptr(0.85),
			Valence:      ptr(0.75),
			Danceability: ptr(0.65),
			Acousticness: ptr(0.25),
		},
		{
			ID:           "3",
			Name:         "Third",
			Artist:       "Artist",
			AddedAt:      baseTime.Add(48 * time.Hour),
			Energy:       ptr(0.82),
			Valence:      ptr(0.72),
			Danceability: ptr(0.62),
			Acousticness: ptr(0.22),
		},
	}

	eras, _ := DetectMoodEras(tracks, MoodConfig{NumClusters: 1, MinClusterSize: 1})

	if len(eras) != 1 {
		t.Fatalf("expected 1 era, got %d", len(eras))
	}

	era := eras[0]

	// Check that date range is set (earliest to latest)
	if era.StartDate.After(era.EndDate) {
		t.Errorf("StartDate %v should be before or equal to EndDate %v", era.StartDate, era.EndDate)
	}

	// Check that name contains date info
	if era.Name == "" {
		t.Error("era Name should not be empty")
	}
}

func TestMoodEraHasCentroid(t *testing.T) {
	tracks := []Track{
		{
			ID:           "1",
			Name:         "Track 1",
			Artist:       "Artist",
			AddedAt:      time.Now(),
			Energy:       ptr(0.8),
			Valence:      ptr(0.7),
			Danceability: ptr(0.6),
			Acousticness: ptr(0.2),
		},
		{
			ID:           "2",
			Name:         "Track 2",
			Artist:       "Artist",
			AddedAt:      time.Now(),
			Energy:       ptr(0.9),
			Valence:      ptr(0.8),
			Danceability: ptr(0.7),
			Acousticness: ptr(0.1),
		},
	}

	eras, _ := DetectMoodEras(tracks, MoodConfig{NumClusters: 1, MinClusterSize: 1})

	if len(eras) != 1 {
		t.Fatalf("expected 1 era, got %d", len(eras))
	}

	era := eras[0]

	// Check centroid has all expected keys
	expectedKeys := []string{"energy", "valence", "danceability", "acousticness"}
	for _, key := range expectedKeys {
		if _, ok := era.Centroid[key]; !ok {
			t.Errorf("centroid missing key %q", key)
		}
	}
}

// ptr is a helper to create a pointer to a float32
func ptr(f float32) *float32 {
	return &f
}
