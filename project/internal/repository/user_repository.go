package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lr2/internal/models"
)

var (
	ErrEmailTaken   = errors.New("email already taken")
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (models.User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES (?, ?)`, email, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.User{}, fmt.Errorf("create user: get id: %w", err)
	}
	return models.User{ID: id, Email: email, PasswordHash: passwordHash}, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}
