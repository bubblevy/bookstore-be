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
	Password               string     `gorm:"type:varchar(255);not null" json:"-"`
	Role                   UserRole   `gorm:"type:varchar(20);not null;default:user" json:"role"`
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

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strong_password"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserResponse struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ResendActivationRequest struct {
	Email string `json:"email" validate:"required,email"`
}
