package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EraRepository handles era database operations.
type EraRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new era with its associated tracks.
func (r *EraRepository) Create(ctx context.Context, era *Era, trackIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert era
	eraQuery := `
		INSERT INTO eras (id, user_id, name, top_tags, start_date, end_date, playlist_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at
	`
	if era.ID == uuid.Nil {
		era.ID = uuid.New()
	}
	err = tx.QueryRow(ctx, eraQuery,
		era.ID,
		era.UserID,
		era.Name,
		era.TopTags,
		era.StartDate,
		era.EndDate,
		era.PlaylistID,
	).Scan(&era.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting era: %w", err)
	}

	// Insert era_tracks
	if len(trackIDs) > 0 {
		tracksQuery := `
			INSERT INTO era_tracks (era_id, track_id)
			SELECT $1, unnest($2::text[])
		`
		_, err = tx.Exec(ctx, tracksQuery, era.ID, trackIDs)
		if err != nil {
			return fmt.Errorf("inserting era tracks: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// Get retrieves an era by ID.
func (r *EraRepository) Get(ctx context.Context, id uuid.UUID) (*Era, error) {
	query := `
		SELECT id, user_id, name, top_tags, start_date, end_date, playlist_id, created_at
		FROM eras
		WHERE id = $1
	`
	var era Era
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&era.ID,
		&era.UserID,
		&era.Name,
		&era.TopTags,
		&era.StartDate,
		&era.EndDate,
		&era.PlaylistID,
		&era.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying era: %w", err)
	}
	return &era, nil
}

// GetForUser retrieves all eras for a user, ordered by start date desc.
func (r *EraRepository) GetForUser(ctx context.Context, userID string) ([]Era, error) {
	query := `
		SELECT id, user_id, name, top_tags, start_date, end_date, playlist_id, created_at
		FROM eras
		WHERE user_id = $1
		ORDER BY start_date DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user eras: %w", err)
	}
	defer rows.Close()

	var eras []Era
	for rows.Next() {
		var era Era
		if err := rows.Scan(
			&era.ID,
			&era.UserID,
			&era.Name,
			&era.TopTags,
			&era.StartDate,
			&era.EndDate,
			&era.PlaylistID,
			&era.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning era: %w", err)
		}
		eras = append(eras, era)
	}
	return eras, rows.Err()
}

// GetTracks retrieves all tracks for an era.
func (r *EraRepository) GetTracks(ctx context.Context, eraID uuid.UUID) ([]Track, error) {
	query := `
		SELECT t.id, t.name, t.artist, t.album, t.album_id, t.duration_ms, t.created_at
		FROM tracks t
		JOIN era_tracks et ON t.id = et.track_id
		WHERE et.era_id = $1
	`
	rows, err := r.pool.Query(ctx, query, eraID)
	if err != nil {
		return nil, fmt.Errorf("querying era tracks: %w", err)
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
			return nil, fmt.Errorf("scanning track: %w", err)
		}
		tracks = append(tracks, track)
	}
	return tracks, rows.Err()
}

// GetTrackCount returns the number of tracks in an era.
func (r *EraRepository) GetTrackCount(ctx context.Context, eraID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM era_tracks WHERE era_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, eraID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting era tracks: %w", err)
	}
	return count, nil
}

// UpdatePlaylistID sets the Spotify playlist ID for an era.
func (r *EraRepository) UpdatePlaylistID(ctx context.Context, eraID uuid.UUID, playlistID string) error {
	query := `UPDATE eras SET playlist_id = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, eraID, playlistID)
	if err != nil {
		return fmt.Errorf("updating playlist ID: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteForUser removes all eras for a user.
func (r *EraRepository) DeleteForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM eras WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("deleting user eras: %w", err)
	}
	return nil
}

// Delete removes an era by ID.
func (r *EraRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM eras WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting era: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
