package clustering

import (
	"fmt"
	"strings"
)

const (
	sampleTrackCount = 3
	dateFormat       = "2006-01-02"
)

// FormatEraSummary returns a human-readable summary of detected eras.
// Shows date range, track count, and first 3 sample tracks for each era.
// Outliers are summarized by count only.
func FormatEraSummary(eras []Era, outliers []Track) string {
	var sb strings.Builder

	// Calculate total tracks
	totalTracks := len(outliers)
	for _, era := range eras {
		totalTracks += len(era.Tracks)
	}

	// Header
	if len(eras) == 0 {
		sb.WriteString(fmt.Sprintf("No eras found from %d tracks", totalTracks))
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

	sb.WriteString(fmt.Sprintf("Found %d %s from %d tracks", len(eras), eraWord, totalTracks))
	if len(outliers) > 0 {
		sb.WriteString(fmt.Sprintf(" (%d outliers skipped)", len(outliers)))
	}
	sb.WriteString("\n")

	// Era details
	for i, era := range eras {
		sb.WriteString("\n")
		sb.WriteString(formatEra(i+1, era))
	}

	return sb.String()
}

// formatEra formats a single era with its sample tracks.
func formatEra(num int, era Era) string {
	var sb strings.Builder

	startDate := era.StartDate.Format(dateFormat)
	endDate := era.EndDate.Format(dateFormat)

	trackWord := "track"
	if len(era.Tracks) > 1 {
		trackWord = "tracks"
	}

	sb.WriteString(fmt.Sprintf("Era %d: %s to %s (%d %s)\n",
		num, startDate, endDate, len(era.Tracks), trackWord))

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
