package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/eras"
	spotifyclient "github.com/justestif/go-spotify-era-organizer/internal/spotify"
	syncpkg "github.com/justestif/go-spotify-era-organizer/internal/sync"
)

// Handlers contains HTTP handlers for the web application.
type Handlers struct {
	auth        *spotifyauth.Authenticator
	sessions    SessionManager
	templates   *Templates
	oauthStates *oauthStateStore
	db          *db.DB
	syncService *syncpkg.Service
	eraService  *eras.Service
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
	Auth        *spotifyauth.Authenticator
	Sessions    SessionManager
	Templates   *Templates
	DB          *db.DB
	SyncService *syncpkg.Service
	EraService  *eras.Service
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(deps HandlerDeps) *Handlers {
	return &Handlers{
		auth:        deps.Auth,
		sessions:    deps.Sessions,
		templates:   deps.Templates,
		oauthStates: newOAuthStateStore(),
		db:          deps.DB,
		syncService: deps.SyncService,
		eraService:  deps.EraService,
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
			Email:       spotifyUser.Email,
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

	// Trigger initial sync if this is the user's first time (async)
	if h.syncService != nil && h.db != nil {
		go h.triggerInitialSync(token, userID)
	}

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// triggerInitialSync checks if user needs initial sync and runs it.
func (h *Handlers) triggerInitialSync(token *oauth2.Token, userID string) {
	ctx := context.Background()

	// Check if user has ever synced
	lastSync, err := h.syncService.GetLastSyncTime(ctx, userID)
	if err != nil {
		log.Printf("Error checking last sync time for user %s: %v", userID, err)
		return
	}

	// Only sync if never synced before
	if lastSync != nil {
		return
	}

	log.Printf("Starting initial sync for user %s", userID)

	// Create Spotify client with the token
	httpClient := h.auth.Client(ctx, token)
	spotifyAPI := spotify.New(httpClient)
	client := spotifyclient.New(spotifyAPI)

	// Run sync with force=true (bypass cooldown for initial sync)
	result, err := h.syncService.SyncLikedSongs(ctx, client, userID, true)
	if err != nil {
		log.Printf("Error during initial sync for user %s: %v", userID, err)
		return
	}

	log.Printf("Initial sync complete for user %s: %d tracks synced", userID, result.TracksCount)
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
