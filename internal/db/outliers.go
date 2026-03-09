package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutlierRepository handles outlier track database operations.
type OutlierRepository struct {
	pool *pgxpool.Pool
}

// SaveForUser replaces all outlier tracks for a user with the given track IDs.
func (r *OutlierRepository) SaveForUser(ctx context.Context, userID string, trackIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing outliers
	_, err = tx.Exec(ctx, `DELETE FROM outlier_tracks WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("deleting existing outliers: %w", err)
	}

	// Insert new outliers
	if len(trackIDs) > 0 {
		_, err = tx.Exec(ctx,
			`INSERT INTO outlier_tracks (user_id, track_id) SELECT $1, unnest($2::text[])`,
			userID, trackIDs,
		)
		if err != nil {
			return fmt.Errorf("inserting outliers: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// GetForUser retrieves all outlier tracks for a user with full track data.
func (r *OutlierRepository) GetForUser(ctx context.Context, userID string) ([]Track, error) {
	query := `
		SELECT t.id, t.name, t.artist, t.album, t.album_id, t.duration_ms, t.created_at
		FROM tracks t
		JOIN outlier_tracks ot ON t.id = ot.track_id
		WHERE ot.user_id = $1
		ORDER BY t.name
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying outlier tracks: %w", err)
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		if err := rows.Scan(
			&track.ID,
			&track.Name,
			&track.Artist,
			&track.Album,
			&track.AlbumID,
			&track.DurationMs,
			&track.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning outlier track: %w", err)
		}
		tracks = append(tracks, track)
	}
	return tracks, rows.Err()
}

// DeleteForUser removes all outlier tracks for a user.
func (r *OutlierRepository) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM outlier_tracks WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("deleting outliers: %w", err)
	}
	return nil
}

// Count returns the number of outlier tracks for a user.
func (r *OutlierRepository) Count(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outlier_tracks WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting outliers: %w", err)
	}
	return count, nil
}

// AssignToEra removes a track from outliers and adds it to an era, atomically.
func (r *OutlierRepository) AssignToEra(ctx context.Context, userID string, trackID string, eraID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify the outlier exists for this user
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM outlier_tracks WHERE user_id = $1 AND track_id = $2)`,
		userID, trackID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking outlier existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	// Verify the era belongs to this user
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM eras WHERE id = $1 AND user_id = $2)`,
		eraID, userID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking era ownership: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	// Remove from outliers
	_, err = tx.Exec(ctx,
		`DELETE FROM outlier_tracks WHERE user_id = $1 AND track_id = $2`,
		userID, trackID,
	)
	if err != nil {
		return fmt.Errorf("removing from outliers: %w", err)
	}

	// Add to era_tracks
	_, err = tx.Exec(ctx,
		`INSERT INTO era_tracks (era_id, track_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		eraID, trackID,
	)
	if err != nil {
		return fmt.Errorf("adding to era tracks: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
