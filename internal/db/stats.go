package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepository handles statistics database queries.
type StatsRepository struct {
	pool *pgxpool.Pool
}

// TopTag represents a tag with its track count.
type TopTag struct {
	Name  string
	Count int
}

// ArtistCount represents an artist with their track count.
type ArtistCount struct {
	Name  string
	Count int
}

// MonthCount represents tracks added in a given month.
type MonthCount struct {
	Month string // "2024-01"
	Count int
}

// GetTopTags returns the most common tags for a user's tracks.
func (r *StatsRepository) GetTopTags(ctx context.Context, userID string, limit int) ([]TopTag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tt.tag_name, COUNT(DISTINCT tt.track_id) as count
		 FROM track_tags tt
		 JOIN user_tracks ut ON tt.track_id = ut.track_id
		 WHERE ut.user_id = $1
		 GROUP BY tt.tag_name
		 ORDER BY count DESC
		 LIMIT $2`,
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top tags: %w", err)
	}
	defer rows.Close()

	var tags []TopTag
	for rows.Next() {
		var t TopTag
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, fmt.Errorf("scanning top tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetTopArtists returns the most common artists for a user's tracks.
func (r *StatsRepository) GetTopArtists(ctx context.Context, userID string, limit int) ([]ArtistCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.artist, COUNT(*) as count
		 FROM tracks t
		 JOIN user_tracks ut ON t.id = ut.track_id
		 WHERE ut.user_id = $1
		 GROUP BY t.artist
		 ORDER BY count DESC
		 LIMIT $2`,
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top artists: %w", err)
	}
	defer rows.Close()

	var artists []ArtistCount
	for rows.Next() {
		var a ArtistCount
		if err := rows.Scan(&a.Name, &a.Count); err != nil {
			return nil, fmt.Errorf("scanning top artist: %w", err)
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

// GetTracksPerMonth returns the number of tracks added per month.
func (r *StatsRepository) GetTracksPerMonth(ctx context.Context, userID string) ([]MonthCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT TO_CHAR(ut.added_at, 'YYYY-MM') as month, COUNT(*) as count
		 FROM user_tracks ut
		 WHERE ut.user_id = $1
		 GROUP BY month
		 ORDER BY month`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying tracks per month: %w", err)
	}
	defer rows.Close()

	var months []MonthCount
	for rows.Next() {
		var m MonthCount
		if err := rows.Scan(&m.Month, &m.Count); err != nil {
			return nil, fmt.Errorf("scanning month count: %w", err)
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

// GetTotalTracks returns the total number of tracks for a user.
func (r *StatsRepository) GetTotalTracks(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_tracks WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("querying total tracks: %w", err)
	}
	return count, nil
}

// GetTotalArtists returns the total number of distinct artists for a user.
func (r *StatsRepository) GetTotalArtists(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT t.artist)
		 FROM tracks t
		 JOIN user_tracks ut ON t.id = ut.track_id
		 WHERE ut.user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("querying total artists: %w", err)
	}
	return count, nil
}

// GetTotalEras returns the total number of eras for a user.
func (r *StatsRepository) GetTotalEras(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM eras WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("querying total eras: %w", err)
	}
	return count, nil
}

// GetDateRange returns the earliest and latest added_at timestamps for a user's tracks.
func (r *StatsRepository) GetDateRange(ctx context.Context, userID string) (*time.Time, *time.Time, error) {
	var minTime, maxTime *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT MIN(added_at), MAX(added_at) FROM user_tracks WHERE user_id = $1`,
		userID).Scan(&minTime, &maxTime)
	if err != nil {
		return nil, nil, fmt.Errorf("querying date range: %w", err)
	}
	return minTime, maxTime, nil
}
