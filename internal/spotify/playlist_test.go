package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePlaylist(t *testing.T) {
	tests := []struct {
		name       string
		respStatus int
		respBody   string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "success",
			respStatus: 201,
			respBody:   `{"id": "playlist123", "name": "My Era"}`,
			wantID:     "playlist123",
		},
		{
			name:       "api error",
			respStatus: 403,
			respBody:   `{"error": {"message": "forbidden"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq createPlaylistRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/me/playlists" {
					t.Errorf("expected /me/playlists, got %s", r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
					t.Fatalf("decoding request body: %v", err)
				}
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer server.Close()

			c := &Client{httpClient: server.Client()}
			// Override base URL by replacing the helper temporarily
			origURL := spotifyBaseURL
			defer func() { setBaseURL(origURL) }()
			setBaseURL(server.URL)

			id, err := c.CreatePlaylist(context.Background(), "My Era", "desc", false)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("got id %q, want %q", id, tt.wantID)
			}
			if gotReq.Name != "My Era" {
				t.Errorf("got name %q, want %q", gotReq.Name, "My Era")
			}
			if gotReq.Description != "desc" {
				t.Errorf("got description %q, want %q", gotReq.Description, "desc")
			}
			if gotReq.Public != false {
				t.Error("expected public=false")
			}
		})
	}
}

func TestAddTracksToPlaylist(t *testing.T) {
	tests := []struct {
		name     string
		trackIDs []string
		wantReqs int
		wantErr  bool
	}{
		{
			name:     "empty",
			trackIDs: nil,
			wantReqs: 0,
		},
		{
			name:     "single batch",
			trackIDs: []string{"a", "b", "c"},
			wantReqs: 1,
		},
		{
			name:     "multiple batches",
			trackIDs: makeIDs(150),
			wantReqs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCount := 0
			var allURIs []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqCount++
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.HasPrefix(r.URL.Path, "/playlists/pl1/items") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				var body addItemsRequest
				json.NewDecoder(r.Body).Decode(&body)
				allURIs = append(allURIs, body.URIs...)
				w.WriteHeader(200)
				w.Write([]byte(`{"snapshot_id":"snap"}`))
			}))
			defer server.Close()

			c := &Client{httpClient: server.Client()}
			origURL := spotifyBaseURL
			defer func() { setBaseURL(origURL) }()
			setBaseURL(server.URL)

			err := c.AddTracksToPlaylist(context.Background(), "pl1", tt.trackIDs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reqCount != tt.wantReqs {
				t.Errorf("got %d requests, want %d", reqCount, tt.wantReqs)
			}
			// Verify URIs have spotify:track: prefix
			for _, uri := range allURIs {
				if !strings.HasPrefix(uri, "spotify:track:") {
					t.Errorf("URI missing prefix: %s", uri)
				}
			}
			if len(tt.trackIDs) > 0 && len(allURIs) != len(tt.trackIDs) {
				t.Errorf("got %d URIs, want %d", len(allURIs), len(tt.trackIDs))
			}
		})
	}
}

func makeIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = strings.Repeat("x", 5)
	}
	return ids
}
