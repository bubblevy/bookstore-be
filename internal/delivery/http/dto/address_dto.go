package dto

import "time"

type CreateAddressRequest struct {
	ReceiverName    string `json:"receiver_name" validate:"required,min=2,max=150"`
	Phone           string `json:"phone" validate:"required,min=8,max=20"`
	AddressLine     string `json:"address_line" validate:"required,min=5"`
	BiteshipAreaID  string `json:"biteship_area_id" validate:"required"`
	ProvinceName    string `json:"province_name" validate:"required"`
	CityName        string `json:"city_name" validate:"required"`
	DistrictName    string `json:"district_name" validate:"required"`
	SubdistrictName string `json:"subdistrict_name"`
	PostalCode      string `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool   `json:"is_primary"`
}

type UpdateAddressRequest struct {
	ReceiverName    string `json:"receiver_name" validate:"required,min=2,max=150"`
	Phone           string `json:"phone" validate:"required,min=8,max=20"`
	AddressLine     string `json:"address_line" validate:"required,min=5"`
	BiteshipAreaID  string `json:"biteship_area_id" validate:"required"`
	ProvinceName    string `json:"province_name" validate:"required"`
	CityName        string `json:"city_name" validate:"required"`
	DistrictName    string `json:"district_name" validate:"required"`
	SubdistrictName string `json:"subdistrict_name"`
	PostalCode      string `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool   `json:"is_primary"`
}

type AddressResponse struct {
	ID              uint      `json:"id"`
	UserID          uint      `json:"user_id"`
	ReceiverName    string    `json:"receiver_name"`
	Phone           string    `json:"phone"`
	AddressLine     string    `json:"address_line"`
	BiteshipAreaID  string    `json:"biteship_area_id"`
	ProvinceName    string    `json:"province_name"`
	CityName        string    `json:"city_name"`
	DistrictName    string    `json:"district_name"`
	SubdistrictName string    `json:"subdistrict_name"`
	PostalCode      string    `json:"postal_code"`
	IsPrimary       bool      `json:"is_primary"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
