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
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	WeightGrams int     `json:"weight_grams"`
	Subtotal    float64 `json:"subtotal"`
}

type CheckoutPreviewResponse struct {
	Items            []CheckoutItemPreview `json:"items"`
	ItemsSubtotal    float64               `json:"items_subtotal"`
	TotalWeightGrams int                   `json:"total_weight_grams"`
	Courier          string                `json:"courier"`
	CourierService   string                `json:"courier_service"`
	ShippingCost     float64               `json:"shipping_cost"`
	DiscountCode     *string               `json:"discount_code,omitempty"`
	DiscountAmount   float64               `json:"discount_amount"`
	PaymentMethod    string                `json:"payment_method"`
	AdminFee         float64               `json:"admin_fee"`
	TotalPayment     float64               `json:"total_payment"`
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
	TotalAmount     float64             `json:"total_amount"`
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
