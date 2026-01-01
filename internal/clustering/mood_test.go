package clustering

import (
	"testing"
	"time"
)

func TestGenerateMoodName(t *testing.T) {
	tests := []struct {
		name     string
		centroid map[string]float32
		want     string
	}{
		{
			name: "high energy high valence",
			centroid: map[string]float32{
				"energy":       0.8,
				"valence":      0.7,
				"danceability": 0.6,
				"acousticness": 0.2,
			},
			want: "Upbeat Party",
		},
		{
			name: "high energy low valence",
			centroid: map[string]float32{
				"energy":       0.8,
				"valence":      0.3,
				"danceability": 0.6,
				"acousticness": 0.2,
			},
			want: "Intense & Dark",
		},
		{
			name: "low energy high valence",
			centroid: map[string]float32{
				"energy":       0.4,
				"valence":      0.7,
				"danceability": 0.5,
				"acousticness": 0.3,
			},
			want: "Chill & Happy",
		},
		{
			name: "low energy low valence",
			centroid: map[string]float32{
				"energy":       0.3,
				"valence":      0.3,
				"danceability": 0.4,
				"acousticness": 0.4,
			},
			want: "Reflective & Melancholy",
		},
		{
			name: "high acousticness adds modifier",
			centroid: map[string]float32{
				"energy":       0.4,
				"valence":      0.7,
				"danceability": 0.5,
				"acousticness": 0.8,
			},
			want: "Chill & Happy (Acoustic)",
		},
		{
			name: "boundary energy exactly 0.6 is low",
			centroid: map[string]float32{
				"energy":       0.6,
				"valence":      0.7,
				"danceability": 0.5,
				"acousticness": 0.2,
			},
			want: "Chill & Happy",
		},
		{
			name: "boundary valence exactly 0.5 is low",
			centroid: map[string]float32{
				"energy":       0.8,
				"valence":      0.5,
				"danceability": 0.6,
				"acousticness": 0.2,
			},
			want: "Intense & Dark",
		},
		{
			name: "boundary acousticness exactly 0.6 no modifier",
			centroid: map[string]float32{
				"energy":       0.8,
				"valence":      0.7,
				"danceability": 0.6,
				"acousticness": 0.6,
			},
			want: "Upbeat Party",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateMoodName(tt.centroid)
			if got != tt.want {
				t.Errorf("generateMoodName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetMoodCategory(t *testing.T) {
	centroid := map[string]float32{
		"energy":       0.8,
		"valence":      0.7,
		"danceability": 0.6,
		"acousticness": 0.2,
	}

	category := GetMoodCategory(centroid)

	if category.Name != "Upbeat Party" {
		t.Errorf("Name = %q, want %q", category.Name, "Upbeat Party")
	}

	if category.Energy != 0.8 {
		t.Errorf("Energy = %v, want 0.8", category.Energy)
	}

	if category.Valence != 0.7 {
		t.Errorf("Valence = %v, want 0.7", category.Valence)
	}

	if category.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestFormatEraName(t *testing.T) {
	tests := []struct {
		name      string
		moodName  string
		startYear int
		startMon  int
		startDay  int
		endYear   int
		endMon    int
		endDay    int
		want      string
	}{
		{
			name:      "different dates",
			moodName:  "Upbeat Party",
			startYear: 2024, startMon: 1, startDay: 15,
			endYear: 2024, endMon: 2, endDay: 3,
			want: "Upbeat Party: Jan 15, 2024 - Feb 3, 2024",
		},
		{
			name:      "same date",
			moodName:  "Chill & Happy",
			startYear: 2024, startMon: 3, startDay: 10,
			endYear: 2024, endMon: 3, endDay: 10,
			want: "Chill & Happy: Mar 10, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := makeDate(tt.startYear, tt.startMon, tt.startDay)
			end := makeDate(tt.endYear, tt.endMon, tt.endDay)

			got := formatEraName(tt.moodName, start, end)
			if got != tt.want {
				t.Errorf("formatEraName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasAudioFeatures(t *testing.T) {
	energy := float32(0.5)
	valence := float32(0.5)
	danceability := float32(0.5)
	acousticness := float32(0.5)

	tests := []struct {
		name  string
		track Track
		want  bool
	}{
		{
			name: "all features present",
			track: Track{
				Energy:       &energy,
				Valence:      &valence,
				Danceability: &danceability,
				Acousticness: &acousticness,
			},
			want: true,
		},
		{
			name:  "no features",
			track: Track{},
			want:  false,
		},
		{
			name: "missing energy",
			track: Track{
				Valence:      &valence,
				Danceability: &danceability,
				Acousticness: &acousticness,
			},
			want: false,
		},
		{
			name: "missing valence",
			track: Track{
				Energy:       &energy,
				Danceability: &danceability,
				Acousticness: &acousticness,
			},
			want: false,
		},
		{
			name: "missing danceability",
			track: Track{
				Energy:       &energy,
				Valence:      &valence,
				Acousticness: &acousticness,
			},
			want: false,
		},
		{
			name: "missing acousticness",
			track: Track{
				Energy:       &energy,
				Valence:      &valence,
				Danceability: &danceability,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAudioFeatures(&tt.track)
			if got != tt.want {
				t.Errorf("hasAudioFeatures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFeatures(t *testing.T) {
	energy := float32(0.8)
	valence := float32(0.7)
	danceability := float32(0.6)
	acousticness := float32(0.2)

	track := Track{
		Energy:       &energy,
		Valence:      &valence,
		Danceability: &danceability,
		Acousticness: &acousticness,
	}

	coords := extractFeatures(&track)

	if len(coords) != 4 {
		t.Fatalf("expected 4 coordinates, got %d", len(coords))
	}

	// Order should match featureNames: energy, valence, danceability, acousticness
	// Use approximate comparison due to float32->float64 conversion
	expected := []float32{0.8, 0.7, 0.6, 0.2}
	for i, want := range expected {
		got := float32(coords[i])
		if got != want {
			t.Errorf("coords[%d] = %v, want %v", i, got, want)
		}
	}
}

// makeDate creates a time.Time for testing
func makeDate(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
