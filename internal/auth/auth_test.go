package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenCache_SaveAndLoad(t *testing.T) {
	tests := []struct {
		name  string
		token *oauth2.Token
	}{
		{
			name: "basic token",
			token: &oauth2.Token{
				AccessToken:  "test-access-token",
				TokenType:    "Bearer",
				RefreshToken: "test-refresh-token",
				Expiry:       time.Now().Add(time.Hour),
			},
		},
		{
			name: "token without refresh",
			token: &oauth2.Token{
				AccessToken: "access-only",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(30 * time.Minute),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "token.json")
			cache := NewTokenCache(path)

			// Save token
			if err := cache.Save(tt.token); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			// Load token
			loaded, err := cache.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if loaded == nil {
				t.Fatal("Load() returned nil token")
			}

			if loaded.AccessToken != tt.token.AccessToken {
				t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tt.token.AccessToken)
			}

			if loaded.RefreshToken != tt.token.RefreshToken {
				t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tt.token.RefreshToken)
			}

			if loaded.TokenType != tt.token.TokenType {
				t.Errorf("TokenType = %q, want %q", loaded.TokenType, tt.token.TokenType)
			}
		})
	}
}

func TestTokenCache_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "token.json")
	cache := NewTokenCache(path)

	token, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if token != nil {
		t.Errorf("Load() = %v, want nil for non-existent file", token)
	}
}

func TestTokenCache_SaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deeply", "token.json")
	cache := NewTokenCache(path)

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	if err := cache.Save(token); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	parentDir := filepath.Dir(path)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Save() did not create parent directory")
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Save() did not create token file")
	}
}

func TestTokenCache_SaveNilToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	cache := NewTokenCache(path)

	err := cache.Save(nil)
	if err == nil {
		t.Error("Save(nil) should return error")
	}
}

func TestTokenCache_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	cache := NewTokenCache(path)

	// Save a token first
	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := cache.Save(token); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Delete it
	if err := cache.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Delete() did not remove token file")
	}
}

func TestTokenCache_DeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	cache := NewTokenCache(path)

	// Should not error when file doesn't exist
	if err := cache.Delete(); err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent file", err)
	}
}

func TestTokenCache_Path(t *testing.T) {
	path := "/custom/path/token.json"
	cache := NewTokenCache(path)

	if cache.Path() != path {
		t.Errorf("Path() = %q, want %q", cache.Path(), path)
	}
}

func TestTokenCache_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	cache := NewTokenCache(path)

	token := &oauth2.Token{
		AccessToken: "secret-token",
		TokenType:   "Bearer",
	}

	if err := cache.Save(token); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Check file is not world-readable (0600)
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		t.Errorf("File permissions = %o, want 0600 (no group/other access)", mode)
	}
}

func TestNew_MissingCredentials(t *testing.T) {
	// Clear env vars
	originalID := os.Getenv("SPOTIFY_ID")
	originalSecret := os.Getenv("SPOTIFY_SECRET")
	defer func() {
		os.Setenv("SPOTIFY_ID", originalID)
		os.Setenv("SPOTIFY_SECRET", originalSecret)
	}()

	tests := []struct {
		name   string
		id     string
		secret string
	}{
		{"both missing", "", ""},
		{"id missing", "", "secret"},
		{"secret missing", "id", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SPOTIFY_ID", tt.id)
			os.Setenv("SPOTIFY_SECRET", tt.secret)

			_, err := New()
			if err != ErrMissingCredentials {
				t.Errorf("New() error = %v, want ErrMissingCredentials", err)
			}
		})
	}
}

func TestNew_WithCredentials(t *testing.T) {
	// Set env vars
	originalID := os.Getenv("SPOTIFY_ID")
	originalSecret := os.Getenv("SPOTIFY_SECRET")
	defer func() {
		os.Setenv("SPOTIFY_ID", originalID)
		os.Setenv("SPOTIFY_SECRET", originalSecret)
	}()

	os.Setenv("SPOTIFY_ID", "test-client-id")
	os.Setenv("SPOTIFY_SECRET", "test-client-secret")

	auth, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if auth == nil {
		t.Error("New() returned nil authenticator")
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error = %v", err)
	}

	if len(state1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("generateState() length = %d, want 32", len(state1))
	}

	// Verify randomness - generate another and compare
	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error = %v", err)
	}

	if state1 == state2 {
		t.Error("generateState() returned same value twice")
	}
}
