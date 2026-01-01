package lastfm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetTags(t *testing.T) {
	tests := []struct {
		name           string
		artist         string
		track          string
		trackResponse  any
		artistResponse any
		wantTags       []Tag
		wantErr        error
	}{
		{
			name:   "track has tags",
			artist: "Radiohead",
			track:  "Paranoid Android",
			trackResponse: trackTagsResponse{
				TopTags: struct {
					Tag  []Tag `json:"tag"`
					Attr struct {
						Artist string `json:"artist"`
						Track  string `json:"track"`
					} `json:"@attr"`
				}{
					Tag: []Tag{
						{Name: "alternative", Count: 100, URL: "http://last.fm/tag/alternative"},
						{Name: "rock", Count: 80, URL: "http://last.fm/tag/rock"},
					},
				},
			},
			wantTags: []Tag{
				{Name: "alternative", Count: 100, URL: "http://last.fm/tag/alternative"},
				{Name: "rock", Count: 80, URL: "http://last.fm/tag/rock"},
			},
			wantErr: nil,
		},
		{
			name:   "track empty falls back to artist",
			artist: "Cher",
			track:  "Unknown Song",
			trackResponse: trackTagsResponse{
				TopTags: struct {
					Tag  []Tag `json:"tag"`
					Attr struct {
						Artist string `json:"artist"`
						Track  string `json:"track"`
					} `json:"@attr"`
				}{
					Tag: []Tag{},
				},
			},
			artistResponse: artistTagsResponse{
				TopTags: struct {
					Tag  []Tag `json:"tag"`
					Attr struct {
						Artist string `json:"artist"`
					} `json:"@attr"`
				}{
					Tag: []Tag{
						{Name: "pop", URL: "http://last.fm/tag/pop"},
						{Name: "dance", URL: "http://last.fm/tag/dance"},
					},
				},
			},
			wantTags: []Tag{
				{Name: "pop", URL: "http://last.fm/tag/pop"},
				{Name: "dance", URL: "http://last.fm/tag/dance"},
			},
			wantErr: nil,
		},
		{
			name:   "both empty returns empty slice",
			artist: "Unknown Artist",
			track:  "Unknown Track",
			trackResponse: trackTagsResponse{
				TopTags: struct {
					Tag  []Tag `json:"tag"`
					Attr struct {
						Artist string `json:"artist"`
						Track  string `json:"track"`
					} `json:"@attr"`
				}{
					Tag: []Tag{},
				},
			},
			artistResponse: artistTagsResponse{
				TopTags: struct {
					Tag  []Tag `json:"tag"`
					Attr struct {
						Artist string `json:"artist"`
					} `json:"@attr"`
				}{
					Tag: []Tag{},
				},
			},
			wantTags: []Tag{},
			wantErr:  nil,
		},
		{
			name:          "invalid API key",
			artist:        "Test",
			track:         "Test",
			trackResponse: apiError{Error: 10, Message: "Invalid API key"},
			wantTags:      nil,
			wantErr:       ErrInvalidAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method := r.URL.Query().Get("method")

				var resp any
				switch method {
				case "track.getTopTags":
					resp = tt.trackResponse
				case "artist.getTopTags":
					resp = tt.artistResponse
				default:
					t.Fatalf("unexpected method: %s", method)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := &Client{
				apiKey:     "test-api-key",
				httpClient: server.Client(),
				baseURL:    server.URL + "/",
				cache:      make(map[string][]Tag),
			}

			tags, err := client.GetTags(context.Background(), tt.artist, tt.track)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if len(tags) != len(tt.wantTags) {
					t.Errorf("GetTags() got %d tags, want %d", len(tags), len(tt.wantTags))
					return
				}
				for i, tag := range tags {
					if tag.Name != tt.wantTags[i].Name {
						t.Errorf("GetTags() tag[%d].Name = %s, want %s", i, tag.Name, tt.wantTags[i].Name)
					}
				}
			}
		})
	}
}

func TestGetTags_Caching(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		resp := trackTagsResponse{
			TopTags: struct {
				Tag  []Tag `json:"tag"`
				Attr struct {
					Artist string `json:"artist"`
					Track  string `json:"track"`
				} `json:"@attr"`
			}{
				Tag: []Tag{{Name: "rock", Count: 100, URL: "http://last.fm/tag/rock"}},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/",
		cache:      make(map[string][]Tag),
	}

	// First call - should hit server
	tags1, err := client.GetTags(context.Background(), "Artist", "Track")
	if err != nil {
		t.Fatalf("First GetTags() error = %v", err)
	}
	if len(tags1) != 1 {
		t.Fatalf("First GetTags() got %d tags, want 1", len(tags1))
	}

	// Second call - should hit cache
	tags2, err := client.GetTags(context.Background(), "Artist", "Track")
	if err != nil {
		t.Fatalf("Second GetTags() error = %v", err)
	}
	if len(tags2) != 1 {
		t.Fatalf("Second GetTags() got %d tags, want 1", len(tags2))
	}

	// Should only have made one request
	if count := requestCount.Load(); count != 1 {
		t.Errorf("Expected 1 request, got %d", count)
	}
}

func TestGetTags_RateLimitRetry(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		// Fail first 2 requests with rate limit, succeed on 3rd
		if count < 3 {
			resp := apiError{Error: 29, Message: "Rate limit exceeded"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := trackTagsResponse{
			TopTags: struct {
				Tag  []Tag `json:"tag"`
				Attr struct {
					Artist string `json:"artist"`
					Track  string `json:"track"`
				} `json:"@attr"`
			}{
				Tag: []Tag{{Name: "rock", Count: 100, URL: "http://last.fm/tag/rock"}},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/",
		cache:      make(map[string][]Tag),
	}

	// Use short timeout context for faster test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tags, err := client.GetTags(ctx, "Artist", "Track")
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}

	if len(tags) != 1 || tags[0].Name != "rock" {
		t.Errorf("GetTags() got unexpected tags: %v", tags)
	}

	// Should have made 3 requests (2 rate limited + 1 success)
	if count := requestCount.Load(); count != 3 {
		t.Errorf("Expected 3 requests, got %d", count)
	}
}

func TestGetTags_RateLimitExhausted(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		// Always return rate limit error
		resp := apiError{Error: 29, Message: "Rate limit exceeded"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/",
		cache:      make(map[string][]Tag),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.GetTags(ctx, "Artist", "Track")

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("GetTags() error = %v, want ErrRateLimited", err)
	}

	// Should have made 4 requests (1 initial + 3 retries)
	if count := requestCount.Load(); count != 4 {
		t.Errorf("Expected 4 requests, got %d", count)
	}
}

func TestNewClient(t *testing.T) {
	cfg := &Config{APIKey: "test-key"}
	client := NewClient(cfg)

	if client.apiKey != "test-key" {
		t.Errorf("NewClient() apiKey = %s, want test-key", client.apiKey)
	}
	if client.httpClient == nil {
		t.Error("NewClient() httpClient is nil")
	}
	if client.cache == nil {
		t.Error("NewClient() cache is nil")
	}
	if client.baseURL != baseURL {
		t.Errorf("NewClient() baseURL = %s, want %s", client.baseURL, baseURL)
	}
}
