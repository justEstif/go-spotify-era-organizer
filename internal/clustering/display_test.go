package clustering

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMoodEraSummary(t *testing.T) {
	// Helper to create dates
	makeDate := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	// Helper to create a MoodEra
	makeMoodEra := func(name string, tracks []Track, topTags []string) MoodEra {
		if len(tracks) == 0 {
			return MoodEra{Name: name, TopTags: topTags}
		}
		return MoodEra{
			Name:      name,
			Tracks:    tracks,
			TopTags:   topTags,
			StartDate: tracks[0].AddedAt,
			EndDate:   tracks[len(tracks)-1].AddedAt,
		}
	}

	tests := []struct {
		name           string
		eras           []MoodEra
		outliers       []Track
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:     "no eras no outliers",
			eras:     nil,
			outliers: nil,
			wantContains: []string{
				"No mood eras found from 0 tracks",
			},
		},
		{
			name: "no eras with outliers",
			eras: nil,
			outliers: []Track{
				{ID: "1", Name: "Track 1", Artist: "Artist 1"},
				{ID: "2", Name: "Track 2", Artist: "Artist 2"},
			},
			wantContains: []string{
				"No mood eras found from 2 tracks",
				"(2 outliers skipped)",
			},
		},
		{
			name: "single era with tags",
			eras: []MoodEra{
				makeMoodEra("rock & indie & alternative: Jan 1 - Jan 3, 2024", []Track{
					{ID: "1", Name: "Track 1", Artist: "Artist 1", AddedAt: makeDate(2024, 1, 1)},
					{ID: "2", Name: "Track 2", Artist: "Artist 2", AddedAt: makeDate(2024, 1, 2)},
					{ID: "3", Name: "Track 3", Artist: "Artist 3", AddedAt: makeDate(2024, 1, 3)},
				}, []string{"rock", "indie", "alternative"}),
			},
			outliers: nil,
			wantContains: []string{
				"Found 1 mood era from 3 tracks",
				"Era 1:",
				"rock & indie & alternative",
				"Tags: rock, indie, alternative",
				"Track 1",
				"Track 2",
				"Track 3",
			},
			wantNotContain: []string{
				"outliers",
			},
		},
		{
			name: "multiple eras with outliers",
			eras: []MoodEra{
				makeMoodEra("electronic & dance: Feb 1 - Feb 5, 2024", []Track{
					{ID: "1", Name: "Dance Track 1", Artist: "DJ 1", AddedAt: makeDate(2024, 2, 1)},
					{ID: "2", Name: "Dance Track 2", Artist: "DJ 2", AddedAt: makeDate(2024, 2, 5)},
				}, []string{"electronic", "dance"}),
				makeMoodEra("chill & ambient: Mar 1 - Mar 4, 2024", []Track{
					{ID: "3", Name: "Chill Track 1", Artist: "Artist 1", AddedAt: makeDate(2024, 3, 1)},
					{ID: "4", Name: "Chill Track 2", Artist: "Artist 2", AddedAt: makeDate(2024, 3, 4)},
				}, []string{"chill", "ambient"}),
			},
			outliers: []Track{
				{ID: "5", Name: "Random Track", Artist: "Unknown"},
			},
			wantContains: []string{
				"Found 2 mood eras from 5 tracks",
				"(1 outliers skipped)",
				"Era 1:",
				"Era 2:",
				"Tags: electronic, dance",
				"Tags: chill, ambient",
			},
		},
		{
			name: "era with more than 3 tracks shows count",
			eras: []MoodEra{
				makeMoodEra("pop: May 1 - May 5, 2024", []Track{
					{ID: "1", Name: "Pop 1", Artist: "Singer 1", AddedAt: makeDate(2024, 5, 1)},
					{ID: "2", Name: "Pop 2", Artist: "Singer 2", AddedAt: makeDate(2024, 5, 2)},
					{ID: "3", Name: "Pop 3", Artist: "Singer 3", AddedAt: makeDate(2024, 5, 3)},
					{ID: "4", Name: "Pop 4", Artist: "Singer 4", AddedAt: makeDate(2024, 5, 4)},
					{ID: "5", Name: "Pop 5", Artist: "Singer 5", AddedAt: makeDate(2024, 5, 5)},
				}, []string{"pop"}),
			},
			outliers: nil,
			wantContains: []string{
				"5 tracks",
				"Pop 1",
				"Pop 2",
				"Pop 3",
				"... and 2 more",
			},
			wantNotContain: []string{
				"Pop 4",
				"Pop 5",
			},
		},
		{
			name: "era without tags",
			eras: []MoodEra{
				makeMoodEra("Mixed: Jun 1, 2024", []Track{
					{ID: "1", Name: "Track 1", Artist: "Artist 1", AddedAt: makeDate(2024, 6, 1)},
				}, nil),
			},
			outliers: nil,
			wantContains: []string{
				"Era 1:",
				"1 track",
			},
			wantNotContain: []string{
				"Tags:",
			},
		},
		{
			name: "single track era uses singular",
			eras: []MoodEra{
				makeMoodEra("rock: Jul 1, 2024", []Track{
					{ID: "1", Name: "OnlySong", Artist: "OnlyArtist", AddedAt: makeDate(2024, 7, 1)},
				}, []string{"rock"}),
			},
			outliers: nil,
			wantContains: []string{
				"(1 track)",
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

func TestFormatMoodEra(t *testing.T) {
	makeDate := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	era := MoodEra{
		Name: "rock & metal: Jan 1 - Jan 5, 2024",
		Tracks: []Track{
			{ID: "1", Name: "Heavy Song", Artist: "Metal Band", AddedAt: makeDate(2024, 1, 1)},
			{ID: "2", Name: "Rock Anthem", Artist: "Rock Band", AddedAt: makeDate(2024, 1, 5)},
		},
		TopTags:   []string{"rock", "metal", "heavy"},
		StartDate: makeDate(2024, 1, 1),
		EndDate:   makeDate(2024, 1, 5),
	}

	got := formatMoodEra(1, era)

	wantContains := []string{
		"Era 1:",
		"rock & metal",
		"2 tracks",
		"Tags: rock, metal, heavy",
		"Heavy Song",
		"Rock Band",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("formatMoodEra() missing expected content %q\nGot:\n%s", want, got)
		}
	}
}
