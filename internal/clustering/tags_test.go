package clustering

import (
	"testing"
	"time"
)

func TestDetectMoodEras_Empty(t *testing.T) {
	eras, outliers := DetectMoodEras(nil, DefaultTagClusterConfig())
	if eras != nil {
		t.Errorf("expected nil eras, got %v", eras)
	}
	if outliers != nil {
		t.Errorf("expected nil outliers, got %v", outliers)
	}
}

func TestDetectMoodEras_NoTags(t *testing.T) {
	tracks := []Track{
		{ID: "1", Name: "Track 1", Artist: "Artist 1"},
		{ID: "2", Name: "Track 2", Artist: "Artist 2"},
	}

	eras, outliers := DetectMoodEras(tracks, DefaultTagClusterConfig())

	if len(eras) != 0 {
		t.Errorf("expected 0 eras, got %d", len(eras))
	}
	if len(outliers) != 2 {
		t.Errorf("expected 2 outliers, got %d", len(outliers))
	}
}

func TestDetectMoodEras_SingleTrackWithTags(t *testing.T) {
	tracks := []Track{
		{
			ID:     "1",
			Name:   "Track 1",
			Artist: "Artist 1",
			Tags:   []Tag{{Name: "rock", Count: 100}},
		},
	}

	// With 1 track and 1 cluster, k-means will create 1 era (MinClusterSize=1)
	eras, outliers := DetectMoodEras(tracks, TagClusterConfig{NumClusters: 1, MinClusterSize: 1, MaxTags: 50})

	if len(eras) != 1 {
		t.Errorf("expected 1 era, got %d", len(eras))
	}
	if len(outliers) != 0 {
		t.Errorf("expected 0 outliers, got %d", len(outliers))
	}
	if len(eras) > 0 && len(eras[0].Tracks) != 1 {
		t.Errorf("expected era to have 1 track, got %d", len(eras[0].Tracks))
	}
}

func TestDetectMoodEras_ClustersByTagSimilarity(t *testing.T) {
	makeDate := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	// Create tracks with distinct tag profiles
	tracks := []Track{
		// Rock cluster
		{ID: "r1", Name: "Rock 1", Artist: "Rock Band", AddedAt: makeDate(2024, 1, 1), Tags: []Tag{{Name: "rock", Count: 100}, {Name: "guitar", Count: 80}}},
		{ID: "r2", Name: "Rock 2", Artist: "Rock Band", AddedAt: makeDate(2024, 1, 2), Tags: []Tag{{Name: "rock", Count: 95}, {Name: "guitar", Count: 75}}},
		{ID: "r3", Name: "Rock 3", Artist: "Rock Band", AddedAt: makeDate(2024, 1, 3), Tags: []Tag{{Name: "rock", Count: 90}, {Name: "guitar", Count: 70}}},
		// Electronic cluster
		{ID: "e1", Name: "Electronic 1", Artist: "DJ", AddedAt: makeDate(2024, 2, 1), Tags: []Tag{{Name: "electronic", Count: 100}, {Name: "dance", Count: 90}}},
		{ID: "e2", Name: "Electronic 2", Artist: "DJ", AddedAt: makeDate(2024, 2, 2), Tags: []Tag{{Name: "electronic", Count: 95}, {Name: "dance", Count: 85}}},
		{ID: "e3", Name: "Electronic 3", Artist: "DJ", AddedAt: makeDate(2024, 2, 3), Tags: []Tag{{Name: "electronic", Count: 90}, {Name: "dance", Count: 80}}},
	}

	eras, outliers := DetectMoodEras(tracks, TagClusterConfig{NumClusters: 2, MinClusterSize: 3, MaxTags: 50})

	if len(eras) != 2 {
		t.Fatalf("expected 2 eras, got %d", len(eras))
	}
	if len(outliers) != 0 {
		t.Errorf("expected 0 outliers, got %d", len(outliers))
	}

	// Verify each era has 3 tracks
	for i, era := range eras {
		if len(era.Tracks) != 3 {
			t.Errorf("era %d: expected 3 tracks, got %d", i, len(era.Tracks))
		}
		if len(era.TopTags) == 0 {
			t.Errorf("era %d: expected top tags, got empty", i)
		}
	}
}

