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

type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, userAgent, ip *string) (*domain.Session, error) {
	var sess domain.Session

	err := s.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token_hash, created_at, updated_at, expires_at, last_seen_at, revoked_at, user_agent, ip_address::text
	`, userID, tokenHash, expiresAt, userAgent, ip).Scan(
		&sess.ID,
		&sess.UserID,
		&sess.TokenHash,
		&sess.CreatedAt,
		&sess.UpdatedAt,
		&sess.ExpiresAt,
		&sess.LastSeenAt,
		&sess.RevokedAt,
		&sess.UserAgent,
		&sess.IPAddress,
	)
	if err != nil {
		return nil, err
	}

	return &sess, nil
}

func (s *SessionStore) GetActiveByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var sess domain.Session

	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, created_at, updated_at, expires_at, last_seen_at, revoked_at, user_agent, ip_address::text
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tokenHash).Scan(
		&sess.ID,
		&sess.UserID,
		&sess.TokenHash,
		&sess.CreatedAt,
		&sess.UpdatedAt,
		&sess.ExpiresAt,
		&sess.LastSeenAt,
		&sess.RevokedAt,
		&sess.UserAgent,
		&sess.IPAddress,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &sess, nil
}

func (s *SessionStore) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = NOW(),
		    updated_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (s *SessionStore) TouchLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE sessions
		SET last_seen_at = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, id, at)
	return err
}
