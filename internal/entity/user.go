package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type UserRole string

var (
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrAccountInactive             = errors.New("account is inactive")
	ErrEmailAlreadyRegistered      = errors.New("email already registered")
	ErrUserNotFound                = errors.New("user not found")
	ErrInvalidUserId               = errors.New("invalid user id")
	ErrInvalidToken                = errors.New("invalid or expired token")
	ErrInvalidTokenClaims          = errors.New("invalid token claims")
	ErrInvalidTokenType            = errors.New("invalid token type")
	ErrInvalidTokenIssuer          = errors.New("invalid token issuer")
	ErrInvalidRefreshToken         = errors.New("invalid or expired refresh token")
	ErrFailedHashPassword          = errors.New("failed to hash password")
	ErrFailedGenerateToken         = errors.New("failed to generate token")
	ErrActivationTokenRequired     = errors.New("activation token is required")
	ErrInvalidActivationToken      = errors.New("invalid activation token")
	ErrActivationTokenExpired      = errors.New("activation token has expired")
	ErrChangePasswordTokenRequired = errors.New("change password token is required")
	ErrChangePasswordTokenExpired  = errors.New("change password token has expired")
	ErrPasswordMismatch            = errors.New("password confirmation mismatch")
	ErrResetPasswordTokenRequired  = errors.New("reset password token is required")
	ErrResetPasswordTokenExpired   = errors.New("reset password token has expired")
	ErrInvalidCurrentPassword      = errors.New("current password is incorrect")
	ErrAccountAlreadyActive        = errors.New("account is already active")
)

const (
	RoleCustomer UserRole = "customer"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	gorm.Model
	Name                   string     `gorm:"type:varchar(100);not null" json:"name"`
	Email                  string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password               string     `gorm:"type:varchar(255);" json:"-"`
	Role                   UserRole   `gorm:"type:varchar(20);not null;default:customer" json:"role"`
	GoogleID               *string    `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	IsActive               bool       `gorm:"default:false;not null" json:"is_active"`
	ActivationToken        *string    `gorm:"type:varchar(255);index" json:"-"`
	ActivationExpiresAt    *time.Time `json:"-"`
	ResetPasswordToken     *string    `gorm:"type:varchar(255);index" json:"-"`
	ResetPasswordExpiresAt *time.Time `json:"-"`

	// one to many relationship
	Sessions  []UserSession `gorm:"foreignKey:UserID"`
	Addresses []Address     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"addresses,omitempty"`
	Orders    []Order       `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

func (User) TableName() string {
	return "users"
}
