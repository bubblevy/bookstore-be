package entity

import (
	"time"

	"gorm.io/gorm"
)

type UserSession struct {
	gorm.Model
	UserID           uint      `gorm:"not null;index"`
	RefreshTokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	ExpiresAt        time.Time `gorm:"not null;index"`
	RevokedAt        *time.Time
}

func (UserSession) TableName() string {
	return "user_sessions"
}
