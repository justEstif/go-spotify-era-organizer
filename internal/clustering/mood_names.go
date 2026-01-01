package clustering

// generateMoodName creates a descriptive name based on audio feature centroid values.
// Uses a 2x2 energy/valence quadrant system with acousticness modifier.
//
// Quadrants:
//   - High Energy + High Valence = "Upbeat Party"
//   - High Energy + Low Valence  = "Intense & Dark"
//   - Low Energy  + High Valence = "Chill & Happy"
//   - Low Energy  + Low Valence  = "Reflective & Melancholy"
//
// Acousticness modifier: if > 0.6, appends "Acoustic" to the name.
func generateMoodName(centroid map[string]float32) string {
	energy := centroid["energy"]
	valence := centroid["valence"]
	acousticness := centroid["acousticness"]

	var baseName string

	// Determine quadrant based on energy and valence thresholds
	highEnergy := energy > 0.6
	highValence := valence > 0.5

	switch {
	case highEnergy && highValence:
		baseName = "Upbeat Party"
	case highEnergy && !highValence:
		baseName = "Intense & Dark"
	case !highEnergy && highValence:
		baseName = "Chill & Happy"
	default: // low energy, low valence
		baseName = "Reflective & Melancholy"
	}

	// Add acoustic modifier if acousticness is high
	if acousticness > 0.6 {
		return baseName + " (Acoustic)"
	}

	return baseName
}

// MoodCategory represents a mood classification for display purposes.
type MoodCategory struct {
	Name        string  // Display name
	Energy      float32 // Average energy level
	Valence     float32 // Average positivity
	Description string  // Brief description of the mood
}

// GetMoodCategory returns a detailed mood category for a centroid.
func GetMoodCategory(centroid map[string]float32) MoodCategory {
	name := generateMoodName(centroid)
	energy := centroid["energy"]
	valence := centroid["valence"]

	var description string
	switch {
	case energy > 0.6 && valence > 0.5:
		description = "High-energy, positive vibes - perfect for dancing and celebrations"
	case energy > 0.6 && valence <= 0.5:
		description = "Intense, driving energy with darker emotional tones"
	case energy <= 0.6 && valence > 0.5:
		description = "Relaxed and uplifting - great for unwinding"
	default:
		description = "Contemplative and introspective - ideal for quiet moments"
	}

	return MoodCategory{
		Name:        name,
		Energy:      energy,
		Valence:     valence,
		Description: description,
	}
}
