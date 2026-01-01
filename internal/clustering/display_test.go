package clustering

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMoodEraSummary(t *testing.T) {
	// Helper to create tracks
	makeTrack := func(name, artist string, daysAgo int) Track {
		return Track{
			ID:      name,
			Name:    name,
			Artist:  artist,
			AddedAt: time.Now().AddDate(0, 0, -daysAgo),
		}
	}

	// Helper to create a mood era
	makeMoodEra := func(name string, tracks []Track, centroid map[string]float32) MoodEra {
		if len(tracks) == 0 {
			return MoodEra{Name: name, Centroid: centroid}
		}
		return MoodEra{
			Name:      name,
			Tracks:    tracks,
			Centroid:  centroid,
			StartDate: tracks[0].AddedAt,
			EndDate:   tracks[len(tracks)-1].AddedAt,
		}
	}

	defaultCentroid := map[string]float32{
		"energy":       0.75,
		"valence":      0.65,
		"danceability": 0.70,
		"acousticness": 0.20,
	}

	tests := []struct {
		name           string
		eras           []MoodEra
		outliers       []Track
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:     "empty eras no outliers",
			eras:     nil,
			outliers: nil,
			wantContains: []string{
				"No mood eras found from 0 tracks",
			},
			wantNotContain: []string{
				"outliers",
			},
		},
		{
			name:     "empty eras with outliers",
			eras:     nil,
			outliers: []Track{makeTrack("Song1", "Artist1", 10)},
			wantContains: []string{
				"No mood eras found from 1 tracks",
				"(1 outliers skipped)",
			},
		},
		{
			name: "single era with 3 tracks",
			eras: []MoodEra{
				makeMoodEra("Upbeat Party: Jan 1 - Jan 3, 2024", []Track{
					makeTrack("Song1", "Artist1", 10),
					makeTrack("Song2", "Artist2", 9),
					makeTrack("Song3", "Artist3", 8),
				}, defaultCentroid),
			},
			outliers: nil,
			wantContains: []string{
				"Found 1 mood era from 3 tracks",
				"Era 1:",
				"Upbeat Party",
				"(3 tracks)",
				`"Song1" - Artist1`,
				`"Song2" - Artist2`,
				`"Song3" - Artist3`,
				"Mood: Energy=75%",
			},
			wantNotContain: []string{
				"and", // No "and N more" for exactly 3 tracks
				"outliers",
			},
		},
		{
			name: "single era with 5 tracks shows and N more",
			eras: []MoodEra{
				makeMoodEra("Chill & Happy: Feb 1 - Feb 5, 2024", []Track{
					makeTrack("Song1", "Artist1", 10),
					makeTrack("Song2", "Artist2", 9),
					makeTrack("Song3", "Artist3", 8),
					makeTrack("Song4", "Artist4", 7),
					makeTrack("Song5", "Artist5", 6),
				}, defaultCentroid),
			},
			outliers: nil,
			wantContains: []string{
				"Found 1 mood era from 5 tracks",
				`"Song1" - Artist1`,
				`"Song2" - Artist2`,
				`"Song3" - Artist3`,
				"... and 2 more",
			},
			wantNotContain: []string{
				`"Song4"`,
				`"Song5"`,
			},
		},
		{
			name: "multiple eras with outliers",
			eras: []MoodEra{
				makeMoodEra("Intense & Dark: Mar 1 - Mar 4, 2024", []Track{
					makeTrack("Era1Song1", "Artist1", 30),
					makeTrack("Era1Song2", "Artist2", 29),
					makeTrack("Era1Song3", "Artist3", 28),
					makeTrack("Era1Song4", "Artist4", 27),
				}, defaultCentroid),
				makeMoodEra("Reflective & Melancholy: Apr 1 - Apr 2, 2024", []Track{
					makeTrack("Era2Song1", "ArtistA", 10),
					makeTrack("Era2Song2", "ArtistB", 9),
				}, defaultCentroid),
			},
			outliers: []Track{
				makeTrack("Outlier1", "OutlierArtist", 50),
				makeTrack("Outlier2", "OutlierArtist", 51),
				makeTrack("Outlier3", "OutlierArtist", 52),
			},
			wantContains: []string{
				"Found 2 mood eras from 9 tracks",
				"(3 outliers skipped)",
				"Era 1:",
				"Intense & Dark",
				"(4 tracks)",
				"... and 1 more",
				"Era 2:",
				"Reflective & Melancholy",
				"(2 tracks)",
				`"Era2Song1" - ArtistA`,
			},
			wantNotContain: []string{
				"Outlier", // Outliers should not be listed
			},
		},
		{
			name: "single track era uses singular",
			eras: []MoodEra{
				makeMoodEra("Upbeat Party: May 1, 2024", []Track{
					makeTrack("OnlySong", "OnlyArtist", 5),
				}, defaultCentroid),
			},
			outliers: nil,
			wantContains: []string{
				"(1 track)",
			},
		},
		{
			name: "shows mood indicators",
			eras: []MoodEra{
				makeMoodEra("Test Era", []Track{
					makeTrack("Song1", "Artist1", 1),
					makeTrack("Song2", "Artist2", 2),
				}, map[string]float32{
					"energy":       0.85,
					"valence":      0.60,
					"danceability": 0.75,
					"acousticness": 0.10,
				}),
			},
			outliers: nil,
			wantContains: []string{
				"Mood: Energy=85% Valence=60% Danceability=75%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMoodEraSummary(tt.eras, tt.outliers)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatMoodEraSummary() missing expected content %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("FormatMoodEraSummary() contains unexpected content %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}
