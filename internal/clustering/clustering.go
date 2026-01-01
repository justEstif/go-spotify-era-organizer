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
	Tracks     []Track
	StartDate  time.Time
	EndDate    time.Time
	SplitIndex int // 1-based index if this era was split from a larger era (0 if not split)
	SplitTotal int // Total number of sub-eras if split (0 if not split)
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
	MaxTracks      int           // Maximum tracks per era (0 = no limit)
	OutlierMode    OutlierMode   // How to handle small clusters (default: skip)
}

// DefaultConfig returns the recommended default configuration.
func DefaultConfig() Config {
	return Config{
		GapThreshold:   7 * 24 * time.Hour, // 7 days
		MinClusterSize: 3,
		MaxTracks:      30,
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

// gapInfo holds information about a gap between consecutive tracks.
type gapInfo struct {
	index    int           // Index of the track after the gap
	duration time.Duration // Duration of the gap
}

// SplitLargeEras splits eras that exceed maxTracks into smaller sub-eras.
// It uses natural sub-gap detection: finds the largest gaps within an era
// and splits at those points to preserve natural listening patterns.
// If maxTracks is 0 or negative, no splitting is performed.
func SplitLargeEras(eras []Era, maxTracks int) []Era {
	if maxTracks <= 0 {
		return eras
	}

	var result []Era
	for _, era := range eras {
		if len(era.Tracks) <= maxTracks {
			// Era is small enough, keep as-is
			result = append(result, era)
			continue
		}

		// Split this era
		subEras := splitEra(era, maxTracks)
		result = append(result, subEras...)
	}

	return result
}

// splitEra splits a single era into sub-eras at natural gap boundaries.
func splitEra(era Era, maxTracks int) []Era {
	tracks := era.Tracks
	numTracks := len(tracks)

	// Calculate how many splits we need
	numSubEras := (numTracks + maxTracks - 1) / maxTracks // ceiling division
	numSplits := numSubEras - 1

	if numSplits == 0 {
		return []Era{era}
	}

	// Calculate all gaps between consecutive tracks
	gaps := make([]gapInfo, numTracks-1)
	for i := 1; i < numTracks; i++ {
		gaps[i-1] = gapInfo{
			index:    i,
			duration: tracks[i].AddedAt.Sub(tracks[i-1].AddedAt),
		}
	}

	// Sort gaps by duration (largest first)
	slices.SortFunc(gaps, func(a, b gapInfo) int {
		if a.duration > b.duration {
			return -1
		}
		if a.duration < b.duration {
			return 1
		}
		return 0
	})

	// Pick the top N gaps as split points
	splitIndices := make([]int, numSplits)
	for i := 0; i < numSplits; i++ {
		splitIndices[i] = gaps[i].index
	}

	// Sort split indices in ascending order
	slices.Sort(splitIndices)

	// Create sub-eras based on split points
	subEras := make([]Era, numSubEras)
	start := 0
	for i := 0; i < numSubEras; i++ {
		var end int
		if i < numSplits {
			end = splitIndices[i]
		} else {
			end = numTracks
		}

		subTracks := tracks[start:end]
		subEras[i] = Era{
			Tracks:     subTracks,
			StartDate:  subTracks[0].AddedAt,
			EndDate:    subTracks[len(subTracks)-1].AddedAt,
			SplitIndex: i + 1,
			SplitTotal: numSubEras,
		}
		start = end
	}

	return subEras
}
