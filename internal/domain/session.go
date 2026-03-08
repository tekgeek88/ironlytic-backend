package domain

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string

	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time

	LastSeenAt *time.Time
	RevokedAt  *time.Time

	UserAgent *string
	IPAddress *string
}
