// Package db provides PostgreSQL database access for the Spotify Era Organizer.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Common errors.
var (
	ErrNotFound = errors.New("not found")
)

// DB wraps a PostgreSQL connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new database connection pool.
func New(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying connection pool for advanced operations.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Users returns a UserRepository.
func (db *DB) Users() *UserRepository {
	return &UserRepository{pool: db.pool}
}

// Sessions returns a SessionRepository.
func (db *DB) Sessions() *SessionRepository {
	return &SessionRepository{pool: db.pool}
}

// Tracks returns a TrackRepository.
func (db *DB) Tracks() *TrackRepository {
	return &TrackRepository{pool: db.pool}
}

// Tags returns a TagRepository.
func (db *DB) Tags() *TagRepository {
	return &TagRepository{pool: db.pool}
}

// Eras returns an EraRepository.
func (db *DB) Eras() *EraRepository {
	return &EraRepository{pool: db.pool}
}
