package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TrackRepository handles track database operations.
type TrackRepository struct {
	pool *pgxpool.Pool
}

// Upsert creates or updates a track.
func (r *TrackRepository) Upsert(ctx context.Context, track *Track) error {
	query := `
		INSERT INTO tracks (id, name, artist, album, album_id, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			artist = EXCLUDED.artist,
			album = EXCLUDED.album,
			album_id = EXCLUDED.album_id,
			duration_ms = EXCLUDED.duration_ms
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query,
		track.ID,
		track.Name,
		track.Artist,
		track.Album,
		track.AlbumID,
		track.DurationMs,
	).Scan(&track.CreatedAt)
	if err != nil {
		return fmt.Errorf("upserting track: %w", err)
	}
	return nil
}

// UpsertBatch inserts or updates multiple tracks efficiently.
func (r *TrackRepository) UpsertBatch(ctx context.Context, tracks []Track) error {
	if len(tracks) == 0 {
		return nil
	}

	query := `
		INSERT INTO tracks (id, name, artist, album, album_id, duration_ms, created_at)
		SELECT * FROM unnest($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::int[], $7::timestamptz[])
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			artist = EXCLUDED.artist,
			album = EXCLUDED.album,
			album_id = EXCLUDED.album_id,
			duration_ms = EXCLUDED.duration_ms
	`

	ids := make([]string, len(tracks))
	names := make([]string, len(tracks))
	artists := make([]string, len(tracks))
	albums := make([]*string, len(tracks))
	albumIDs := make([]*string, len(tracks))
	durations := make([]*int, len(tracks))
	createdAts := make([]time.Time, len(tracks))

	now := time.Now()
	for i, t := range tracks {
		ids[i] = t.ID
		names[i] = t.Name
		artists[i] = t.Artist
		albums[i] = t.Album
		albumIDs[i] = t.AlbumID
		durations[i] = t.DurationMs
		createdAts[i] = now
	}

	_, err := r.pool.Exec(ctx, query, ids, names, artists, albums, albumIDs, durations, createdAts)
	if err != nil {
		return fmt.Errorf("batch upserting tracks: %w", err)
	}
	return nil
}

// Get retrieves a track by ID.
func (r *TrackRepository) Get(ctx context.Context, id string) (*Track, error) {
	query := `
		SELECT id, name, artist, album, album_id, duration_ms, created_at
		FROM tracks
		WHERE id = $1
	`
	var track Track
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&track.ID,
		&track.Name,
		&track.Artist,
		&track.Album,
		&track.AlbumID,
		&track.DurationMs,
		&track.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying track: %w", err)
	}
	return &track, nil
}

// GetUserTracks retrieves all liked tracks for a user, ordered by added_at desc.
func (r *TrackRepository) GetUserTracks(ctx context.Context, userID string) ([]Track, error) {
	query := `
		SELECT t.id, t.name, t.artist, t.album, t.album_id, t.duration_ms, t.created_at
		FROM tracks t
		JOIN user_tracks ut ON t.id = ut.track_id
		WHERE ut.user_id = $1
		ORDER BY ut.added_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user tracks: %w", err)
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

// GetUserTracksWithAddedAt retrieves all liked tracks for a user with their added_at timestamps.
func (r *TrackRepository) GetUserTracksWithAddedAt(ctx context.Context, userID string) ([]UserTrack, []Track, error) {
	query := `
		SELECT t.id, t.name, t.artist, t.album, t.album_id, t.duration_ms, t.created_at, ut.added_at
		FROM tracks t
		JOIN user_tracks ut ON t.id = ut.track_id
		WHERE ut.user_id = $1
		ORDER BY ut.added_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("querying user tracks: %w", err)
	}
	defer rows.Close()

	var userTracks []UserTrack
	var tracks []Track
	for rows.Next() {
		var track Track
		var addedAt time.Time
		if err := rows.Scan(
			&track.ID,
			&track.Name,
			&track.Artist,
			&track.Album,
			&track.AlbumID,
			&track.DurationMs,
			&track.CreatedAt,
			&addedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scanning track: %w", err)
		}
		tracks = append(tracks, track)
		userTracks = append(userTracks, UserTrack{
			UserID:  userID,
			TrackID: track.ID,
			AddedAt: addedAt,
		})
	}
	return userTracks, tracks, rows.Err()
}

// LinkToUser links a track to a user's library.
func (r *TrackRepository) LinkToUser(ctx context.Context, userID, trackID string, addedAt time.Time) error {
	query := `
		INSERT INTO user_tracks (user_id, track_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, track_id) DO UPDATE SET added_at = EXCLUDED.added_at
	`
	_, err := r.pool.Exec(ctx, query, userID, trackID, addedAt)
	if err != nil {
		return fmt.Errorf("linking track to user: %w", err)
	}
	return nil
}

// LinkBatchToUser links multiple tracks to a user's library efficiently.
func (r *TrackRepository) LinkBatchToUser(ctx context.Context, userID string, tracks []UserTrack) error {
	if len(tracks) == 0 {
		return nil
	}

	query := `
		INSERT INTO user_tracks (user_id, track_id, added_at)
		SELECT $1, * FROM unnest($2::text[], $3::timestamptz[])
		ON CONFLICT (user_id, track_id) DO UPDATE SET added_at = EXCLUDED.added_at
	`

	trackIDs := make([]string, len(tracks))
	addedAts := make([]time.Time, len(tracks))

	for i, t := range tracks {
		trackIDs[i] = t.TrackID
		addedAts[i] = t.AddedAt
	}

	_, err := r.pool.Exec(ctx, query, userID, trackIDs, addedAts)
	if err != nil {
		return fmt.Errorf("batch linking tracks to user: %w", err)
	}
	return nil
}

// UnlinkAllFromUser removes all tracks from a user's library.
func (r *TrackRepository) UnlinkAllFromUser(ctx context.Context, userID string) error {
	query := `DELETE FROM user_tracks WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("unlinking all tracks from user: %w", err)
	}
	return nil
}
