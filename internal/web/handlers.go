package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/justestif/go-spotify-era-organizer/internal/analysis"
	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/eras"
	spotifyclient "github.com/justestif/go-spotify-era-organizer/internal/spotify"
	syncpkg "github.com/justestif/go-spotify-era-organizer/internal/sync"
)

// Handlers contains HTTP handlers for the web application.
type Handlers struct {
	auth            *spotifyauth.Authenticator
	sessions        SessionManager
	templates       *Templates
	oauthStates     *oauthStateStore
	db              *db.DB
	syncService     *syncpkg.Service
	eraService      *eras.Service
	analysisService *analysis.Service
}

// oauthStateStore stores OAuth state tokens server-side to avoid cookie issues
// with localhost vs 127.0.0.1 during development.
type oauthStateStore struct {
	mu     sync.RWMutex
	states map[string]time.Time
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{
		states: make(map[string]time.Time),
	}
}

func (s *oauthStateStore) Set(state string) {
	s.mu.Lock()
	s.states[state] = time.Now()
	s.mu.Unlock()
}

func (s *oauthStateStore) Validate(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	created, ok := s.states[state]
	if !ok {
		return false
	}

	// Remove the state (single use)
	delete(s.states, state)

	// Check if state is expired (5 minutes)
	return time.Since(created) < 5*time.Minute
}

