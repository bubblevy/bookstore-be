package dto

import "time"

type OrderListItemResponse struct {
	ID             uint      `json:"id"`
	InvoiceNumber  string    `json:"invoice_number"`
	Status         string    `json:"status"`
	TotalAmount    int       `json:"total_amount"`
	TotalQuantity  int       `json:"total_quantity"`
	FirstItemTitle string    `json:"first_item_title"`
	FirstItemCover *string   `json:"first_item_cover,omitempty"`
	ExtraItemCount int       `json:"extra_item_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type OrderRecipientResponse struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Province    string `json:"province"`
	City        string `json:"city"`
	District    string `json:"district"`
	Subdistrict string `json:"subdistrict"`
	PostalCode  string `json:"postal_code"`
}

type OrderShippingResponse struct {
	Courier        string  `json:"courier"`
	CourierService string  `json:"courier_service"`
	TrackingNumber *string `json:"tracking_number,omitempty"`
	Status         string  `json:"status"`
	TrackingURL    string  `json:"tracking_url"`
}

type OrderPaymentResponse struct {
	PaymentType string     `json:"payment_type"`
	Bank        *string    `json:"bank,omitempty"`
	VANumber    *string    `json:"va_number,omitempty"`
	BillKey     *string    `json:"bill_key,omitempty"`
	BillerCode  *string    `json:"biller_code,omitempty"`
	QRURL       *string    `json:"qr_url,omitempty"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

type OrderItemResponse struct {
	BookID      uint    `json:"book_id"`
	Title       string  `json:"title"`
	CoverImage  *string `json:"cover_image,omitempty"`
	Price       int     `json:"price"`
	Quantity    int     `json:"quantity"`
	WeightGrams int     `json:"weight_grams"`
	Subtotal    int     `json:"subtotal"`
}

type OrderPricingResponse struct {
	ItemsSubtotal  int `json:"items_subtotal"`
	ShippingCost   int `json:"shipping_cost"`
	DiscountAmount int `json:"discount_amount"`
	AdminFee       int `json:"admin_fee"`
	TotalAmount    int `json:"total_amount"`
}

type OrderDetailResponse struct {
	ID                 uint                   `json:"id"`
	InvoiceNumber      string                 `json:"invoice_number"`
	Status             string                 `json:"status"`
	PaymentDeadline    time.Time              `json:"payment_deadline"`
	PaidAt             *time.Time             `json:"paid_at,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty"`
	CancellationReason *string                `json:"cancellation_reason,omitempty"`
	ReturnedAt         *time.Time             `json:"returned_at,omitempty"`
	ReturnReason       *string                `json:"return_reason,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	Recipient          OrderRecipientResponse `json:"recipient"`
	Shipping           OrderShippingResponse  `json:"shipping"`
	Payment            *OrderPaymentResponse  `json:"payment,omitempty"`
	Items              []OrderItemResponse    `json:"items"`
	Pricing            OrderPricingResponse   `json:"pricing"`
}
