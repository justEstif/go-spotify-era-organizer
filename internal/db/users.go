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
		INSERT INTO users (id, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`
	now := time.Now()
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.DisplayName,
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
		SELECT id, display_name, created_at, updated_at, last_sync_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.DisplayName,
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
		INSERT INTO users (id, display_name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.DisplayName,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting user: %w", err)
	}
	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// RecentlyActive returns users who have synced since the given time.
func (r *UserRepository) RecentlyActive(ctx context.Context, since time.Time) ([]User, error) {
	query := `
		SELECT id, display_name, created_at, updated_at, last_sync_at
		FROM users
		WHERE last_sync_at >= $1
		ORDER BY last_sync_at DESC
	`
	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("querying recently active users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt, &u.LastSyncAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
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
