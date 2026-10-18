package dto

import "time"

type CheckoutPreviewRequest struct {
	CartItemIDs    []uint  `json:"cart_item_ids" validate:"required,min=1"`
	AddressID      *uint   `json:"address_id,omitempty"`
	Courier        string  `json:"courier" validate:"required"`
	CourierService string  `json:"courier_service" validate:"required"`
	DiscountCode   *string `json:"discount_code,omitempty"`
	PaymentMethod  string  `json:"payment_method" validate:"required"`
}

type CheckoutItemPreview struct {
	BookID      uint    `json:"book_id"`
	Title       string  `json:"title"`
	CoverImage  *string `json:"cover_image,omitempty"`
	Price       int     `json:"price"`
	Quantity    int     `json:"quantity"`
	WeightGrams int     `json:"weight_grams"`
	Subtotal    int     `json:"subtotal"`
}

type AddressPreviewResponse struct {
	ID              uint   `json:"id"`
	ReceiverName    string `json:"receiver_name"`
	Phone           string `json:"phone"`
	AddressLine     string `json:"address_line"`
	ProvinceName    string `json:"province_name"`
	CityName        string `json:"city_name"`
	DistrictName    string `json:"district_name"`
	SubdistrictName string `json:"subdistrict_name"`
	PostalCode      string `json:"postal_code"`
}

type CheckoutPreviewResponse struct {
	Address          *AddressPreviewResponse `json:"address,omitempty"`
	Items            []CheckoutItemPreview   `json:"items"`
	ItemsSubtotal    int                     `json:"items_subtotal"`
	TotalWeightGrams int                     `json:"total_weight_grams"`
	Courier          string                  `json:"courier"`
	CourierService   string                  `json:"courier_service"`
	ShippingCost     int                     `json:"shipping_cost"`
	DiscountCode     *string                 `json:"discount_code,omitempty"`
	DiscountAmount   int                     `json:"discount_amount"`
	PaymentMethod    string                  `json:"payment_method"`
	AdminFee         int                     `json:"admin_fee"`
	TotalPayment     int                     `json:"total_payment"`
}

type CreateOrderRequest struct {
	CartItemIDs    []uint  `json:"cart_item_ids" validate:"required,min=1"`
	AddressID      *uint   `json:"address_id,omitempty"`
	Courier        string  `json:"courier" validate:"required"`
	CourierService string  `json:"courier_service" validate:"required"`
	DiscountCode   *string `json:"discount_code,omitempty"`
	PaymentMethod  string  `json:"payment_method" validate:"required"`
}

type PaymentInfoResponse struct {
	PaymentType string  `json:"payment_type"`
	Bank        *string `json:"bank,omitempty"`
	VANumber    *string `json:"va_number,omitempty"`
	BillKey     *string `json:"bill_key,omitempty"`
	BillerCode  *string `json:"biller_code,omitempty"`
	QRURL       *string `json:"qr_url,omitempty"`
}

type CreateOrderResponse struct {
	InvoiceNumber   string              `json:"invoice_number"`
	Status          string              `json:"status"`
	TotalAmount     int                 `json:"total_amount"`
	PaymentDeadline time.Time           `json:"payment_deadline"`
	PaymentInfo     PaymentInfoResponse `json:"payment_info"`
}

type MidtransWebhookPayload struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
}
