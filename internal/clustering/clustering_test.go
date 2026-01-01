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
