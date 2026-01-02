// Package web provides the HTTP server and web UI for the Spotify Era Organizer.
package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/justestif/go-spotify-era-organizer/internal/db"
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

// SessionManager defines the interface for session management.
type SessionManager interface {
	Create(ctx context.Context, token *oauth2.Token, userID, userName string) (*Session, error)
	Get(ctx context.Context, id string) *Session
	Delete(ctx context.Context, id string)
	UpdateToken(ctx context.Context, id string, token *oauth2.Token)
	GetFromRequest(r *http.Request) *Session
	SetCookie(w http.ResponseWriter, session *Session)
	ClearCookie(w http.ResponseWriter)
}

// ============================================================================
// In-Memory Session Store (for development/testing)
// ============================================================================

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
func (s *SessionStore) Create(_ context.Context, token *oauth2.Token, userID, userName string) (*Session, error) {
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
func (s *SessionStore) Get(_ context.Context, id string) *Session {
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
func (s *SessionStore) Delete(_ context.Context, id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// UpdateToken updates the OAuth token for a session.
func (s *SessionStore) UpdateToken(_ context.Context, id string, token *oauth2.Token) {
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
	return s.Get(r.Context(), cookie.Value)
}

// SetCookie sets the session cookie on the response.
func (s *SessionStore) SetCookie(w http.ResponseWriter, session *Session) {
	setCookie(w, session)
}

// ClearCookie removes the session cookie from the response.
func (s *SessionStore) ClearCookie(w http.ResponseWriter) {
	clearCookie(w)
}

// ============================================================================
// Database-Backed Session Store
// ============================================================================

// DBSessionStore manages user sessions in PostgreSQL.
type DBSessionStore struct {
	database *db.DB
}

// NewDBSessionStore creates a new database-backed session store.
func NewDBSessionStore(database *db.DB) *DBSessionStore {
	return &DBSessionStore{database: database}
}

// Create generates a new session and stores it in the database.
func (s *DBSessionStore) Create(ctx context.Context, token *oauth2.Token, userID, userName string) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dbSession := &db.Session{
		ID:           id,
		UserID:       userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sessionTTL),
	}

	if err := s.database.Sessions().Create(ctx, dbSession); err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		Token:     token,
		UserID:    userID,
		UserName:  userName,
		CreatedAt: now,
	}, nil
}

// Get retrieves a session by ID from the database.
func (s *DBSessionStore) Get(ctx context.Context, id string) *Session {
	dbSession, err := s.database.Sessions().Get(ctx, id)
	if err != nil {
		return nil
	}

	// Get user info for the session
	user, err := s.database.Users().Get(ctx, dbSession.UserID)
	if err != nil {
		return nil
	}

	return &Session{
		ID: dbSession.ID,
		Token: &oauth2.Token{
			AccessToken:  dbSession.AccessToken,
			RefreshToken: dbSession.RefreshToken,
			Expiry:       dbSession.TokenExpiry,
			TokenType:    "Bearer",
		},
		UserID:    dbSession.UserID,
		UserName:  user.DisplayName,
		CreatedAt: dbSession.CreatedAt,
	}
}

// Delete removes a session from the database.
func (s *DBSessionStore) Delete(ctx context.Context, id string) {
	_ = s.database.Sessions().Delete(ctx, id)
}

// UpdateToken updates the OAuth token for a session in the database.
func (s *DBSessionStore) UpdateToken(ctx context.Context, id string, token *oauth2.Token) {
	_ = s.database.Sessions().UpdateToken(ctx, id, token.AccessToken, token.RefreshToken, token.Expiry)
}

// GetFromRequest extracts the session from the request cookie.
func (s *DBSessionStore) GetFromRequest(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return s.Get(r.Context(), cookie.Value)
}

// SetCookie sets the session cookie on the response.
func (s *DBSessionStore) SetCookie(w http.ResponseWriter, session *Session) {
	setCookie(w, session)
}

// ClearCookie removes the session cookie from the response.
func (s *DBSessionStore) ClearCookie(w http.ResponseWriter) {
	clearCookie(w)
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateSessionID creates a cryptographically random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// setCookie sets the session cookie on the response.
func setCookie(w http.ResponseWriter, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// clearCookie removes the session cookie from the response.
func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// Ensure both stores implement SessionManager.
var (
	_ SessionManager = (*SessionStore)(nil)
	_ SessionManager = (*DBSessionStore)(nil)
)
