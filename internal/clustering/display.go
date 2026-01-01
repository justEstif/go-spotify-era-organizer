package clustering

import (
	"fmt"
	"strings"
)

const (
	sampleTrackCount = 3
	dateFormat       = "2006-01-02"
)

// FormatMoodEraSummary returns a human-readable summary of detected mood-based eras.
// Shows mood name, date range, track count, and sample tracks for each era.
func FormatMoodEraSummary(eras []MoodEra, outliers []Track) string {
	var sb strings.Builder

	// Calculate total tracks
	totalTracks := len(outliers)
	for _, era := range eras {
		totalTracks += len(era.Tracks)
	}

	// Header
	if len(eras) == 0 {
		sb.WriteString(fmt.Sprintf("No mood eras found from %d tracks", totalTracks))
		if len(outliers) > 0 {
			sb.WriteString(fmt.Sprintf(" (%d outliers skipped)", len(outliers)))
		}
		sb.WriteString("\n")
		return sb.String()
	}

	eraWord := "era"
	if len(eras) > 1 {
		eraWord = "eras"
	}

	sb.WriteString(fmt.Sprintf("Found %d mood %s from %d tracks", len(eras), eraWord, totalTracks))
	if len(outliers) > 0 {
		sb.WriteString(fmt.Sprintf(" (%d outliers skipped)", len(outliers)))
	}
	sb.WriteString("\n")

	// Era details
	for i, era := range eras {
		sb.WriteString("\n")
		sb.WriteString(formatMoodEra(i+1, era))
	}

	return sb.String()
}

// formatMoodEra formats a single mood era with its sample tracks.
func formatMoodEra(num int, era MoodEra) string {
	var sb strings.Builder

	trackWord := "track"
	if len(era.Tracks) > 1 {
		trackWord = "tracks"
	}

	// Era header with mood name
	sb.WriteString(fmt.Sprintf("Era %d: %s (%d %s)\n", num, era.Name, len(era.Tracks), trackWord))

	// Show top tags
	if len(era.TopTags) > 0 {
		sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(era.TopTags, ", ")))
	}

	// Show sample tracks (first 3)
	sampleCount := min(sampleTrackCount, len(era.Tracks))
	for i := 0; i < sampleCount; i++ {
		track := era.Tracks[i]
		sb.WriteString(fmt.Sprintf("  • \"%s\" - %s\n", track.Name, track.Artist))
	}

	// Show "and N more" if needed
	remaining := len(era.Tracks) - sampleTrackCount
	if remaining > 0 {
		sb.WriteString(fmt.Sprintf("  ... and %d more\n", remaining))
	}

	return sb.String()
}
