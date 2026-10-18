package dto

type GetShippingRatesRequest struct {
	CartItemIDs []uint `json:"cart_item_ids" validate:"required,min=1"`
	AddressID   *uint  `json:"address_id,omitempty"`
}

type CourierOptionResponse struct {
	CourierCode        string  `json:"courier_code"`
	CourierName        string  `json:"courier_name"`
	CourierServiceName string  `json:"courier_service_name"`
	CourierServiceCode string  `json:"courier_service_code"`
	Description        string  `json:"description"`
	Price              int     `json:"price"`
	ETD                string  `json:"etd"`
	Type               string  `json:"type"`
}

type ShippingRatesResponse struct {
	TotalWeightGrams int                     `json:"total_weight_grams"`
	DestinationArea  string                  `json:"destination_area"`
	Options          []CourierOptionResponse `json:"options"`
}

type FulfillOrderRequest struct {
	OrderNote string `json:"order_note"`
}

type ProcessReturnRequest struct {
	Reason string `json:"reason"`
}

type FulfillOrderResponse struct {
	ShipmentID      uint    `json:"shipment_id"`
	InvoiceNumber   string  `json:"invoice_number"`
	BiteshipOrderID string  `json:"biteship_order_id"`
	WaybillNumber   *string `json:"waybill_number,omitempty"`
	Status          string  `json:"status"`
	Courier         string  `json:"courier"`
	CourierService  string  `json:"courier_service"`
	TrackingURL     *string `json:"tracking_url,omitempty"`
}
