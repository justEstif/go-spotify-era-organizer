package lastfm

import (
	"errors"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantKey  string
		wantErr  error
	}{
		{
			name:     "valid API key",
			envValue: "abc123def456abc123def456abc12345",
			wantKey:  "abc123def456abc123def456abc12345",
			wantErr:  nil,
		},
		{
			name:     "missing API key",
			envValue: "",
			wantKey:  "",
			wantErr:  ErrMissingAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env var
			original := os.Getenv("LASTFM_API_KEY")
			defer os.Setenv("LASTFM_API_KEY", original)

			if tt.envValue == "" {
				os.Unsetenv("LASTFM_API_KEY")
			} else {
				os.Setenv("LASTFM_API_KEY", tt.envValue)
			}

			cfg, err := LoadConfig()

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if cfg == nil {
					t.Fatal("LoadConfig() returned nil config with no error")
				}
				if cfg.APIKey != tt.wantKey {
					t.Errorf("LoadConfig() APIKey = %v, want %v", cfg.APIKey, tt.wantKey)
				}
			} else {
				if cfg != nil {
					t.Errorf("LoadConfig() returned non-nil config with error")
				}
			}
		})
	}
}
