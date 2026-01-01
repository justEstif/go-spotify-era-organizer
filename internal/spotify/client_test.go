package spotify

import (
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"
)

func TestConvertTrack(t *testing.T) {
	tests := []struct {
		name           string
		saved          spotify.SavedTrack
		expectedID     string
		expectedName   string
		expectedArtist string
		expectedTime   time.Time
	}{
		{
			name: "single artist",
			saved: spotify.SavedTrack{
				AddedAt: "2024-01-15T10:30:00Z",
				FullTrack: spotify.FullTrack{
					SimpleTrack: spotify.SimpleTrack{
						ID:   "track123",
						Name: "Test Song",
						Artists: []spotify.SimpleArtist{
							{Name: "Artist One"},
						},
					},
				},
			},
			expectedID:     "track123",
			expectedName:   "Test Song",
			expectedArtist: "Artist One",
			expectedTime:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name: "multiple artists",
			saved: spotify.SavedTrack{
				AddedAt: "2023-06-20T15:45:00Z",
				FullTrack: spotify.FullTrack{
					SimpleTrack: spotify.SimpleTrack{
						ID:   "track456",
						Name: "Collab Track",
						Artists: []spotify.SimpleArtist{
							{Name: "Artist A"},
							{Name: "Artist B"},
							{Name: "Artist C"},
						},
					},
				},
			},
			expectedID:     "track456",
			expectedName:   "Collab Track",
			expectedArtist: "Artist A, Artist B, Artist C",
			expectedTime:   time.Date(2023, 6, 20, 15, 45, 0, 0, time.UTC),
		},
		{
			name: "invalid timestamp uses zero value",
			saved: spotify.SavedTrack{
				AddedAt: "not-a-valid-timestamp",
				FullTrack: spotify.FullTrack{
					SimpleTrack: spotify.SimpleTrack{
						ID:   "track789",
						Name: "Old Song",
						Artists: []spotify.SimpleArtist{
							{Name: "Mystery Artist"},
						},
					},
				},
			},
			expectedID:     "track789",
			expectedName:   "Old Song",
			expectedArtist: "Mystery Artist",
			expectedTime:   time.Time{}, // zero value
		},
		{
			name: "no artists",
			saved: spotify.SavedTrack{
				AddedAt: "2024-03-01T00:00:00Z",
				FullTrack: spotify.FullTrack{
					SimpleTrack: spotify.SimpleTrack{
						ID:      "track000",
						Name:    "Unknown Track",
						Artists: []spotify.SimpleArtist{},
					},
				},
			},
			expectedID:     "track000",
			expectedName:   "Unknown Track",
			expectedArtist: "",
			expectedTime:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTrack(tt.saved)

			if got.ID != tt.expectedID {
				t.Errorf("ID = %q, want %q", got.ID, tt.expectedID)
			}
			if got.Name != tt.expectedName {
				t.Errorf("Name = %q, want %q", got.Name, tt.expectedName)
			}
			if got.Artist != tt.expectedArtist {
				t.Errorf("Artist = %q, want %q", got.Artist, tt.expectedArtist)
			}
			if !got.AddedAt.Equal(tt.expectedTime) {
				t.Errorf("AddedAt = %v, want %v", got.AddedAt, tt.expectedTime)
			}
		})
	}
}

func TestBatchChunking(t *testing.T) {
	tests := []struct {
		name          string
		totalTracks   int
		expectedBatch []struct{ start, end int }
	}{
		{
			name:        "less than 100",
			totalTracks: 50,
			expectedBatch: []struct{ start, end int }{
				{0, 50},
			},
		},
		{
			name:        "exactly 100",
			totalTracks: 100,
			expectedBatch: []struct{ start, end int }{
				{0, 100},
			},
		},
		{
			name:        "more than 100",
			totalTracks: 250,
			expectedBatch: []struct{ start, end int }{
				{0, 100},
				{100, 200},
				{200, 250},
			},
		},
		{
			name:        "exactly 200",
			totalTracks: 200,
			expectedBatch: []struct{ start, end int }{
				{0, 100},
				{100, 200},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var batches []struct{ start, end int }

			for i := 0; i < tt.totalTracks; i += maxTracksPerRequest {
				end := min(i+maxTracksPerRequest, tt.totalTracks)
				batches = append(batches, struct{ start, end int }{i, end})
			}

			if len(batches) != len(tt.expectedBatch) {
				t.Errorf("got %d batches, want %d", len(batches), len(tt.expectedBatch))
				return
			}

			for i, batch := range batches {
				if batch.start != tt.expectedBatch[i].start || batch.end != tt.expectedBatch[i].end {
					t.Errorf("batch %d = {%d, %d}, want {%d, %d}",
						i, batch.start, batch.end,
						tt.expectedBatch[i].start, tt.expectedBatch[i].end)
				}
			}
		})
	}
}
