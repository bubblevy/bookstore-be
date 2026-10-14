package rajaongkir

type BaseResponse[T any] struct {
	Meta struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
		Status  string `json:"status"`
	} `json:"meta"`
	Data T `json:"data"`
}

type DomesticDestination struct {
	ID          int    `json:"id"`
	Label       string `json:"label"`
	Subdistrict string `json:"subdistrict_name"`
	District    string `json:"district_name"`
	City        string `json:"city_name"`
	Province    string `json:"province_name"`
	ZipCode     string `json:"zip_code"`
}

type CalculateDomesticCostRequest struct {
	OriginID      int    `json:"origin"`
	DestinationID int    `json:"destination"`
	WeightGrams   int    `json:"weight"`
	Courier       string `json:"courier"`
	Price         string `json:"price"`
}

type ShippingCostResult struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Service     string `json:"service"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`
	ETD         string `json:"etd"`
}
