package entity

import "gorm.io/gorm"

type UserAddress struct {
	gorm.Model
	UserID          uint    `gorm:"not null;index" json:"user_id"`
	ReceiverName    string  `gorm:"type:varchar(150);not null" json:"receiver_name"`
	Phone           string  `gorm:"type:varchar(20);not null" json:"phone"`
	AddressLine     string  `gorm:"type:text;not null" json:"address_line"`
	ProvinceID      int     `gorm:"not null" json:"province_id"`
	ProvinceName    string  `gorm:"type:varchar(100);not null" json:"province_name"`
	CityID          int     `gorm:"not null" json:"city_id"`
	CityName        string  `gorm:"type:varchar(100);not null" json:"city_name"`
	SubdistrictID   *int    `json:"subdistrict_id"`
	SubdistrictName *string `gorm:"type:varchar(100)" json:"subdistrict_name"`
	PostalCode      string  `gorm:"type:varchar(10);not null" json:"postal_code"`
	IsPrimary       bool    `gorm:"default:false;not null" json:"is_primary"`

	// belonngs to relationship
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserAddress) TableName() string {
	return "user_addresses"
}
