// Package clustering implements mood-based clustering using audio features.
package clustering

import (
	"fmt"
	"slices"
	"time"

	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

// MoodConfig holds mood-based clustering parameters.
type MoodConfig struct {
	NumClusters    int // Number of clusters to create (default: 3)
	MinClusterSize int // Minimum tracks per era (smaller clusters become outliers)
}

// DefaultMoodConfig returns the recommended default configuration.
func DefaultMoodConfig() MoodConfig {
	return MoodConfig{
		NumClusters:    3,
		MinClusterSize: 3,
	}
}

// MoodEra represents a cluster of tracks grouped by mood/vibe.
type MoodEra struct {
	Name      string             // Descriptive name: "Upbeat Party: Jan 15 - Feb 3, 2024"
	Tracks    []Track            // Tracks in this era
	Centroid  map[string]float32 // Average feature values for this cluster
	StartDate time.Time          // Earliest track add date
	EndDate   time.Time          // Latest track add date
}

// trackObservation wraps a Track to implement clusters.Observation interface.
type trackObservation struct {
	track  *Track
	coords clusters.Coordinates
}

func (o trackObservation) Coordinates() clusters.Coordinates {
	return o.coords
}

func (o trackObservation) Distance(point clusters.Coordinates) float64 {
	return o.coords.Distance(point)
}

// featureNames defines the audio features used for clustering.
var featureNames = []string{"energy", "valence", "danceability", "acousticness"}

// DetectMoodEras groups tracks by audio feature similarity using k-means clustering.
// Returns mood-based eras and outlier tracks that don't fit into any era.
// Tracks missing audio features are treated as outliers.
func DetectMoodEras(tracks []Track, cfg MoodConfig) ([]MoodEra, []Track) {
	if len(tracks) == 0 {
		return nil, nil
	}

	if cfg.NumClusters <= 0 {
		cfg.NumClusters = DefaultMoodConfig().NumClusters
	}

	// Separate tracks with and without audio features
	var validTracks []*Track
	var missingFeatures []Track

	for i := range tracks {
		t := &tracks[i]
		if hasAudioFeatures(t) {
			validTracks = append(validTracks, t)
		} else {
			missingFeatures = append(missingFeatures, *t)
		}
	}

	// If fewer valid tracks than clusters, everything is an outlier
	if len(validTracks) < cfg.NumClusters {
		var outliers []Track
		for _, t := range validTracks {
			outliers = append(outliers, *t)
		}
		outliers = append(outliers, missingFeatures...)
		return nil, outliers
	}

	// Build observations for k-means
	observations := make([]trackObservation, len(validTracks))
	for i, t := range validTracks {
		observations[i] = trackObservation{
			track:  t,
			coords: extractFeatures(t),
		}
	}

	// Convert to clusters.Observations interface
	var obs clusters.Observations
	for _, o := range observations {
		obs = append(obs, o)
	}

	// Run k-means clustering
	km := kmeans.New()
	result, err := km.Partition(obs, cfg.NumClusters)
	if err != nil {
		// On error, treat all as outliers
		fmt.Printf("Warning: k-means clustering failed: %v\n", err)
		var outliers []Track
		for _, t := range validTracks {
			outliers = append(outliers, *t)
		}
		outliers = append(outliers, missingFeatures...)
		return nil, outliers
	}

	// Build MoodEras from clusters
	var eras []MoodEra
	var outliers []Track

	for _, cluster := range result {
		// Extract tracks from this cluster
		var clusterTracks []Track
		for _, obs := range cluster.Observations {
			if to, ok := obs.(trackObservation); ok {
				clusterTracks = append(clusterTracks, *to.track)
			}
		}

		// Check minimum size
		if len(clusterTracks) < cfg.MinClusterSize {
			outliers = append(outliers, clusterTracks...)
			continue
		}

		// Sort tracks by AddedAt
		slices.SortFunc(clusterTracks, func(a, b Track) int {
			return a.AddedAt.Compare(b.AddedAt)
		})

		// Build centroid map
		centroid := make(map[string]float32)
		for i, name := range featureNames {
			centroid[name] = float32(cluster.Center[i])
		}

		// Generate era name
		startDate := clusterTracks[0].AddedAt
		endDate := clusterTracks[len(clusterTracks)-1].AddedAt
		moodName := generateMoodName(centroid)
		fullName := formatEraName(moodName, startDate, endDate)

		eras = append(eras, MoodEra{
			Name:      fullName,
			Tracks:    clusterTracks,
			Centroid:  centroid,
			StartDate: startDate,
			EndDate:   endDate,
		})
	}

	// Add tracks with missing features to outliers
	outliers = append(outliers, missingFeatures...)

	// Sort eras by start date (most recent first)
	slices.SortFunc(eras, func(a, b MoodEra) int {
		return b.StartDate.Compare(a.StartDate) // Descending
	})

	return eras, outliers
}

// hasAudioFeatures checks if a track has the required audio features for clustering.
func hasAudioFeatures(t *Track) bool {
	return t.Energy != nil &&
		t.Valence != nil &&
		t.Danceability != nil &&
		t.Acousticness != nil
}

// extractFeatures extracts the audio features used for clustering as a coordinate vector.
func extractFeatures(t *Track) clusters.Coordinates {
	return clusters.Coordinates{
		float64(*t.Energy),
		float64(*t.Valence),
		float64(*t.Danceability),
		float64(*t.Acousticness),
	}
}

// formatEraName combines a mood name with date range.
func formatEraName(moodName string, start, end time.Time) string {
	const dateFormat = "Jan 2, 2006"
	startStr := start.Format(dateFormat)
	endStr := end.Format(dateFormat)

	if startStr == endStr {
		return fmt.Sprintf("%s: %s", moodName, startStr)
	}
	return fmt.Sprintf("%s: %s - %s", moodName, startStr, endStr)
}
