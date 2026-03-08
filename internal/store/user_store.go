package store

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tekgeek88/ironlytic-backend/internal/domain"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, email, passwordHash string, displayName *string) (*domain.User, error) {
	var u domain.User

	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, display_name, created_at, updated_at, last_login_at
	`, email, passwordHash, displayName).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User

	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, created_at, updated_at, last_login_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User

	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (s *UserStore) TouchLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET last_login_at = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, id, at)
	return err
}
