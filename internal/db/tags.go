package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TagRepository handles track tag database operations.
type TagRepository struct {
	pool *pgxpool.Pool
}

// UpsertBatch inserts or updates multiple tags efficiently.
func (r *TagRepository) UpsertBatch(ctx context.Context, tags []TrackTag) error {
	if len(tags) == 0 {
		return nil
	}

	query := `
		INSERT INTO track_tags (track_id, tag_name, tag_count, source, fetched_at)
		SELECT * FROM unnest($1::text[], $2::text[], $3::int[], $4::text[], $5::timestamptz[])
		ON CONFLICT (track_id, tag_name) DO UPDATE SET
			tag_count = EXCLUDED.tag_count,
			source = EXCLUDED.source,
			fetched_at = EXCLUDED.fetched_at
	`

	trackIDs := make([]string, len(tags))
	tagNames := make([]string, len(tags))
	tagCounts := make([]int, len(tags))
	sources := make([]string, len(tags))
	fetchedAts := make([]time.Time, len(tags))

	for i, t := range tags {
		trackIDs[i] = t.TrackID
		tagNames[i] = t.TagName
		tagCounts[i] = t.TagCount
		sources[i] = t.Source
		fetchedAts[i] = t.FetchedAt
	}

	_, err := r.pool.Exec(ctx, query, trackIDs, tagNames, tagCounts, sources, fetchedAts)
	if err != nil {
		return fmt.Errorf("batch upserting tags: %w", err)
	}
	return nil
}

// GetForTrack retrieves all tags for a track.
func (r *TagRepository) GetForTrack(ctx context.Context, trackID string) ([]TrackTag, error) {
	query := `
		SELECT track_id, tag_name, tag_count, source, fetched_at
		FROM track_tags
		WHERE track_id = $1
		ORDER BY tag_count DESC
	`
	rows, err := r.pool.Query(ctx, query, trackID)
	if err != nil {
		return nil, fmt.Errorf("querying track tags: %w", err)
	}
	defer rows.Close()

	var tags []TrackTag
	for rows.Next() {
		var tag TrackTag
		if err := rows.Scan(
			&tag.TrackID,
			&tag.TagName,
			&tag.TagCount,
			&tag.Source,
			&tag.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// GetForTracks retrieves tags for multiple tracks, returning a map of track ID to tags.
func (r *TagRepository) GetForTracks(ctx context.Context, trackIDs []string) (map[string][]TrackTag, error) {
	if len(trackIDs) == 0 {
		return make(map[string][]TrackTag), nil
	}

	query := `
		SELECT track_id, tag_name, tag_count, source, fetched_at
		FROM track_tags
		WHERE track_id = ANY($1)
		ORDER BY track_id, tag_count DESC
	`
	rows, err := r.pool.Query(ctx, query, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("querying track tags: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]TrackTag)
	for rows.Next() {
		var tag TrackTag
		if err := rows.Scan(
			&tag.TrackID,
			&tag.TagName,
			&tag.TagCount,
			&tag.Source,
			&tag.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		result[tag.TrackID] = append(result[tag.TrackID], tag)
	}
	return result, rows.Err()
}

// GetStale returns track IDs with tags older than the given time.
func (r *TagRepository) GetStale(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT track_id
		FROM track_tags
		WHERE fetched_at < $1
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, olderThan, limit)
	if err != nil {
		return nil, fmt.Errorf("querying stale tags: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning track ID: %w", err)
		}
		trackIDs = append(trackIDs, id)
	}
	return trackIDs, rows.Err()
}

// GetTracksWithoutTags returns track IDs that have no tags.
func (r *TagRepository) GetTracksWithoutTags(ctx context.Context, trackIDs []string) ([]string, error) {
	if len(trackIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT id
		FROM unnest($1::text[]) AS id
		WHERE id NOT IN (SELECT DISTINCT track_id FROM track_tags)
	`
	rows, err := r.pool.Query(ctx, query, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("querying tracks without tags: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning track ID: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

// DeleteForTrack removes all tags for a track.
func (r *TagRepository) DeleteForTrack(ctx context.Context, trackID string) error {
	query := `DELETE FROM track_tags WHERE track_id = $1`
	_, err := r.pool.Exec(ctx, query, trackID)
	if err != nil {
		return fmt.Errorf("deleting track tags: %w", err)
	}
	return nil
}
