package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lr2/internal/models"
)

var ErrSessionNotFound = errors.New("session not found or expired")
const sessionTimeLayout = time.RFC3339

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt.UTC().Format(sessionTimeLayout))
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetUser(ctx context.Context, id string) (models.User, error) {
	var (
		u          models.User
		expiresStr string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`, id,
	).Scan(&u.ID, &u.Email, &expiresStr)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrSessionNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("get session: %w", err)
	}

	expires, err := time.Parse(sessionTimeLayout, expiresStr)
	if err != nil {
		return models.User{}, fmt.Errorf("parse session expiry %q: %w", expiresStr, err)
	}
	if !time.Now().Before(expires) {
		_ = r.Delete(ctx, id)
		return models.User{}, ErrSessionNotFound
	}
	return u, nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().UTC().Format(sessionTimeLayout))
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
