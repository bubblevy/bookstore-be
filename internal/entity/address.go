package entity

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrAddressNotFound      = errors.New("address not found")
	ErrAddressLimitReached  = errors.New("maximum address limit reached")
	ErrUnauthorizedAddress  = errors.New("unauthorized to access this address")
	ErrPrimaryAddressDelete = errors.New("cannot delete primary address directly")
)

type Address struct {
	gorm.Model
	UserID          uint   `gorm:"not null;index" json:"user_id"`
	ReceiverName    string `gorm:"type:varchar(150);not null" json:"receiver_name"`
	Phone           string `gorm:"type:varchar(30);not null" json:"phone"`
	AddressLine     string `gorm:"type:text;not null" json:"address_line"`
	BiteshipAreaID  string `gorm:"type:varchar(100);not null;index" json:"biteship_area_id"`
	ProvinceName    string `gorm:"type:varchar(100);not null" json:"province_name"`
	CityName        string `gorm:"type:varchar(100);not null" json:"city_name"`
	DistrictName    string `gorm:"type:varchar(100);not null" json:"district_name"`
	SubdistrictName string `gorm:"type:varchar(100);default:''" json:"subdistrict_name"`
	PostalCode      string `gorm:"type:varchar(10);not null" json:"postal_code"`
	IsPrimary       bool   `gorm:"default:false" json:"is_primary"`
}

func (Address) TableName() string {
	return "addresses"
}
