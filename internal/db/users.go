package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles user database operations.
type UserRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, display_name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	now := time.Now()
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.DisplayName,
		user.Email,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

// Get retrieves a user by ID.
func (r *UserRepository) Get(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, display_name, email, created_at, updated_at, last_sync_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.DisplayName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSyncAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return &user, nil
}

// Upsert creates or updates a user.
func (r *UserRepository) Upsert(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, display_name, email, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			email = EXCLUDED.email,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.DisplayName,
		user.Email,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting user: %w", err)
	}
	return nil
}

// UpdateLastSync updates the last sync timestamp for a user.
func (r *UserRepository) UpdateLastSync(ctx context.Context, id string, syncTime time.Time) error {
	query := `
		UPDATE users
		SET last_sync_at = $2, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, syncTime)
	if err != nil {
		return fmt.Errorf("updating last sync: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
