package dto

type CreateAddressRequest struct {
	ReceiverName    string  `json:"receiver_name" validate:"required,min=3,max=150"`
	Phone           string  `json:"phone" validate:"required,min=9,max=20"`
	AddressLine     string  `json:"address_line" validate:"required,min=10,max=255"`
	ProvinceID      int     `json:"province_id" validate:"required,gt=0"`
	ProvinceName    string  `json:"province_name" validate:"required"`
	CityID          int     `json:"city_id" validate:"required,gt=0"`
	CityName        string  `json:"city_name" validate:"required"`
	SubdistrictID   *int    `json:"subdistrict_id" validate:"omitempty,gt=0"`
	SubdistrictName *string `json:"subdistrict_name" validate:"omitempty"`
	PostalCode      string  `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool    `json:"is_primary"`
}

type UpdateAddressRequest struct {
	ReceiverName    string  `json:"receiver_name" validate:"required,min=3,max=150"`
	Phone           string  `json:"phone" validate:"required,min=9,max=20"`
	AddressLine     string  `json:"address_line" validate:"required,min=10,max=255"`
	ProvinceID      int     `json:"province_id" validate:"required,gt=0"`
	ProvinceName    string  `json:"province_name" validate:"required"`
	CityID          int     `json:"city_id" validate:"required,gt=0"`
	CityName        string  `json:"city_name" validate:"required"`
	SubdistrictID   *int    `json:"subdistrict_id" validate:"omitempty,gt=0"`
	SubdistrictName *string `json:"subdistrict_name" validate:"omitempty"`
	PostalCode      string  `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool    `json:"is_primary"`
}
