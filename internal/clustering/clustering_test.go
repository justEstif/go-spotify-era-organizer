package clustering

import (
	"testing"
	"time"
)

func TestDetectEras(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tests := []struct {
		name         string
		tracks       []Track
		cfg          Config
		wantEras     int
		wantOutliers int
	}{
		{
			name:         "empty input",
			tracks:       []Track{},
			cfg:          DefaultConfig(),
			wantEras:     0,
			wantOutliers: 0,
		},
		{
			name: "single track becomes outlier",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
			},
			cfg:          DefaultConfig(),
			wantEras:     0,
			wantOutliers: 1,
		},
		{
			name: "all continuous - one era",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				{ID: "4", AddedAt: baseTime.Add(3 * day)},
				{ID: "5", AddedAt: baseTime.Add(4 * day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     1,
			wantOutliers: 0,
		},
		{
			name: "clear split - two eras",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				// 10-day gap here
				{ID: "4", AddedAt: baseTime.Add(12 * day)},
				{ID: "5", AddedAt: baseTime.Add(13 * day)},
				{ID: "6", AddedAt: baseTime.Add(14 * day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     2,
			wantOutliers: 0,
		},
		{
			name: "outlier in middle",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				// gap
				{ID: "4", AddedAt: baseTime.Add(12 * day)}, // isolated
				// gap
				{ID: "5", AddedAt: baseTime.Add(22 * day)},
				{ID: "6", AddedAt: baseTime.Add(23 * day)},
				{ID: "7", AddedAt: baseTime.Add(24 * day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     2,
			wantOutliers: 1,
		},
		{
			name: "exactly at threshold splits",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				{ID: "4", AddedAt: baseTime.Add(2*day + 7*day)}, // exactly 7 days from previous
				{ID: "5", AddedAt: baseTime.Add(2*day + 8*day)},
				{ID: "6", AddedAt: baseTime.Add(2*day + 9*day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     2,
			wantOutliers: 0,
		},
		{
			name: "just under threshold stays together",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				{ID: "4", AddedAt: baseTime.Add(2*day + 7*day - time.Second)}, // just under 7 days
				{ID: "5", AddedAt: baseTime.Add(2*day + 8*day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     1,
			wantOutliers: 0,
		},
		{
			name: "unsorted input gets sorted",
			tracks: []Track{
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
			},
			cfg:          DefaultConfig(),
			wantEras:     1,
			wantOutliers: 0,
		},
		{
			name: "custom config - larger gap threshold",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				{ID: "3", AddedAt: baseTime.Add(2 * day)},
				// 10-day gap - would split with default, but not with 14-day threshold
				{ID: "4", AddedAt: baseTime.Add(12 * day)},
				{ID: "5", AddedAt: baseTime.Add(13 * day)},
				{ID: "6", AddedAt: baseTime.Add(14 * day)},
			},
			cfg: Config{
				GapThreshold:   14 * 24 * time.Hour,
				MinClusterSize: 3,
			},
			wantEras:     1,
			wantOutliers: 0,
		},
		{
			name: "custom config - smaller min cluster size",
			tracks: []Track{
				{ID: "1", AddedAt: baseTime},
				{ID: "2", AddedAt: baseTime.Add(1 * day)},
				// gap
				{ID: "3", AddedAt: baseTime.Add(12 * day)},
				{ID: "4", AddedAt: baseTime.Add(13 * day)},
			},
			cfg: Config{
				GapThreshold:   7 * 24 * time.Hour,
				MinClusterSize: 2,
			},
			wantEras:     2,
			wantOutliers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eras, outliers := DetectEras(tt.tracks, tt.cfg)

			if len(eras) != tt.wantEras {
				t.Errorf("got %d eras, want %d", len(eras), tt.wantEras)
			}

			if len(outliers) != tt.wantOutliers {
				t.Errorf("got %d outliers, want %d", len(outliers), tt.wantOutliers)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GapThreshold != 7*24*time.Hour {
		t.Errorf("GapThreshold = %v, want 7 days", cfg.GapThreshold)
	}

	if cfg.MinClusterSize != 3 {
		t.Errorf("MinClusterSize = %d, want 3", cfg.MinClusterSize)
	}

	if cfg.MaxTracks != 30 {
		t.Errorf("MaxTracks = %d, want 30", cfg.MaxTracks)
	}

	if cfg.OutlierMode != OutlierModeSkip {
		t.Errorf("OutlierMode = %s, want %s", cfg.OutlierMode, OutlierModeSkip)
	}
}

func TestEraDateBoundaries(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tracks := []Track{
		{ID: "1", AddedAt: baseTime},
		{ID: "2", AddedAt: baseTime.Add(1 * day)},
		{ID: "3", AddedAt: baseTime.Add(2 * day)},
	}

	eras, _ := DetectEras(tracks, DefaultConfig())

	if len(eras) != 1 {
		t.Fatalf("expected 1 era, got %d", len(eras))
	}

	era := eras[0]
	if !era.StartDate.Equal(baseTime) {
		t.Errorf("StartDate = %v, want %v", era.StartDate, baseTime)
	}

	expectedEnd := baseTime.Add(2 * day)
	if !era.EndDate.Equal(expectedEnd) {
		t.Errorf("EndDate = %v, want %v", era.EndDate, expectedEnd)
	}
}

func TestEraTrackOrder(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	// Input in reverse order
	tracks := []Track{
		{ID: "3", Name: "Third", AddedAt: baseTime.Add(2 * day)},
		{ID: "1", Name: "First", AddedAt: baseTime},
		{ID: "2", Name: "Second", AddedAt: baseTime.Add(1 * day)},
	}

	eras, _ := DetectEras(tracks, DefaultConfig())

	if len(eras) != 1 {
		t.Fatalf("expected 1 era, got %d", len(eras))
	}

	// Verify tracks are sorted by AddedAt within era
	era := eras[0]
	if era.Tracks[0].ID != "1" {
		t.Errorf("first track ID = %s, want 1", era.Tracks[0].ID)
	}
	if era.Tracks[1].ID != "2" {
		t.Errorf("second track ID = %s, want 2", era.Tracks[1].ID)
	}
	if era.Tracks[2].ID != "3" {
		t.Errorf("third track ID = %s, want 3", era.Tracks[2].ID)
	}
}

func TestDetectErasDoesNotMutateInput(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	// Input in reverse order
	tracks := []Track{
		{ID: "3", AddedAt: baseTime.Add(2 * day)},
		{ID: "1", AddedAt: baseTime},
		{ID: "2", AddedAt: baseTime.Add(1 * day)},
	}

	// Save original order
	originalFirst := tracks[0].ID

	DetectEras(tracks, DefaultConfig())

	// Verify input slice wasn't modified
	if tracks[0].ID != originalFirst {
		t.Errorf("input slice was mutated: first element ID = %s, want %s", tracks[0].ID, originalFirst)
	}
}

func TestSplitLargeEras(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	hour := time.Hour

	// Helper to create N tracks with varying gaps
	createTracks := func(n int, gaps []time.Duration) []Track {
		tracks := make([]Track, n)
		current := baseTime
		for i := 0; i < n; i++ {
			tracks[i] = Track{
				ID:      string(rune('A' + i)),
				AddedAt: current,
			}
			if i < len(gaps) {
				current = current.Add(gaps[i])
			} else {
				current = current.Add(1 * hour) // default 1 hour gap
			}
		}
		return tracks
	}

	tests := []struct {
		name          string
		era           Era
		maxTracks     int
		wantSubEras   int
		wantSplitInfo bool // whether sub-eras should have SplitIndex/SplitTotal set
	}{
		{
			name: "no split needed - under limit",
			era: Era{
				Tracks:    createTracks(5, nil),
				StartDate: baseTime,
				EndDate:   baseTime.Add(4 * hour),
			},
			maxTracks:     10,
			wantSubEras:   1,
			wantSplitInfo: false,
		},
		{
			name: "no split needed - exactly at limit",
			era: Era{
				Tracks:    createTracks(10, nil),
				StartDate: baseTime,
				EndDate:   baseTime.Add(9 * hour),
			},
			maxTracks:     10,
			wantSubEras:   1,
			wantSplitInfo: false,
		},
		{
			name: "split into 2 - just over limit",
			era: Era{
				Tracks: createTracks(12, []time.Duration{
					1 * hour, 1 * hour, 1 * hour, 1 * hour,
					5 * hour, // largest gap at index 5
					1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour,
				}),
				StartDate: baseTime,
				EndDate:   baseTime.Add(15 * hour),
			},
			maxTracks:     10,
			wantSubEras:   2,
			wantSplitInfo: true,
		},
		{
			name: "split into 3 - with natural gaps",
			era: Era{
				Tracks: createTracks(25, []time.Duration{
					1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour,
					8 * hour, // gap 1 at index 8
					1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour,
					6 * hour, // gap 2 at index 16
					1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour, 1 * hour,
				}),
				StartDate: baseTime,
				EndDate:   baseTime.Add(30 * hour),
			},
			maxTracks:     10,
			wantSubEras:   3,
			wantSplitInfo: true,
		},
		{
			name: "maxTracks 0 means no limit",
			era: Era{
				Tracks:    createTracks(100, nil),
				StartDate: baseTime,
				EndDate:   baseTime.Add(99 * hour),
			},
			maxTracks:     0,
			wantSubEras:   1,
			wantSplitInfo: false,
		},
		{
			name: "maxTracks negative means no limit",
			era: Era{
				Tracks:    createTracks(100, nil),
				StartDate: baseTime,
				EndDate:   baseTime.Add(99 * hour),
			},
			maxTracks:     -5,
			wantSubEras:   1,
			wantSplitInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eras := []Era{tt.era}
			result := SplitLargeEras(eras, tt.maxTracks)

			if len(result) != tt.wantSubEras {
				t.Errorf("got %d sub-eras, want %d", len(result), tt.wantSubEras)
			}

			if tt.wantSplitInfo && len(result) > 1 {
				for i, era := range result {
					if era.SplitIndex != i+1 {
						t.Errorf("sub-era %d: SplitIndex = %d, want %d", i, era.SplitIndex, i+1)
					}
					if era.SplitTotal != tt.wantSubEras {
						t.Errorf("sub-era %d: SplitTotal = %d, want %d", i, era.SplitTotal, tt.wantSubEras)
					}
				}
			}

			if !tt.wantSplitInfo && len(result) == 1 {
				if result[0].SplitIndex != 0 || result[0].SplitTotal != 0 {
					t.Errorf("unsplit era should have SplitIndex=0, SplitTotal=0, got %d, %d",
						result[0].SplitIndex, result[0].SplitTotal)
				}
			}

			// Verify all tracks are preserved
			totalTracks := 0
			for _, era := range result {
				totalTracks += len(era.Tracks)
			}
			if totalTracks != len(tt.era.Tracks) {
				t.Errorf("total tracks = %d, want %d", totalTracks, len(tt.era.Tracks))
			}
		})
	}
}

func TestSplitLargeErasNaturalGaps(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	hour := time.Hour

	// Create 20 tracks with clear gaps at positions 7 and 14
	tracks := make([]Track, 20)
	current := baseTime
	for i := 0; i < 20; i++ {
		tracks[i] = Track{
			ID:      string(rune('A' + i)),
			AddedAt: current,
		}
		if i == 6 {
			current = current.Add(10 * hour) // Big gap after track 7
		} else if i == 13 {
			current = current.Add(8 * hour) // Second big gap after track 14
		} else {
			current = current.Add(1 * hour)
		}
	}

	era := Era{
		Tracks:    tracks,
		StartDate: tracks[0].AddedAt,
		EndDate:   tracks[len(tracks)-1].AddedAt,
	}

	result := SplitLargeEras([]Era{era}, 8)

	// Should split into 3 sub-eras at the natural gap points
	if len(result) != 3 {
		t.Fatalf("got %d sub-eras, want 3", len(result))
	}

	// First sub-era should have tracks 0-6 (7 tracks)
	if len(result[0].Tracks) != 7 {
		t.Errorf("first sub-era has %d tracks, want 7", len(result[0].Tracks))
	}

	// Second sub-era should have tracks 7-13 (7 tracks)
	if len(result[1].Tracks) != 7 {
		t.Errorf("second sub-era has %d tracks, want 7", len(result[1].Tracks))
	}

	// Third sub-era should have tracks 14-19 (6 tracks)
	if len(result[2].Tracks) != 6 {
		t.Errorf("third sub-era has %d tracks, want 6", len(result[2].Tracks))
	}
}

func TestSplitLargeErasPreservesSmallEras(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	hour := time.Hour

	smallEra := Era{
		Tracks: []Track{
			{ID: "1", AddedAt: baseTime},
			{ID: "2", AddedAt: baseTime.Add(1 * hour)},
			{ID: "3", AddedAt: baseTime.Add(2 * hour)},
		},
		StartDate: baseTime,
		EndDate:   baseTime.Add(2 * hour),
	}

	largeTracks := make([]Track, 50)
	for i := 0; i < 50; i++ {
		largeTracks[i] = Track{
			ID:      string(rune('A' + i)),
			AddedAt: baseTime.Add(time.Duration(100+i) * hour),
		}
	}
	largeEra := Era{
		Tracks:    largeTracks,
		StartDate: largeTracks[0].AddedAt,
		EndDate:   largeTracks[len(largeTracks)-1].AddedAt,
	}

	eras := []Era{smallEra, largeEra}
	result := SplitLargeEras(eras, 20)

	// Small era should be unchanged (1 era)
	// Large era should be split (50/20 = 3 eras)
	// Total: 4 eras
	if len(result) != 4 {
		t.Errorf("got %d eras, want 4", len(result))
	}

	// First era should be the small one, unchanged
	if len(result[0].Tracks) != 3 {
		t.Errorf("first era (small) has %d tracks, want 3", len(result[0].Tracks))
	}
	if result[0].SplitIndex != 0 || result[0].SplitTotal != 0 {
		t.Errorf("small era should not have split info set")
	}
}
