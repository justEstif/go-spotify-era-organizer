// Package clustering implements era detection using gap-based temporal clustering.
package clustering

import (
	"slices"
	"time"
)

// Track represents a song with its add timestamp.
type Track struct {
	ID      string
	Name    string
	Artist  string
	AddedAt time.Time
}

// Era represents a cluster of tracks added during a time period.
type Era struct {
	Tracks    []Track
	StartDate time.Time
	EndDate   time.Time
}

// OutlierMode determines how tracks from small clusters are handled.
type OutlierMode string

const (
	// OutlierModeSkip returns outliers but takes no action on them.
	// This is the default behavior.
	OutlierModeSkip OutlierMode = "skip"
)

// Config holds clustering parameters.
type Config struct {
	GapThreshold   time.Duration // Minimum gap to split eras
	MinClusterSize int           // Minimum tracks per era
	OutlierMode    OutlierMode   // How to handle small clusters (default: skip)
}

// DefaultConfig returns the recommended default configuration.
func DefaultConfig() Config {
	return Config{
		GapThreshold:   7 * 24 * time.Hour, // 7 days
		MinClusterSize: 3,
		OutlierMode:    OutlierModeSkip,
	}
}

// DetectEras groups tracks into temporal eras based on add dates.
// Returns valid eras and outlier tracks that didn't fit into any era.
func DetectEras(tracks []Track, cfg Config) ([]Era, []Track) {
	if len(tracks) == 0 {
		return nil, nil
	}

	// Sort by AddedAt ascending
	sorted := make([]Track, len(tracks))
	copy(sorted, tracks)
	slices.SortFunc(sorted, func(a, b Track) int {
		return a.AddedAt.Compare(b.AddedAt)
	})

	// Build clusters by detecting gaps
	var clusters [][]Track
	current := []Track{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		gap := sorted[i].AddedAt.Sub(sorted[i-1].AddedAt)
		if gap >= cfg.GapThreshold {
			clusters = append(clusters, current)
			current = []Track{sorted[i]}
		} else {
			current = append(current, sorted[i])
		}
	}
	clusters = append(clusters, current) // Don't forget last cluster

	// Filter by minimum size, collect outliers
	var eras []Era
	var outliers []Track

	for _, cluster := range clusters {
		if len(cluster) >= cfg.MinClusterSize {
			eras = append(eras, Era{
				Tracks:    cluster,
				StartDate: cluster[0].AddedAt,
				EndDate:   cluster[len(cluster)-1].AddedAt,
			})
		} else {
			outliers = append(outliers, cluster...)
		}
	}

	return eras, outliers
}
