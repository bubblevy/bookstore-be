package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrAddressNotFound            = errors.New("address not found")
	ErrAddressForbidden           = errors.New("you are not authorized to access this address")
	ErrFailedCreateAddress        = errors.New("failed to create address")
	ErrFailedUpdateAddress        = errors.New("failed to update address")
	ErrFailedDeleteAddress        = errors.New("failed to delete address")
	ErrFailedFetchAddress         = errors.New("failed to fetch address")
	ErrAddressLimitReached        = errors.New("maximum address limit reached (maximum 5 addresses)")
	ErrCannotDeletePrimaryAddress = errors.New("cannot delete primary address, please set another address as primary first")
)

type UserAddress struct {
	gorm.Model
	UserID          uint    `gorm:"not null;index" json:"user_id"`
	ReceiverName    string  `gorm:"type:varchar(150);not null" json:"receiver_name"`
	Phone           string  `gorm:"type:varchar(20);not null" json:"phone"`
	AddressLine     string  `gorm:"type:varchar(255);not null" json:"address_line"`
	DestinationID   int     `gorm:"not null;index" json:"destination_id"`
	ProvinceName    string  `gorm:"type:varchar(100);not null" json:"province_name"`
	CityName        string  `gorm:"type:varchar(100);not null" json:"city_name"`
	DistrictName    string  `gorm:"type:varchar(100);not null" json:"district_name"`
	SubdistrictName *string `gorm:"type:varchar(100)" json:"subdistrict_name"`
	PostalCode      string  `gorm:"type:varchar(10);not null" json:"postal_code"`
	IsPrimary       bool    `gorm:"default:false;not null" json:"is_primary"`

	// belonngs to relationship
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserAddress) TableName() string {
	return "user_addresses"
}
