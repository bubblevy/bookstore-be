package entity

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

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
	Addresses []UserAddress `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"addresses,omitempty"`
	Orders    []Order       `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

func (User) TableName() string {
	return "users"
}
