package jobs

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/justestif/go-spotify-era-organizer/internal/analysis"
	spotifyclient "github.com/justestif/go-spotify-era-organizer/internal/spotify"
)

// AuthClientFactory creates a Spotify client from an OAuth2 token.
type AuthClientFactory struct {
	Auth *spotifyauth.Authenticator
}

// NewClient creates a new Spotify client from a token stored in job metadata.
func (f *AuthClientFactory) NewClient(ctx context.Context, job *Job) (*spotifyclient.Client, error) {
	tokenVal, ok := job.Meta["token"]
	if !ok {
		return nil, fmt.Errorf("no token in job metadata")
	}
	token, ok := tokenVal.(*oauth2.Token)
	if !ok {
		return nil, fmt.Errorf("invalid token type in job metadata")
	}

	httpClient := f.Auth.Client(ctx, token)
	spotifyAPI := spotify.New(httpClient)
	return spotifyclient.New(spotifyAPI, httpClient), nil
}

// SyncHandler creates a JobHandler that runs the sync pipeline.
func SyncHandler(analysisService *analysis.Service, factory *AuthClientFactory) JobHandler {
	return func(ctx context.Context, job *Job, progress func(int, string)) error {
		progress(0, "Starting sync...")

		client, err := factory.NewClient(ctx, job)
		if err != nil {
			return fmt.Errorf("creating Spotify client: %w", err)
		}

		progress(5, "Checking sync status...")

		userID := job.UserID

		progress(10, "Fetching liked songs from Spotify...")

		result, err := analysisService.RunSync(ctx, client, userID)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		progress(90, fmt.Sprintf("Synced %d tracks, detected %d eras", result.TracksSynced, result.ErasDetected))
		progress(100, fmt.Sprintf("Complete! %d tracks, %d eras", result.TracksSynced, result.ErasDetected))
		return nil
	}
}

// AnalyzeHandler creates a JobHandler that runs the full analysis pipeline.
func AnalyzeHandler(analysisService *analysis.Service, factory *AuthClientFactory) JobHandler {
	return func(ctx context.Context, job *Job, progress func(int, string)) error {
		progress(0, "Starting analysis...")

		client, err := factory.NewClient(ctx, job)
		if err != nil {
			return fmt.Errorf("creating Spotify client: %w", err)
		}

		userID := job.UserID

		progress(10, "Fetching liked songs from Spotify...")

		result, err := analysisService.RunAnalysis(ctx, client, userID, true)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		progress(90, fmt.Sprintf("Found %d tracks, %d eras, %d outliers", result.TotalTracks, result.ErasDetected, result.OutlierCount))
		progress(100, fmt.Sprintf("Complete! %d eras from %d tracks", result.ErasDetected, result.TotalTracks))
		return nil
	}
}

// ReclusterHandler creates a JobHandler that re-runs clustering.
func ReclusterHandler(analysisService *analysis.Service) JobHandler {
	return func(ctx context.Context, job *Job, progress func(int, string)) error {
		progress(0, "Starting re-clustering...")

		// The recluster doesn't need a Spotify client — it works on existing data.
		// For now, just wrap the existing recluster via analysis service or era service.
		// This is a placeholder; the actual recluster still happens inline since it's fast.
		progress(100, "Complete!")
		return nil
	}
}
