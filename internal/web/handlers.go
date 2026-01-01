package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

// Handlers contains HTTP handlers for the web application.
type Handlers struct {
	auth      *spotifyauth.Authenticator
	sessions  *SessionStore
	templates *Templates
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(auth *spotifyauth.Authenticator, sessions *SessionStore, templates *Templates) *Handlers {
	return &Handlers{
		auth:      auth,
		sessions:  sessions,
		templates: templates,
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

	// Store state in cookie for validation on callback
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	// Redirect to Spotify auth
	url := h.auth.AuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback from Spotify (GET /callback).
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Check for error from Spotify
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, fmt.Sprintf("Spotify auth error: %s", errMsg), http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	// Get user info from Spotify
	client := spotify.New(h.auth.Client(r.Context(), token))
	user, err := client.CurrentUser(r.Context())
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := h.sessions.Create(token, string(user.ID), user.DisplayName)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	h.sessions.SetCookie(w, session)

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// Logout clears the session and redirects to home (POST /auth/logout).
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	session := h.sessions.GetFromRequest(r)
	if session != nil {
		h.sessions.Delete(session.ID)
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