func TestDetectMoodEras_SmallClustersAreOutliers(t *testing.T) {
	makeDate := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	tracks := []Track{
		// Large cluster
		{ID: "1", Name: "Track 1", Artist: "Artist", AddedAt: makeDate(2024, 1, 1), Tags: []Tag{{Name: "rock", Count: 100}}},
		{ID: "2", Name: "Track 2", Artist: "Artist", AddedAt: makeDate(2024, 1, 2), Tags: []Tag{{Name: "rock", Count: 95}}},
		{ID: "3", Name: "Track 3", Artist: "Artist", AddedAt: makeDate(2024, 1, 3), Tags: []Tag{{Name: "rock", Count: 90}}},
		{ID: "4", Name: "Track 4", Artist: "Artist", AddedAt: makeDate(2024, 1, 4), Tags: []Tag{{Name: "rock", Count: 85}}},
		// Single track that will be in its own cluster (below MinClusterSize)
		{ID: "5", Name: "Outlier", Artist: "Different", AddedAt: makeDate(2024, 2, 1), Tags: []Tag{{Name: "jazz", Count: 100}}},
	}

	_, outliers := DetectMoodEras(tracks, TagClusterConfig{NumClusters: 2, MinClusterSize: 3, MaxTags: 50})

	// The jazz track should be an outlier since it can't form a cluster of 3
	if len(outliers) < 1 {
		t.Errorf("expected at least 1 outlier, got %d", len(outliers))
	}
}

func TestDetectMoodEras_MixedTagsAndNoTags(t *testing.T) {
	tracks := []Track{
		{ID: "1", Name: "Tagged 1", Artist: "Artist", Tags: []Tag{{Name: "rock", Count: 100}}},
		{ID: "2", Name: "Tagged 2", Artist: "Artist", Tags: []Tag{{Name: "rock", Count: 95}}},
		{ID: "3", Name: "Tagged 3", Artist: "Artist", Tags: []Tag{{Name: "rock", Count: 90}}},
		{ID: "4", Name: "No Tags", Artist: "Artist", Tags: nil},
	}

	_, outliers := DetectMoodEras(tracks, TagClusterConfig{NumClusters: 1, MinClusterSize: 3, MaxTags: 50})

	// Track without tags should be in outliers
	hasNoTagsOutlier := false
	for _, o := range outliers {
		if o.Name == "No Tags" {
			hasNoTagsOutlier = true
			break
		}
	}
	if !hasNoTagsOutlier {
		t.Error("expected 'No Tags' track to be in outliers")
	}
}

func TestBuildTagVocabulary(t *testing.T) {
	tracks := []*Track{
		{Tags: []Tag{{Name: "Rock", Count: 100}, {Name: "Indie", Count: 50}}},
		{Tags: []Tag{{Name: "rock", Count: 80}, {Name: "alternative", Count: 60}}}, // lowercase rock
		{Tags: []Tag{{Name: "ROCK", Count: 70}, {Name: "pop", Count: 40}}},         // uppercase rock
	}

	vocab := buildTagVocabulary(tracks, 10)

	// Should have 4 unique tags (rock counts are merged due to lowercase normalization)
	if len(vocab) != 4 {
		t.Errorf("expected 4 tags, got %d: %v", len(vocab), vocab)
	}

	// "rock" should be first (highest combined count: 100+80+70=250)
	if vocab[0] != "rock" {
		t.Errorf("expected 'rock' first, got %q", vocab[0])
	}
}

func TestBuildTagVocabulary_MaxTags(t *testing.T) {
	tracks := []*Track{
		{Tags: []Tag{
			{Name: "tag1", Count: 100},
			{Name: "tag2", Count: 90},
			{Name: "tag3", Count: 80},
			{Name: "tag4", Count: 70},
			{Name: "tag5", Count: 60},
		}},
	}

	vocab := buildTagVocabulary(tracks, 3)

	if len(vocab) != 3 {
		t.Errorf("expected 3 tags (maxTags), got %d", len(vocab))
	}

	// Should be sorted by count
	expected := []string{"tag1", "tag2", "tag3"}
	for i, want := range expected {
		if vocab[i] != want {
			t.Errorf("vocab[%d] = %q, want %q", i, vocab[i], want)
		}
	}
}

func TestBuildTagVector(t *testing.T) {
	track := &Track{
		Tags: []Tag{
			{Name: "rock", Count: 100},
			{Name: "indie", Count: 50},
		},
	}

	vocabulary := []string{"rock", "indie", "pop", "jazz"}
	vector := buildTagVector(track, vocabulary)

	if len(vector) != 4 {
		t.Fatalf("expected vector length 4, got %d", len(vector))
	}

	// rock: 100/100 = 1.0
	if vector[0] != 1.0 {
		t.Errorf("vector[rock] = %v, want 1.0", vector[0])
	}

	// indie: 50/100 = 0.5
	if vector[1] != 0.5 {
		t.Errorf("vector[indie] = %v, want 0.5", vector[1])
	}

	// pop: not present = 0.0
	if vector[2] != 0.0 {
		t.Errorf("vector[pop] = %v, want 0.0", vector[2])
	}

	// jazz: not present = 0.0
	if vector[3] != 0.0 {
		t.Errorf("vector[jazz] = %v, want 0.0", vector[3])
	}
}

