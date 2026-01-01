// Package tags provides a service for fetching Last.fm tags for music tracks.
package tags

import (
	"context"
	"sync"

	"github.com/justestif/go-spotify-era-organizer/internal/lastfm"
)

// TagSource indicates where the tags came from.
type TagSource string

const (
	// SourceTrack means tags came from track.getTopTags.
	SourceTrack TagSource = "track"
	// SourceArtist means tags came from artist.getTopTags (fallback).
	SourceArtist TagSource = "artist"
	// SourceNone means no tags were found.
	SourceNone TagSource = "none"
)

// Default concurrency for batch processing.
const DefaultConcurrency = 5

// Track represents the minimal track info needed for tag lookup.
type Track struct {
	ID     string
	Name   string
	Artist string
}

// TrackTags holds the tags fetched for a track.
type TrackTags struct {
	TrackID string
	Tags    []lastfm.Tag
	Source  TagSource
	Error   error // Non-nil if fetching failed
}

// TagFetcher abstracts the Last.fm client for testing.
type TagFetcher interface {
	GetTags(ctx context.Context, artist, track string) ([]lastfm.Tag, error)
}

// TagService defines the interface for fetching tags for tracks.
type TagService interface {
	FetchTagsForTracks(ctx context.Context, tracks []Track) ([]TrackTags, error)
}

// Service implements TagService using Last.fm as the tag source.
type Service struct {
	fetcher     TagFetcher
	concurrency int
}

// Option configures a Service.
type Option func(*Service)

// WithConcurrency sets the number of concurrent tag fetch operations.
func WithConcurrency(n int) Option {
	return func(s *Service) {
		if n > 0 {
			s.concurrency = n
		}
	}
}

// NewService creates a new tag service.
func NewService(fetcher TagFetcher, opts ...Option) *Service {
	s := &Service{
		fetcher:     fetcher,
		concurrency: DefaultConcurrency,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// FetchTagsForTracks fetches tags for multiple tracks concurrently.
// Results are returned in the same order as input tracks.
// Individual fetch errors are captured in TrackTags.Error rather than failing the batch.
func (s *Service) FetchTagsForTracks(ctx context.Context, tracks []Track) ([]TrackTags, error) {
	if len(tracks) == 0 {
		return []TrackTags{}, nil
	}

	results := make([]TrackTags, len(tracks))

	// Create work channel and semaphore
	type workItem struct {
		index int
		track Track
	}
	workCh := make(chan workItem, len(tracks))

	// Feed work items
	for i, t := range tracks {
		workCh <- workItem{index: i, track: t}
	}
	close(workCh)

	// Process with worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				select {
				case <-ctx.Done():
					results[work.index] = TrackTags{
						TrackID: work.track.ID,
						Tags:    []lastfm.Tag{},
						Source:  SourceNone,
						Error:   ctx.Err(),
					}
					continue
				default:
				}

				tags, err := s.fetcher.GetTags(ctx, work.track.Artist, work.track.Name)
				result := TrackTags{
					TrackID: work.track.ID,
					Tags:    tags,
					Error:   err,
				}

				// Determine source based on tags
				if err != nil {
					result.Source = SourceNone
					result.Tags = []lastfm.Tag{}
				} else if len(tags) == 0 {
					result.Source = SourceNone
				} else {
					// Last.fm client handles track->artist fallback internally,
					// so we can't distinguish here. Default to "track".
					// If caller needs to know, they'd need enhanced client API.
					result.Source = SourceTrack
				}

				results[work.index] = result
			}
		}()
	}

	wg.Wait()

	// Check if context was cancelled
	if ctx.Err() != nil {
		return results, ctx.Err()
	}

	return results, nil
}
