package entity

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	gorm.Model
	Name                   string     `gorm:"type:varchar(100);not null" json:"name"`
	Email                  string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password               string     `gorm:"type:varchar(255);" json:"-"`
	Role                   UserRole   `gorm:"type:varchar(20);not null;default:user" json:"role"`
	GoogleID               *string    `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	IsActive               bool       `gorm:"default:false;not null" json:"is_active"`
	ActivationToken        *string    `gorm:"type:varchar(255);index" json:"-"`
	ActivationExpiresAt    *time.Time `json:"-"`
	ResetPasswordToken     *string    `gorm:"type:varchar(255);index" json:"-"`
	ResetPasswordExpiresAt *time.Time `json:"-"`

	Sessions []UserSession `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}
