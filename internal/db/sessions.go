package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionRepository handles session database operations.
type SessionRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new session.
func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO sessions (id, user_id, access_token, refresh_token, token_expiry, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.AccessToken,
		session.RefreshToken,
		session.TokenExpiry,
		session.CreatedAt,
		session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("inserting session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID.
func (r *SessionRepository) Get(ctx context.Context, id string) (*Session, error) {
	query := `
		SELECT id, user_id, access_token, refresh_token, token_expiry, created_at, expires_at
		FROM sessions
		WHERE id = $1 AND expires_at > NOW()
	`
	var session Session
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.AccessToken,
		&session.RefreshToken,
		&session.TokenExpiry,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying session: %w", err)
	}
	return &session, nil
}

// Delete removes a session by ID.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// UpdateToken updates the OAuth tokens for a session.
func (r *SessionRepository) UpdateToken(ctx context.Context, id, accessToken, refreshToken string, expiry time.Time) error {
	query := `
		UPDATE sessions
		SET access_token = $2, refresh_token = $3, token_expiry = $4
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, accessToken, refreshToken, expiry)
	if err != nil {
		return fmt.Errorf("updating session token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteExpired removes all expired sessions.
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("deleting expired sessions: %w", err)
	}
	return result.RowsAffected(), nil
}

// DeleteForUser removes all sessions for a user.
func (r *SessionRepository) DeleteForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("deleting user sessions: %w", err)
	}
	return nil
}
