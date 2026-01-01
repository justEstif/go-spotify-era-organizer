package clustering

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

// TagClusterConfig holds tag-based clustering parameters.
type TagClusterConfig struct {
	NumClusters    int // Number of clusters to create (default: 3)
	MinClusterSize int // Minimum tracks per era (smaller clusters become outliers)
	MaxTags        int // Maximum tags to use in vectors (default: 50)
}

// DefaultTagClusterConfig returns the recommended default configuration.
func DefaultTagClusterConfig() TagClusterConfig {
	return TagClusterConfig{
		NumClusters:    3,
		MinClusterSize: 3,
		MaxTags:        50,
	}
}

// MoodEra represents a cluster of tracks grouped by tag similarity.
type MoodEra struct {
	Name      string    // Descriptive name: "Rock & Indie & Pop: Jan 15 - Feb 3, 2024"
	Tracks    []Track   // Tracks in this era
	TopTags   []string  // Top 3 dominant tags for this cluster
	StartDate time.Time // Earliest track add date
	EndDate   time.Time // Latest track add date
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

// DetectMoodEras groups tracks by tag similarity using k-means clustering.
// Returns mood-based eras and outlier tracks that don't fit into any era.
// Tracks without tags are treated as outliers.
func DetectMoodEras(tracks []Track, cfg TagClusterConfig) ([]MoodEra, []Track) {
	if len(tracks) == 0 {
		return nil, nil
	}

	// Apply defaults
	if cfg.NumClusters <= 0 {
		cfg.NumClusters = DefaultTagClusterConfig().NumClusters
	}
	if cfg.MaxTags <= 0 {
		cfg.MaxTags = DefaultTagClusterConfig().MaxTags
	}

	// Separate tracks with and without tags
	var validTracks []*Track
	var noTags []Track

	for i := range tracks {
		t := &tracks[i]
		if len(t.Tags) > 0 {
			validTracks = append(validTracks, t)
		} else {
			noTags = append(noTags, *t)
		}
	}

	// If fewer valid tracks than clusters, everything is an outlier
	if len(validTracks) < cfg.NumClusters {
		var outliers []Track
		for _, t := range validTracks {
			outliers = append(outliers, *t)
		}
		outliers = append(outliers, noTags...)
		return nil, outliers
	}

	// Build tag vocabulary from all tracks
	vocabulary := buildTagVocabulary(validTracks, cfg.MaxTags)
	if len(vocabulary) == 0 {
		// No tags found, all are outliers
		var outliers []Track
		for _, t := range validTracks {
			outliers = append(outliers, *t)
		}
		outliers = append(outliers, noTags...)
		return nil, outliers
	}

	// Build observations for k-means
	observations := make([]trackObservation, len(validTracks))
	for i, t := range validTracks {
		observations[i] = trackObservation{
			track:  t,
			coords: buildTagVector(t, vocabulary),
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
		outliers = append(outliers, noTags...)
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

		// Extract top tags from centroid
		topTags := extractTopTags(cluster.Center, vocabulary, 3)

		// Generate era name
		startDate := clusterTracks[0].AddedAt
		endDate := clusterTracks[len(clusterTracks)-1].AddedAt
		fullName := generateEraName(topTags, startDate, endDate)

		eras = append(eras, MoodEra{
			Name:      fullName,
			Tracks:    clusterTracks,
			TopTags:   topTags,
			StartDate: startDate,
			EndDate:   endDate,
		})
	}

	// Add tracks without tags to outliers
	outliers = append(outliers, noTags...)

	// Sort eras by start date (most recent first)
	slices.SortFunc(eras, func(a, b MoodEra) int {
		return b.StartDate.Compare(a.StartDate) // Descending
	})

	return eras, outliers
}

// tagCount tracks tag name and total count across all tracks.
type tagCount struct {
	name  string
	count int
}

// buildTagVocabulary collects all tags and returns the top N most common.
func buildTagVocabulary(tracks []*Track, maxTags int) []string {
	// Count tag occurrences across all tracks
	counts := make(map[string]int)
	for _, t := range tracks {
		for _, tag := range t.Tags {
			// Normalize tag name to lowercase
			name := strings.ToLower(tag.Name)
			counts[name] += tag.Count
		}
	}

	// Convert to slice for sorting
	tagCounts := make([]tagCount, 0, len(counts))
	for name, count := range counts {
		tagCounts = append(tagCounts, tagCount{name: name, count: count})
	}

	// Sort by count (descending)
	sort.Slice(tagCounts, func(i, j int) bool {
		return tagCounts[i].count > tagCounts[j].count
	})

	// Take top N
	n := min(maxTags, len(tagCounts))
	vocabulary := make([]string, n)
	for i := 0; i < n; i++ {
		vocabulary[i] = tagCounts[i].name
	}

	return vocabulary
}

// buildTagVector creates a feature vector for a track based on its tags.
// Vector values are normalized tag counts (0-1 scale).
func buildTagVector(track *Track, vocabulary []string) clusters.Coordinates {
	// Create vocabulary index for fast lookup
	vocabIndex := make(map[string]int, len(vocabulary))
	for i, tag := range vocabulary {
		vocabIndex[tag] = i
	}

	// Find max count for normalization
	var maxCount int
	for _, tag := range track.Tags {
		if tag.Count > maxCount {
			maxCount = tag.Count
		}
	}
	if maxCount == 0 {
		maxCount = 1 // Avoid division by zero
	}

	// Build vector
	vector := make(clusters.Coordinates, len(vocabulary))
	for _, tag := range track.Tags {
		name := strings.ToLower(tag.Name)
		if idx, ok := vocabIndex[name]; ok {
			// Normalize count to 0-1 scale
			vector[idx] = float64(tag.Count) / float64(maxCount)
		}
	}

	return vector
}

// extractTopTags returns the top N tags from a centroid vector.
func extractTopTags(centroid clusters.Coordinates, vocabulary []string, n int) []string {
	if len(centroid) == 0 || len(vocabulary) == 0 {
		return nil
	}

	// Create (index, weight) pairs
	type tagWeight struct {
		name   string
		weight float64
	}
	weights := make([]tagWeight, len(vocabulary))
	for i, name := range vocabulary {
		weight := 0.0
		if i < len(centroid) {
			weight = centroid[i]
		}
		weights[i] = tagWeight{name: name, weight: weight}
	}

	// Sort by weight (descending)
	sort.Slice(weights, func(i, j int) bool {
		return weights[i].weight > weights[j].weight
	})

	// Take top N with weight > 0
	result := make([]string, 0, n)
	for i := 0; i < len(weights) && len(result) < n; i++ {
		if weights[i].weight > 0 {
			result = append(result, weights[i].name)
		}
	}

	return result
}

// generateEraName creates a descriptive name from top tags and date range.
func generateEraName(topTags []string, start, end time.Time) string {
	const dateFormat = "Jan 2, 2006"

	// Format tags
	var tagPart string
	if len(topTags) == 0 {
		tagPart = "Mixed"
	} else {
		tagPart = strings.Join(topTags, " & ")
	}

	// Format date range
	startStr := start.Format(dateFormat)
	endStr := end.Format(dateFormat)

	if startStr == endStr {
		return fmt.Sprintf("%s: %s", tagPart, startStr)
	}
	return fmt.Sprintf("%s: %s - %s", tagPart, startStr, endStr)
}
