package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type SessionRepository interface {
	Create(ctx context.Context, session *entity.UserSession) error
	FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.UserSession, error)
	Revoke(ctx context.Context, id uint) error
	RevokeAllByUserID(ctx context.Context, userID uint) error
}
