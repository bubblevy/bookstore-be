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
	db := extractDB(ctx, r.db)
	return db.Create(session).Error
}

func (r *sessionRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.UserSession, error) {
	var session entity.UserSession
	db := extractDB(ctx, r.db)
	err := db.Where("refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?",
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
	db := extractDB(ctx, r.db)
	return db.Model(&entity.UserSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

func (r *sessionRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	now := time.Now()
	db := extractDB(ctx, r.db)
	return db.Model(&entity.UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).
		Error
}

func (r *sessionRepository) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptTokenHash string) error {
	now := time.Now()
	db := extractDB(ctx, r.db)
	return db.Model(&entity.UserSession{}).
		Where("user_id = ? AND refresh_token_hash != ?", userID, exceptTokenHash).
		Update("revoked_at", now).Error
}

func (r *sessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	db := extractDB(ctx, r.db)
	result := db.Where("expires_at < ?", time.Now()).
		Delete(&entity.UserSession{})
	return result.RowsAffected, result.Error
}