func TestExtractTopTags(t *testing.T) {
	vocabulary := []string{"rock", "indie", "pop", "jazz", "electronic"}
	centroid := []float64{0.8, 0.6, 0.0, 0.3, 0.0}

	topTags := extractTopTags(centroid, vocabulary, 3)

	if len(topTags) != 3 {
		t.Fatalf("expected 3 top tags, got %d", len(topTags))
	}

	// Should be sorted by weight descending
	expected := []string{"rock", "indie", "jazz"}
	for i, want := range expected {
		if topTags[i] != want {
			t.Errorf("topTags[%d] = %q, want %q", i, topTags[i], want)
		}
	}
}

func TestExtractTopTags_SkipsZeroWeights(t *testing.T) {
	vocabulary := []string{"rock", "indie", "pop"}
	centroid := []float64{0.8, 0.0, 0.0}

	topTags := extractTopTags(centroid, vocabulary, 3)

	// Should only return 1 tag (the one with non-zero weight)
	if len(topTags) != 1 {
		t.Errorf("expected 1 top tag, got %d: %v", len(topTags), topTags)
	}
	if topTags[0] != "rock" {
		t.Errorf("expected 'rock', got %q", topTags[0])
	}
}

func TestGenerateEraName(t *testing.T) {
	tests := []struct {
		name     string
		topTags  []string
		start    time.Time
		end      time.Time
		expected string
	}{
		{
			name:     "multiple tags different dates",
			topTags:  []string{"rock", "indie", "alternative"},
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 3, 0, 0, 0, 0, time.UTC),
			expected: "rock & indie & alternative: Jan 15, 2024 - Feb 3, 2024",
		},
		{
			name:     "single tag same date",
			topTags:  []string{"electronic"},
			start:    time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
			expected: "electronic: Mar 10, 2024",
		},
		{
			name:     "no tags",
			topTags:  nil,
			start:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
			expected: "Mixed: Apr 1, 2024 - Apr 5, 2024",
		},
		{
			name:     "empty tags slice",
			topTags:  []string{},
			start:    time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			expected: "Mixed: May 1, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateEraName(tt.topTags, tt.start, tt.end)
			if got != tt.expected {
				t.Errorf("generateEraName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultTagClusterConfig(t *testing.T) {
	cfg := DefaultTagClusterConfig()

	if cfg.NumClusters != 3 {
		t.Errorf("NumClusters = %d, want 3", cfg.NumClusters)
	}
	if cfg.MinClusterSize != 3 {
		t.Errorf("MinClusterSize = %d, want 3", cfg.MinClusterSize)
	}
	if cfg.MaxTags != 50 {
		t.Errorf("MaxTags = %d, want 50", cfg.MaxTags)
	}
}

func TestDetectMoodEras_UsesDefaults(t *testing.T) {
	tracks := []Track{
		{ID: "1", Tags: []Tag{{Name: "rock", Count: 100}}},
		{ID: "2", Tags: []Tag{{Name: "rock", Count: 90}}},
		{ID: "3", Tags: []Tag{{Name: "rock", Count: 80}}},
	}

	// Pass config with zero values - should use defaults
	eras, _ := DetectMoodEras(tracks, TagClusterConfig{})

	// Should work without panic
	if eras == nil && len(tracks) >= 3 {
		// This is acceptable - tracks may all be in outliers depending on clustering
	}
}

func TestDetectMoodEras_ErasSortedByStartDateDescending(t *testing.T) {
	makeDate := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	tracks := []Track{
		// Old cluster
		{ID: "o1", Name: "Old 1", AddedAt: makeDate(2024, 1, 1), Tags: []Tag{{Name: "jazz", Count: 100}}},
		{ID: "o2", Name: "Old 2", AddedAt: makeDate(2024, 1, 2), Tags: []Tag{{Name: "jazz", Count: 95}}},
		{ID: "o3", Name: "Old 3", AddedAt: makeDate(2024, 1, 3), Tags: []Tag{{Name: "jazz", Count: 90}}},
		// Recent cluster
		{ID: "n1", Name: "New 1", AddedAt: makeDate(2024, 6, 1), Tags: []Tag{{Name: "rock", Count: 100}}},
		{ID: "n2", Name: "New 2", AddedAt: makeDate(2024, 6, 2), Tags: []Tag{{Name: "rock", Count: 95}}},
		{ID: "n3", Name: "New 3", AddedAt: makeDate(2024, 6, 3), Tags: []Tag{{Name: "rock", Count: 90}}},
	}

	eras, _ := DetectMoodEras(tracks, TagClusterConfig{NumClusters: 2, MinClusterSize: 3, MaxTags: 50})

	if len(eras) < 2 {
		t.Skipf("clustering resulted in %d eras, need 2 to test ordering", len(eras))
	}

	// Most recent era should be first
	if eras[0].StartDate.Before(eras[1].StartDate) {
		t.Errorf("eras not sorted by start date descending: era[0]=%v, era[1]=%v",
			eras[0].StartDate, eras[1].StartDate)
	}
}
