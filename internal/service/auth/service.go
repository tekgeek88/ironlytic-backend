package auth

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/tekgeek88/ironlytic-backend/internal/domain"
	"github.com/tekgeek88/ironlytic-backend/internal/repository"
)

type Service struct {
	users    *repository.UserStore
	sessions *repository.SessionStore
	cfg      SessionConfig
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName *string
	UserAgent   *string
	IP          *string
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent *string
	IP        *string
}

type AuthResult struct {
	User      *domain.User
	RawToken  string
	ExpiresAt time.Time
	Session   *domain.Session
}

func NewService(users *repository.UserStore, sessions *repository.SessionStore, cfg SessionConfig) *Service {
	return &Service{
		users: users, sessions: sessions, cfg: cfg,
	}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))

	ph, err := HashPassword(in.Password, DefaultArgonParams)
	if err != nil {
		return nil, err
	}

	u, err := s.users.Create(ctx, email, ph, in.DisplayName)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	return s.createSession(ctx, u.ID, in.UserAgent, in.IP)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(in.Password, u.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	_ = s.users.TouchLastLogin(ctx, u.ID, time.Now())

	return s.createSession(ctx, u.ID, in.UserAgent, in.IP)
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	tokenHash := HashSessionToken(rawToken)
	return s.sessions.RevokeByTokenHash(ctx, tokenHash)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*domain.User, *domain.Session, error) {
	tokenHash := HashSessionToken(rawToken)

	sess, err := s.sessions.GetActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}

	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}

	_ = s.sessions.TouchLastSeen(ctx, sess.ID, time.Now())

	return u, sess, nil
}

func (s *Service) createSession(ctx context.Context, userID uuid.UUID, userAgent, ip *string) (*AuthResult, error) {
	raw, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}

	expires := time.Now().Add(s.cfg.TTL)
	hash := HashSessionToken(raw)

	sess, err := s.sessions.Create(ctx, userID, hash, expires, userAgent, normalizeIP(ip))
	if err != nil {
		return nil, err
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:      u,
		RawToken:  raw,
		ExpiresAt: expires,
		Session:   sess,
	}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func normalizeIP(ip *string) *string {
	if ip == nil || *ip == "" {
		return nil
	}
	parsed := net.ParseIP(*ip)
	if parsed == nil {
		return nil
	}
	s := parsed.String()
	return &s
}
