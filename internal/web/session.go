// Package web provides the HTTP server and web UI for the Spotify Era Organizer.
package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const (
	sessionCookieName = "session_id"
	sessionTTL        = 24 * time.Hour
)

// Session represents an authenticated user session.
type Session struct {
	ID        string
	Token     *oauth2.Token
	UserID    string
	UserName  string
	CreatedAt time.Time
}

// SessionStore manages user sessions in memory.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates a new in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create generates a new session with the given token and user info.
func (s *SessionStore) Create(token *oauth2.Token, userID, userName string) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        id,
		Token:     token,
		UserID:    userID,
		UserName:  userName,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil
	}

	// Check if session has expired
	if time.Since(session.CreatedAt) > sessionTTL {
		return nil
	}

	return session
}

// Delete removes a session by ID.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// UpdateToken updates the OAuth token for a session.
func (s *SessionStore) UpdateToken(id string, token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[id]; ok {
		session.Token = token
	}
}

// GetFromRequest extracts the session from the request cookie.
func (s *SessionStore) GetFromRequest(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return s.Get(cookie.Value)
}

// SetCookie sets the session cookie on the response.
func (s *SessionStore) SetCookie(w http.ResponseWriter, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// ClearCookie removes the session cookie from the response.
func (s *SessionStore) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// generateSessionID creates a cryptographically random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
