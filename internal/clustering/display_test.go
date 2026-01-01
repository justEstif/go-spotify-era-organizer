package clustering

import (
	"strings"
	"testing"
	"time"
)

func TestFormatEraSummary(t *testing.T) {
	// Helper to create tracks
	makeTrack := func(name, artist string, daysAgo int) Track {
		return Track{
			ID:      name,
			Name:    name,
			Artist:  artist,
			AddedAt: time.Now().AddDate(0, 0, -daysAgo),
		}
	}

	// Helper to create an era
	makeEra := func(tracks []Track) Era {
		if len(tracks) == 0 {
			return Era{}
		}
		return Era{
			Tracks:    tracks,
			StartDate: tracks[0].AddedAt,
			EndDate:   tracks[len(tracks)-1].AddedAt,
		}
	}

	tests := []struct {
		name           string
		eras           []Era
		outliers       []Track
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:     "empty eras no outliers",
			eras:     nil,
			outliers: nil,
			wantContains: []string{
				"No eras found from 0 tracks",
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
				"No eras found from 1 tracks",
				"(1 outliers skipped)",
			},
		},
		{
			name: "single era with 3 tracks",
			eras: []Era{
				makeEra([]Track{
					makeTrack("Song1", "Artist1", 10),
					makeTrack("Song2", "Artist2", 9),
					makeTrack("Song3", "Artist3", 8),
				}),
			},
			outliers: nil,
			wantContains: []string{
				"Found 1 era from 3 tracks",
				"Era 1:",
				"(3 tracks)",
				`"Song1" - Artist1`,
				`"Song2" - Artist2`,
				`"Song3" - Artist3`,
			},
			wantNotContain: []string{
				"and", // No "and N more" for exactly 3 tracks
				"outliers",
			},
		},
		{
			name: "single era with 5 tracks shows and N more",
			eras: []Era{
				makeEra([]Track{
					makeTrack("Song1", "Artist1", 10),
					makeTrack("Song2", "Artist2", 9),
					makeTrack("Song3", "Artist3", 8),
					makeTrack("Song4", "Artist4", 7),
					makeTrack("Song5", "Artist5", 6),
				}),
			},
			outliers: nil,
			wantContains: []string{
				"Found 1 era from 5 tracks",
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
			eras: []Era{
				makeEra([]Track{
					makeTrack("Era1Song1", "Artist1", 30),
					makeTrack("Era1Song2", "Artist2", 29),
					makeTrack("Era1Song3", "Artist3", 28),
					makeTrack("Era1Song4", "Artist4", 27),
				}),
				makeEra([]Track{
					makeTrack("Era2Song1", "ArtistA", 10),
					makeTrack("Era2Song2", "ArtistB", 9),
				}),
			},
			outliers: []Track{
				makeTrack("Outlier1", "OutlierArtist", 50),
				makeTrack("Outlier2", "OutlierArtist", 51),
				makeTrack("Outlier3", "OutlierArtist", 52),
			},
			wantContains: []string{
				"Found 2 eras from 9 tracks",
				"(3 outliers skipped)",
				"Era 1:",
				"(4 tracks)",
				"... and 1 more",
				"Era 2:",
				"(2 tracks)",
				`"Era2Song1" - ArtistA`,
			},
			wantNotContain: []string{
				"Outlier", // Outliers should not be listed
			},
		},
		{
			name: "single track era uses singular",
			eras: []Era{
				makeEra([]Track{
					makeTrack("OnlySong", "OnlyArtist", 5),
				}),
			},
			outliers: nil,
			wantContains: []string{
				"(1 track)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatEraSummary(tt.eras, tt.outliers)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatEraSummary() missing expected content %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("FormatEraSummary() contains unexpected content %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}
