package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	// redirectURI uses explicit IPv4 loopback as required by Spotify for local development.
	// See: https://developer.spotify.com/documentation/web-api/concepts/redirect-uri
	redirectURI     = "http://127.0.0.1:8080/callback"
	callbackTimeout = 2 * time.Minute
)

var (
	// ErrMissingCredentials is returned when SPOTIFY_ID or SPOTIFY_SECRET is not set.
	ErrMissingCredentials = errors.New("missing SPOTIFY_ID or SPOTIFY_SECRET environment variable")

	// ErrAuthTimeout is returned when the OAuth callback is not received in time.
	ErrAuthTimeout = errors.New("authentication timed out waiting for callback")

	// ErrStateMismatch is returned when the OAuth state parameter doesn't match.
	ErrStateMismatch = errors.New("OAuth state mismatch")
)

// Authenticator handles Spotify OAuth2 authentication.
type Authenticator struct {
	auth  *spotifyauth.Authenticator
	cache *TokenCache
}

// New creates an Authenticator using SPOTIFY_ID and SPOTIFY_SECRET environment variables.
// Returns ErrMissingCredentials if either variable is not set.
func New() (*Authenticator, error) {
	clientID := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, ErrMissingCredentials
	}

	cache, err := DefaultTokenCache()
	if err != nil {
		return nil, fmt.Errorf("creating token cache: %w", err)
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
	)

	return &Authenticator{
		auth:  auth,
		cache: cache,
	}, nil
}

// Authenticate returns an authenticated Spotify client.
// It first checks for a cached token and uses it if valid/refreshable.
// Otherwise, it runs the full OAuth flow.
func (a *Authenticator) Authenticate(ctx context.Context) (*spotify.Client, error) {
	// Try to use cached token
	token, err := a.cache.Load()
	if err != nil {
		return nil, fmt.Errorf("loading cached token: %w", err)
	}

	if token != nil {
		// Create client with cached token - oauth2 will auto-refresh if needed
		client := spotify.New(a.auth.Client(ctx, token), spotify.WithRetry(true))

		// Verify token works by making a simple API call
		_, err := client.CurrentUser(ctx)
		if err == nil {
			// Token works, save potentially refreshed token
			newToken, tokenErr := client.Token()
			if tokenErr == nil && newToken.AccessToken != token.AccessToken {
				_ = a.cache.Save(newToken)
			}
			return client, nil
		}

		// Token didn't work, fall through to full auth flow
		fmt.Println("Cached token invalid, starting new authentication...")
	}

	// Run full OAuth flow
	return a.runOAuthFlow(ctx)
}

// runOAuthFlow performs the full OAuth authorization code flow.
func (a *Authenticator) runOAuthFlow(ctx context.Context) (*spotify.Client, error) {
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// Channel to receive the token from callback
	tokenCh := make(chan *oauth2.Token, 1)
	errCh := make(chan error, 1)

	// Create HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		a.handleCallback(w, r, state, tokenCh, errCh)
	})

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Print auth URL for user
	authURL := a.auth.AuthURL(state)
	fmt.Println("\nTo authenticate, open this URL in your browser:")
	fmt.Println(authURL)
	fmt.Println("\nWaiting for authentication...")

	// Wait for callback or timeout
	var token *oauth2.Token
	select {
	case token = <-tokenCh:
		// Success
	case err := <-errCh:
		_ = server.Shutdown(ctx)
		return nil, err
	case <-time.After(callbackTimeout):
		_ = server.Shutdown(ctx)
		return nil, ErrAuthTimeout
	case <-ctx.Done():
		_ = server.Shutdown(ctx)
		return nil, ctx.Err()
	}

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	// Cache token
	if err := a.cache.Save(token); err != nil {
		// Log but don't fail - auth succeeded
		fmt.Printf("Warning: failed to cache token: %v\n", err)
	}

	client := spotify.New(a.auth.Client(ctx, token), spotify.WithRetry(true))
	return client, nil
}

// handleCallback processes the OAuth callback from Spotify.
func (a *Authenticator) handleCallback(w http.ResponseWriter, r *http.Request, expectedState string, tokenCh chan<- *oauth2.Token, errCh chan<- error) {
	// Verify state
	if r.URL.Query().Get("state") != expectedState {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		errCh <- ErrStateMismatch
		return
	}

	// Check for error response
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, "Authentication failed: "+errMsg, http.StatusBadRequest)
		errCh <- fmt.Errorf("spotify auth error: %s", errMsg)
		return
	}

	// Exchange code for token
	token, err := a.auth.Token(r.Context(), expectedState, r)
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		errCh <- fmt.Errorf("exchanging code for token: %w", err)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
<h1>Authentication Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)

	tokenCh <- token
}

// generateState creates a random state string for OAuth.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Logout removes the cached token.
func (a *Authenticator) Logout() error {
	return a.cache.Delete()
}
