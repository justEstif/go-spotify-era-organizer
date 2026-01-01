package spotify

import (
	"testing"

	"github.com/zmb3/spotify/v2"

	"github.com/justestif/go-spotify-era-organizer/internal/clustering"
)

func TestApplyAudioFeatures(t *testing.T) {
	track := clustering.Track{ID: "test123", Name: "Test Song"}
	features := &spotify.AudioFeatures{
		Acousticness:     0.5,
		Danceability:     0.7,
		Energy:           0.8,
		Instrumentalness: 0.1,
		Liveness:         0.2,
		Loudness:         -5.0,
		Speechiness:      0.05,
		Tempo:            120.0,
		Valence:          0.6,
	}

	applyAudioFeatures(&track, features)

	tests := []struct {
		name     string
		got      *float32
		expected float32
	}{
		{"Acousticness", track.Acousticness, 0.5},
		{"Danceability", track.Danceability, 0.7},
		{"Energy", track.Energy, 0.8},
		{"Instrumentalness", track.Instrumentalness, 0.1},
		{"Liveness", track.Liveness, 0.2},
		{"Loudness", track.Loudness, -5.0},
		{"Speechiness", track.Speechiness, 0.05},
		{"Tempo", track.Tempo, 120.0},
		{"Valence", track.Valence, 0.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == nil {
				t.Errorf("%s is nil, want %v", tt.name, tt.expected)
				return
			}
			if *tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, *tt.got, tt.expected)
			}
		})
	}
}

func TestApplyAudioFeaturesZeroValues(t *testing.T) {
	track := clustering.Track{ID: "test456", Name: "Silent Track"}
	features := &spotify.AudioFeatures{
		Acousticness:     0.0,
		Danceability:     0.0,
		Energy:           0.0,
		Instrumentalness: 0.0,
		Liveness:         0.0,
		Loudness:         0.0,
		Speechiness:      0.0,
		Tempo:            0.0,
		Valence:          0.0,
	}

	applyAudioFeatures(&track, features)

	// Verify zero values are properly set (not nil)
	if track.Energy == nil {
		t.Error("Energy should not be nil for zero value")
	}
	if track.Energy != nil && *track.Energy != 0.0 {
		t.Errorf("Energy = %v, want 0.0", *track.Energy)
	}

	if track.Valence == nil {
		t.Error("Valence should not be nil for zero value")
	}
	if track.Valence != nil && *track.Valence != 0.0 {
		t.Errorf("Valence = %v, want 0.0", *track.Valence)
	}
}

func TestTrackWithoutAudioFeatures(t *testing.T) {
	// A track that hasn't had audio features applied should have nil fields
	track := clustering.Track{ID: "no-features", Name: "Unknown Track"}

	if track.Energy != nil {
		t.Error("Energy should be nil for track without audio features")
	}
	if track.Valence != nil {
		t.Error("Valence should be nil for track without audio features")
	}
	if track.Danceability != nil {
		t.Error("Danceability should be nil for track without audio features")
	}
}

func TestAudioFeaturesBatchCount(t *testing.T) {
	tests := []struct {
		name          string
		totalTracks   int
		expectedCalls int
	}{
		{"empty", 0, 0},
		{"single track", 1, 1},
		{"less than 100", 50, 1},
		{"exactly 100", 100, 1},
		{"101 tracks", 101, 2},
		{"exactly 200", 200, 2},
		{"250 tracks", 250, 3},
		{"1000 tracks", 1000, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			for i := 0; i < tt.totalTracks; i += maxTracksPerRequest {
				calls++
			}

			// Handle edge case where 0 tracks means 0 calls
			if tt.totalTracks == 0 {
				calls = 0
			}

			if calls != tt.expectedCalls {
				t.Errorf("got %d API calls, want %d", calls, tt.expectedCalls)
			}
		})
	}
}
