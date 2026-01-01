package tags

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justestif/go-spotify-era-organizer/internal/lastfm"
)

// mockFetcher implements TagFetcher for testing.
type mockFetcher struct {
	// tags maps "artist:track" to tags
	tags map[string][]lastfm.Tag
	// errors maps "artist:track" to errors
	errors map[string]error
	// callCount tracks number of GetTags calls
	callCount atomic.Int32
	// delay simulates network latency
	delay time.Duration
}

func newMockFetcher() *mockFetcher {
	return &mockFetcher{
		tags:   make(map[string][]lastfm.Tag),
		errors: make(map[string]error),
	}
}

func (m *mockFetcher) addTags(artist, track string, tags []lastfm.Tag) {
	m.tags[artist+":"+track] = tags
}

func (m *mockFetcher) addError(artist, track string, err error) {
	m.errors[artist+":"+track] = err
}

func (m *mockFetcher) GetTags(ctx context.Context, artist, track string) ([]lastfm.Tag, error) {
	m.callCount.Add(1)

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	key := artist + ":" + track
	if err, ok := m.errors[key]; ok {
		return nil, err
	}
	if tags, ok := m.tags[key]; ok {
		return tags, nil
	}
	return []lastfm.Tag{}, nil
}

func TestFetchTagsForTracks_Empty(t *testing.T) {
	fetcher := newMockFetcher()
	svc := NewService(fetcher)

	results, err := svc.FetchTagsForTracks(context.Background(), []Track{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestFetchTagsForTracks_SingleTrack(t *testing.T) {
	fetcher := newMockFetcher()
	fetcher.addTags("Radiohead", "Creep", []lastfm.Tag{
		{Name: "rock", Count: 100},
		{Name: "alternative", Count: 80},
	})

	svc := NewService(fetcher)
	tracks := []Track{
		{ID: "track1", Name: "Creep", Artist: "Radiohead"},
	}

	results, err := svc.FetchTagsForTracks(context.Background(), tracks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.TrackID != "track1" {
		t.Errorf("expected TrackID 'track1', got %q", r.TrackID)
	}
	if len(r.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(r.Tags))
	}
	if r.Source != SourceTrack {
		t.Errorf("expected source 'track', got %q", r.Source)
	}
	if r.Error != nil {
		t.Errorf("unexpected error: %v", r.Error)
	}
}

func TestFetchTagsForTracks_NoTags(t *testing.T) {
	fetcher := newMockFetcher()
	// No tags configured for this track

	svc := NewService(fetcher)
	tracks := []Track{
		{ID: "track1", Name: "Unknown Song", Artist: "Unknown Artist"},
	}

	results, err := svc.FetchTagsForTracks(context.Background(), tracks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := results[0]
	if r.Source != SourceNone {
		t.Errorf("expected source 'none', got %q", r.Source)
	}
	if len(r.Tags) != 0 {
		t.Errorf("expected empty tags, got %d", len(r.Tags))
	}
}

func TestFetchTagsForTracks_MultipleTracks(t *testing.T) {
	fetcher := newMockFetcher()
	fetcher.addTags("Radiohead", "Creep", []lastfm.Tag{{Name: "rock", Count: 100}})
	fetcher.addTags("Daft Punk", "Get Lucky", []lastfm.Tag{{Name: "electronic", Count: 90}})
	fetcher.addTags("Adele", "Hello", []lastfm.Tag{{Name: "pop", Count: 85}})

	svc := NewService(fetcher, WithConcurrency(2))
	tracks := []Track{
		{ID: "t1", Name: "Creep", Artist: "Radiohead"},
		{ID: "t2", Name: "Get Lucky", Artist: "Daft Punk"},
		{ID: "t3", Name: "Hello", Artist: "Adele"},
	}

	results, err := svc.FetchTagsForTracks(context.Background(), tracks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order is preserved
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	expected := []struct {
		id      string
		tagName string
	}{
		{"t1", "rock"},
		{"t2", "electronic"},
		{"t3", "pop"},
	}

	for i, exp := range expected {
		if results[i].TrackID != exp.id {
			t.Errorf("result[%d]: expected ID %q, got %q", i, exp.id, results[i].TrackID)
		}
		if len(results[i].Tags) == 0 || results[i].Tags[0].Name != exp.tagName {
			t.Errorf("result[%d]: expected tag %q, got %v", i, exp.tagName, results[i].Tags)
		}
	}
}

func TestFetchTagsForTracks_IndividualErrors(t *testing.T) {
	fetcher := newMockFetcher()
	fetcher.addTags("Good Artist", "Good Track", []lastfm.Tag{{Name: "rock", Count: 100}})
	fetcher.addError("Bad Artist", "Bad Track", errors.New("API error"))

	svc := NewService(fetcher)
	tracks := []Track{
		{ID: "t1", Name: "Good Track", Artist: "Good Artist"},
		{ID: "t2", Name: "Bad Track", Artist: "Bad Artist"},
	}

	results, err := svc.FetchTagsForTracks(context.Background(), tracks)
	// Batch should not fail even if individual tracks fail
	if err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First track should succeed
	if results[0].Error != nil {
		t.Errorf("expected no error for t1, got %v", results[0].Error)
	}
	if len(results[0].Tags) != 1 {
		t.Errorf("expected 1 tag for t1, got %d", len(results[0].Tags))
	}

	// Second track should have error
	if results[1].Error == nil {
		t.Error("expected error for t2, got nil")
	}
	if results[1].Source != SourceNone {
		t.Errorf("expected source 'none' for failed track, got %q", results[1].Source)
	}
	if len(results[1].Tags) != 0 {
		t.Errorf("expected empty tags for failed track, got %d", len(results[1].Tags))
	}
}

func TestFetchTagsForTracks_ContextCancellation(t *testing.T) {
	fetcher := newMockFetcher()
	fetcher.delay = 100 * time.Millisecond

	// Add tags for many tracks to ensure workers are busy
	for i := 0; i < 10; i++ {
		fetcher.addTags("Artist", "Track", []lastfm.Tag{{Name: "rock", Count: 100}})
	}

	svc := NewService(fetcher, WithConcurrency(2))

	tracks := make([]Track, 10)
	for i := range tracks {
		tracks[i] = Track{ID: "t", Name: "Track", Artist: "Artist"}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	results, err := svc.FetchTagsForTracks(ctx, tracks)

	// Should return context error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Results should still be returned
	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}
}

func TestFetchTagsForTracks_Concurrency(t *testing.T) {
	fetcher := newMockFetcher()
	fetcher.delay = 10 * time.Millisecond

	for i := 0; i < 20; i++ {
		fetcher.addTags("Artist", "Track", []lastfm.Tag{{Name: "rock", Count: 100}})
	}

	tracks := make([]Track, 20)
	for i := range tracks {
		tracks[i] = Track{ID: "t", Name: "Track", Artist: "Artist"}
	}

	// Test with high concurrency
	svc := NewService(fetcher, WithConcurrency(10))

	start := time.Now()
	_, err := svc.FetchTagsForTracks(context.Background(), tracks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// With 10 concurrent workers and 20 tracks at 10ms each,
	// should complete in roughly 20-30ms (2 batches)
	// Sequential would take 200ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected concurrent execution, took %v", elapsed)
	}

	// Verify all tracks were processed
	if fetcher.callCount.Load() != 20 {
		t.Errorf("expected 20 calls, got %d", fetcher.callCount.Load())
	}
}

func TestWithConcurrency(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive value", 10, 10},
		{"zero uses default", 0, DefaultConcurrency},
		{"negative uses default", -1, DefaultConcurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := newMockFetcher()
			svc := NewService(fetcher, WithConcurrency(tt.input))

			if svc.concurrency != tt.expected {
				t.Errorf("expected concurrency %d, got %d", tt.expected, svc.concurrency)
			}
		})
	}
}

func TestNewService_DefaultConcurrency(t *testing.T) {
	fetcher := newMockFetcher()
	svc := NewService(fetcher)

	if svc.concurrency != DefaultConcurrency {
		t.Errorf("expected default concurrency %d, got %d", DefaultConcurrency, svc.concurrency)
	}
}
