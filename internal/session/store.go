// Package session provides PostgreSQL-backed session management
// replacing PocketBase's JWT auth.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// SessionDuration is how long a session lasts before expiring.
	SessionDuration = 30 * 24 * time.Hour // 30 days

	// tokenLength is the number of random bytes for session tokens.
	tokenLength = 32
)

// SessionUser holds the user data loaded alongside a session.
type SessionUser struct {
	ID       string
	Email    string
	Username string
	Avatar   string
	Points   int
	IsAdmin  bool
	Verified bool
}

// Session represents an active user session.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Created   time.Time
	User      *SessionUser
}

// Store manages sessions in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new session store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// generateToken creates a cryptographically random session token.
func generateToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create creates a new session for the given user and returns the session ID (token).
func (s *Store) Create(ctx context.Context, userID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(SessionDuration)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	return token, nil
}

// Validate looks up a session by token and returns the associated user data.
// Returns nil if the session is expired or not found.
func (s *Store) Validate(ctx context.Context, token string) (*Session, error) {
	var sess Session
	var user SessionUser

	err := s.pool.QueryRow(ctx,
		`SELECT s.id, s.user_id, s.expires_at, s.created,
		        u.id, u.email, u.username, u.avatar, u.points, u.is_admin, u.verified
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id AND u.deleted IS NULL
		 WHERE s.id = $1 AND s.expires_at > NOW()`,
		token,
	).Scan(
		&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.Created,
		&user.ID, &user.Email, &user.Username, &user.Avatar, &user.Points, &user.IsAdmin, &user.Verified,
	)
	if err != nil {
		return nil, nil // not found or expired -- not an error
	}

	sess.User = &user
	return &sess, nil
}

// Delete removes a session by token (logout).
func (s *Store) Delete(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, token)
	return err
}

// DeleteUserSessions removes all sessions for a user (e.g., password change).
func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// Cleanup removes all expired sessions. Should be called periodically.
func (s *Store) Cleanup(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}
