package tags

import (
	"context"
	"fmt"
	"time"

	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/lastfm"
)

// CacheTTL is the duration after which cached tags are considered stale.
const CacheTTL = 30 * 24 * time.Hour // 30 days

// CachedTagFetcher implements TagFetcher with database persistence.
// It checks the database cache first, then falls back to the underlying
// Last.fm client for cache misses, persisting new results.
type CachedTagFetcher struct {
	db     *db.DB
	client *lastfm.Client
}

// NewCachedTagFetcher creates a new CachedTagFetcher that wraps the Last.fm client
// with PostgreSQL persistence.
func NewCachedTagFetcher(database *db.DB, client *lastfm.Client) *CachedTagFetcher {
	return &CachedTagFetcher{
		db:     database,
		client: client,
	}
}

// GetTags fetches tags for a track, using the database cache when available.
// It implements the TagFetcher interface.
func (c *CachedTagFetcher) GetTags(ctx context.Context, artist, track string) ([]lastfm.Tag, error) {
	// For single track lookups, we don't have a track ID, so we go directly to Last.fm
	// This method exists to satisfy the TagFetcher interface
	return c.client.GetTags(ctx, artist, track)
}

// GetTagsForTracks fetches tags for multiple tracks with caching.
// It checks the database cache first, identifies cache misses and stale entries,
// fetches missing/stale tags from Last.fm, and persists the results.
func (c *CachedTagFetcher) GetTagsForTracks(ctx context.Context, tracks []Track) (map[string][]lastfm.Tag, error) {
	if len(tracks) == 0 {
		return make(map[string][]lastfm.Tag), nil
	}

	// Build track ID list and lookup map
	trackIDs := make([]string, len(tracks))
	trackByID := make(map[string]Track, len(tracks))
	for i, t := range tracks {
		trackIDs[i] = t.ID
		trackByID[t.ID] = t
	}

	// Check database cache
	cached, err := c.db.Tags().GetForTracks(ctx, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("getting cached tags: %w", err)
	}

	// Separate valid cached results from stale/missing entries
	result := make(map[string][]lastfm.Tag, len(tracks))
	var needsFetch []Track
	now := time.Now()
	staleThreshold := now.Add(-CacheTTL)

	for _, id := range trackIDs {
		cachedTags, found := cached[id]
		if !found || len(cachedTags) == 0 {
			// Cache miss - need to fetch
			needsFetch = append(needsFetch, trackByID[id])
			continue
		}

		// Check if stale (lazy invalidation)
		if cachedTags[0].FetchedAt.Before(staleThreshold) {
			needsFetch = append(needsFetch, trackByID[id])
			continue
		}

		// Valid cache hit - convert to lastfm.Tag
		result[id] = dbTagsToLastfmTags(cachedTags)
	}

	// Fetch missing/stale tags from Last.fm
	if len(needsFetch) > 0 {
		fetched, err := c.fetchAndPersist(ctx, needsFetch)
		if err != nil {
			// Log error but don't fail - return what we have from cache
			// Individual track errors are handled in fetchAndPersist
			_ = err
		}

		// Merge fetched results
		for id, tags := range fetched {
			result[id] = tags
		}
	}

	return result, nil
}

// fetchAndPersist fetches tags from Last.fm and persists them to the database.
func (c *CachedTagFetcher) fetchAndPersist(ctx context.Context, tracks []Track) (map[string][]lastfm.Tag, error) {
	result := make(map[string][]lastfm.Tag, len(tracks))
	var dbTags []db.TrackTag
	now := time.Now()

	for _, t := range tracks {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		tags, err := c.client.GetTags(ctx, t.Artist, t.Name)
		if err != nil {
			// Skip this track but continue with others
			continue
		}

		result[t.ID] = tags

		// Convert to db.TrackTag for persistence
		source := "track" // Last.fm client handles fallback internally
		for _, tag := range tags {
			dbTags = append(dbTags, db.TrackTag{
				TrackID:   t.ID,
				TagName:   tag.Name,
				TagCount:  tag.Count,
				Source:    source,
				FetchedAt: now,
			})
		}
	}

	// Persist to database
	if len(dbTags) > 0 {
		if err := c.db.Tags().UpsertBatch(ctx, dbTags); err != nil {
			return result, fmt.Errorf("persisting tags: %w", err)
		}
	}

	return result, nil
}

// dbTagsToLastfmTags converts database TrackTag slice to lastfm.Tag slice.
func dbTagsToLastfmTags(dbTags []db.TrackTag) []lastfm.Tag {
	tags := make([]lastfm.Tag, len(dbTags))
	for i, t := range dbTags {
		tags[i] = lastfm.Tag{
			Name:  t.TagName,
			Count: t.TagCount,
		}
	}
	return tags
}
