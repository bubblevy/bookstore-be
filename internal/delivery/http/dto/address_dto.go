package dto

type CreateAddressRequest struct {
	ReceiverName    string  `json:"receiver_name" validate:"required,min=3,max=150"`
	Phone           string  `json:"phone" validate:"required,min=9,max=20"`
	AddressLine     string  `json:"address_line" validate:"required,min=10,max=255"`
	DestinationID   int     `json:"destination_id" validate:"required,gt=0"`
	ProvinceName    string  `json:"province_name" validate:"required"`
	CityName        string  `json:"city_name" validate:"required"`
	DistrictName    string  `json:"district_name" validate:"required"`
	SubdistrictName *string `json:"subdistrict_name" validate:"omitempty"`
	PostalCode      string  `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool    `json:"is_primary"`
}

type UpdateAddressRequest struct {
	ReceiverName    string  `json:"receiver_name" validate:"required,min=3,max=150"`
	Phone           string  `json:"phone" validate:"required,min=9,max=20"`
	AddressLine     string  `json:"address_line" validate:"required,min=10,max=255"`
	DestinationID   int     `json:"destination_id" validate:"required,gt=0"`
	ProvinceName    string  `json:"province_name" validate:"required"`
	CityName        string  `json:"city_name" validate:"required"`
	DistrictName    string  `json:"district_name" validate:"required"`
	SubdistrictName *string `json:"subdistrict_name" validate:"omitempty"`
	PostalCode      string  `json:"postal_code" validate:"required,numeric,len=5"`
	IsPrimary       bool    `json:"is_primary"`
}