// HandlerDeps contains dependencies for handlers.
type HandlerDeps struct {
	Auth            *spotifyauth.Authenticator
	Sessions        SessionManager
	Templates       *Templates
	DB              *db.DB
	SyncService     *syncpkg.Service
	EraService      *eras.Service
	AnalysisService *analysis.Service
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(deps HandlerDeps) *Handlers {
	return &Handlers{
		auth:            deps.Auth,
		sessions:        deps.Sessions,
		templates:       deps.Templates,
		oauthStates:     newOAuthStateStore(),
		db:              deps.DB,
		syncService:     deps.SyncService,
		eraService:      deps.EraService,
		analysisService: deps.AnalysisService,
	}
}

// Home handles the home page (GET /).
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)

	data := HomePageData{
		PageData: PageData{
			Title:       "Spotify Era Organizer",
			CurrentPath: r.URL.Path,
		},
		Authenticated: session != nil,
	}

	if session != nil {
		data.User = &UserData{
			ID:   session.UserID,
			Name: session.UserName,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "home", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Login initiates the Spotify OAuth flow (GET /auth/login).
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	// Generate state for CSRF protection
	state, err := generateOAuthState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store state server-side (avoids cookie issues with localhost vs 127.0.0.1)
	h.oauthStates.Set(state)

	// Redirect to Spotify auth
	url := h.auth.AuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback from Spotify (GET /callback).
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check for error from Spotify
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, fmt.Sprintf("Spotify auth error: %s", errMsg), http.StatusBadRequest)
		return
	}

	// Verify state (stored server-side)
	state := r.URL.Query().Get("state")
	if state == "" || !h.oauthStates.Validate(state) {
		http.Error(w, "Invalid or expired state. Please try logging in again.", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.auth.Token(ctx, state, r)
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	// Get user info from Spotify
	httpClient := h.auth.Client(ctx, token)
	spotifyAPI := spotify.New(httpClient)
	spotifyUser, err := spotifyAPI.CurrentUser(ctx)
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	userID := string(spotifyUser.ID)
	displayName := spotifyUser.DisplayName

	// Upsert user in database (if DB is available)
	if h.db != nil {
		user := &db.User{
			ID:          userID,
			DisplayName: displayName,
		}
		if err := h.db.Users().Upsert(ctx, user); err != nil {
			log.Printf("Warning: failed to upsert user: %v", err)
			// Continue anyway - session can still work
		}
	}

	// Create session
	session, err := h.sessions.Create(ctx, token, userID, displayName)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	h.sessions.SetCookie(w, session)

	// Trigger initial analysis if this is the user's first time (async)
	if h.analysisService != nil {
		go h.triggerInitialAnalysis(token, userID)
	}

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// triggerInitialAnalysis checks if user needs initial sync and runs the full pipeline.
func (h *Handlers) triggerInitialAnalysis(token *oauth2.Token, userID string) {
	ctx := context.Background()

	// Check if user has ever synced
	lastSync, err := h.analysisService.GetLastSyncTime(ctx, userID)
	if err != nil {
		log.Printf("Error checking last sync time for user %s: %v", userID, err)
		return
	}

	// Only run if never synced before
	if lastSync != nil {
		return
	}

	log.Printf("Starting initial analysis for user %s", userID)

	// Create Spotify client with the token
	httpClient := h.auth.Client(ctx, token)
	spotifyAPI := spotify.New(httpClient)
	client := spotifyclient.New(spotifyAPI, httpClient)

	// Run full pipeline with force=true (bypass cooldown for initial sync)
	result, err := h.analysisService.RunAnalysis(ctx, client, userID, true)
	if err != nil {
		log.Printf("Error during initial analysis for user %s: %v", userID, err)
		return
	}

	log.Printf("Initial analysis complete for user %s: %d tracks, %d eras", userID, result.TracksSynced, result.ErasDetected)
}

// Logout clears the session and redirects to home (POST /auth/logout).
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := h.sessions.GetFromRequest(r)
	if session != nil {
		h.sessions.Delete(ctx, session.ID)
	}

	h.sessions.ClearCookie(w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// generateOAuthState creates a random state string for OAuth.
func generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Eras handles the eras page (GET /eras).
func (h *Handlers) Eras(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		http.Redirect(w, r, "/auth/login", http.StatusTemporaryRedirect)
		return
	}

	ctx := r.Context()

	// Get user's eras from database
	var erasData []EraData
	if h.eraService != nil {
		dbEras, err := h.eraService.GetUserEras(ctx, session.UserID)
		if err != nil {
			log.Printf("Error getting user eras: %v", err)
			http.Error(w, "Failed to load eras", http.StatusInternalServerError)
			return
		}

		// Convert to template-friendly format and get track counts
		for _, era := range dbEras {
			trackCount := 0
			if h.db != nil {
				count, err := h.db.Eras().GetTrackCount(ctx, era.ID)
				if err != nil {
					log.Printf("Error getting track count for era %s: %v", era.ID, err)
				} else {
					trackCount = count
				}
			}

			erasData = append(erasData, EraData{
				ID:         era.ID.String(),
				Name:       era.Name,
				TopTags:    era.TopTags,
				StartDate:  era.StartDate,
				EndDate:    era.EndDate,
				TrackCount: trackCount,
				PlaylistID: era.PlaylistID,
			})
		}
	}

	// Get sync status for the header
	var syncStatus *SyncStatusData
	if h.analysisService != nil {
		lastSync, err := h.analysisService.GetLastSyncTime(ctx, session.UserID)
		if err != nil {
			log.Printf("Error getting last sync time: %v", err)
		}
		canSync, nextTime, err := h.analysisService.CanSync(ctx, session.UserID)
		if err != nil {
			log.Printf("Error checking sync status: %v", err)
		}
		syncStatus = &SyncStatusData{
			LastSync: lastSync,
			CanSync:  canSync,
		}
		if !nextTime.IsZero() {
			syncStatus.NextSyncAvailable = &nextTime
		}
	}

	data := ErasPageData{
		PageData: PageData{
			Title:       "Your Eras - Spotify Era Organizer",
			CurrentPath: r.URL.Path,
			User: &UserData{
				ID:   session.UserID,
				Name: session.UserName,
			},
		},
		Eras:       erasData,
		SyncStatus: syncStatus,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Render(w, "eras", data); err != nil {
		log.Printf("Error rendering eras template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// EraTracks handles fetching tracks for an era (GET /eras/{id}/tracks).
// This is an HTMX partial endpoint.
func (h *Handlers) EraTracks(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eraID := chi.URLParam(r, "id")
	if eraID == "" {
		http.Error(w, "Era ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get tracks for the era
	var tracks []TrackData
	if h.eraService != nil {
		dbTracks, err := h.eraService.GetEraTracks(ctx, eraID)
		if err != nil {
			log.Printf("Error getting era tracks: %v", err)
			http.Error(w, "Failed to load tracks", http.StatusInternalServerError)
			return
		}

		for _, t := range dbTracks {
			album := ""
			if t.Album != nil {
				album = *t.Album
			}
			tracks = append(tracks, TrackData{
				ID:     t.ID,
				Name:   t.Name,
				Artist: t.Artist,
				Album:  album,
			})
		}
	}

	// Render only the track list partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.RenderPartial(w, "era-tracks", tracks); err != nil {
		log.Printf("Error rendering era tracks: %v", err)
		http.Error(w, "Failed to render tracks", http.StatusInternalServerError)
		return
	}
}

// AnalyzeResponse is the JSON response for POST /api/analyze.
type AnalyzeResponse struct {
	EraCount     int    `json:"era_count"`
	OutlierCount int    `json:"outlier_count"`
	TotalTracks  int    `json:"total_tracks"`
	Message      string `json:"message"`
}

// ErrorResponse is the JSON response for errors.
type ErrorResponse struct {
	Error string `json:"error"`
}

// EraJSON is the JSON representation of an era.
type EraJSON struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	TopTags    []string `json:"top_tags"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	TrackCount int      `json:"track_count,omitempty"`
}

// TrackJSON is the JSON representation of a track.
type TrackJSON struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Album  string `json:"album,omitempty"`
}

// Analyze handles the full analysis pipeline (POST /api/analyze).
// This triggers: sync → tags → clustering → eras.
func (h *Handlers) Analyze(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.analysisService == nil {
		h.jsonError(w, "Analysis service not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	userID := session.UserID

	// Create Spotify client
	httpClient := h.auth.Client(ctx, session.Token)
	spotifyAPI := spotify.New(httpClient)
	client := spotifyclient.New(spotifyAPI, httpClient)

	log.Printf("Starting analysis for user %s", userID)

	result, err := h.analysisService.RunAnalysis(ctx, client, userID, false)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := AnalyzeResponse{
		EraCount:     result.ErasDetected,
		OutlierCount: result.OutlierCount,
		TotalTracks:  result.TotalTracks,
		Message:      fmt.Sprintf("Detected %d eras from %d tracks", result.ErasDetected, result.TotalTracks),
	}

	h.jsonResponse(w, resp, http.StatusOK)
}

// GetEras returns all eras for the authenticated user (GET /api/eras).
func (h *Handlers) GetEras(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	if h.eraService == nil {
		h.jsonError(w, "Era service not configured", http.StatusServiceUnavailable)
		return
	}

	dbEras, err := h.eraService.GetUserEras(ctx, session.UserID)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Failed to get eras: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to JSON format with track counts
	result := make([]EraJSON, 0, len(dbEras))
	for _, era := range dbEras {
		trackCount := 0
		if h.db != nil {
			count, err := h.db.Eras().GetTrackCount(ctx, era.ID)
			if err == nil {
				trackCount = count
			}
		}

		result = append(result, EraJSON{
			ID:         era.ID.String(),
			Name:       era.Name,
			TopTags:    era.TopTags,
			StartDate:  era.StartDate.Format("2006-01-02"),
			EndDate:    era.EndDate.Format("2006-01-02"),
			TrackCount: trackCount,
		})
	}

	h.jsonResponse(w, result, http.StatusOK)
}

// GetEraTracksAPI returns tracks for a specific era (GET /api/eras/{id}/tracks).
func (h *Handlers) GetEraTracksAPI(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eraID := chi.URLParam(r, "id")
	if eraID == "" {
		h.jsonError(w, "Era ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if h.eraService == nil {
		h.jsonError(w, "Era service not configured", http.StatusServiceUnavailable)
		return
	}

	dbTracks, err := h.eraService.GetEraTracks(ctx, eraID)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Failed to get tracks: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to JSON format
	result := make([]TrackJSON, 0, len(dbTracks))
	for _, t := range dbTracks {
		album := ""
		if t.Album != nil {
			album = *t.Album
		}
		result = append(result, TrackJSON{
			ID:     t.ID,
			Name:   t.Name,
			Artist: t.Artist,
			Album:  album,
		})
	}

	h.jsonResponse(w, result, http.StatusOK)
}

// jsonResponse writes a JSON response.
func (h *Handlers) jsonResponse(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// jsonError writes a JSON error response.
func (h *Handlers) jsonError(w http.ResponseWriter, message string, status int) {
	h.jsonResponse(w, ErrorResponse{Error: message}, status)
}

// SyncResponse is the JSON response for POST /api/sync.
type SyncResponse struct {
	TracksSynced int       `json:"tracks_synced"`
	NewTracks    int       `json:"new_tracks,omitempty"`
	ErasDetected int       `json:"eras_detected"`
	SyncedAt     time.Time `json:"synced_at"`
	Message      string    `json:"message"`
}

// SyncErrorResponse is the JSON error response for sync cooldown.
type SyncErrorResponse struct {
	Error             string     `json:"error"`
	NextSyncAvailable *time.Time `json:"next_sync_available,omitempty"`
}

// SyncStatusResponse is the JSON response for GET /api/sync/status.
type SyncStatusResponse struct {
	LastSync          *time.Time `json:"last_sync,omitempty"`
	CanSync           bool       `json:"can_sync"`
	NextSyncAvailable *time.Time `json:"next_sync_available,omitempty"`
}

// GetSyncStatus returns the sync status for the authenticated user (GET /api/sync/status).
func (h *Handlers) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	if h.analysisService == nil {
		h.jsonError(w, "Analysis service not configured", http.StatusServiceUnavailable)
		return
	}

	// Get last sync time
	lastSync, err := h.analysisService.GetLastSyncTime(ctx, session.UserID)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Failed to get sync status: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if sync is allowed
	canSync, nextTime, err := h.analysisService.CanSync(ctx, session.UserID)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Failed to check sync status: %v", err), http.StatusInternalServerError)
		return
	}

	resp := SyncStatusResponse{
		LastSync: lastSync,
		CanSync:  canSync,
	}
	if !nextTime.IsZero() {
		resp.NextSyncAvailable = &nextTime
	}

	h.jsonResponse(w, resp, http.StatusOK)
}

// SyncLibrary syncs liked songs from Spotify and re-detects eras (POST /api/sync).
func (h *Handlers) SyncLibrary(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.analysisService == nil {
		h.jsonError(w, "Analysis service not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	userID := session.UserID

	// Check cooldown before doing work
	canSync, nextTime, err := h.analysisService.CanSync(ctx, userID)
	if err != nil {
		h.jsonError(w, fmt.Sprintf("Failed to check sync status: %v", err), http.StatusInternalServerError)
		return
	}
	if !canSync {
		resp := SyncErrorResponse{
			Error:             "Sync not available yet",
			NextSyncAvailable: &nextTime,
		}
		h.jsonResponse(w, resp, http.StatusLocked)
		return
	}

	// Create Spotify client
	httpClient := h.auth.Client(ctx, session.Token)
	spotifyAPI := spotify.New(httpClient)
	client := spotifyclient.New(spotifyAPI, httpClient)

	log.Printf("Starting sync for user %s", userID)

	result, err := h.analysisService.RunSync(ctx, client, userID)
	if err != nil {
		if errors.Is(err, syncpkg.ErrSyncTooRecent) {
			h.jsonError(w, "Sync not available yet", http.StatusLocked)
			return
		}
		h.jsonError(w, fmt.Sprintf("Sync failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := SyncResponse{
		TracksSynced: result.TracksSynced,
		NewTracks:    result.NewTracks,
		ErasDetected: result.ErasDetected,
		SyncedAt:     result.SyncedAt,
		Message:      fmt.Sprintf("Synced %d tracks, detected %d eras", result.TracksSynced, result.ErasDetected),
	}

	h.jsonResponse(w, resp, http.StatusOK)
}
