package pgsql

import (
	"context"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) repository.SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) Create(ctx context.Context, session *entity.UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.UserSession, error) {
	var session entity.UserSession
	err := r.db.WithContext(ctx).
		Where("refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?",
			tokenHash,
			time.Now(),
		).First(&session).Error

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *sessionRepository) Revoke(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.UserSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

func (r *sessionRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).
		Error
}
