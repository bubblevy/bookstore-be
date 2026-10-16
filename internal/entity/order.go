package entity

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	OrderStatusWaitingPayment OrderStatus = "waiting_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusProcessing     OrderStatus = "processing"
	OrderStatusShipped        OrderStatus = "shipped"
	OrderStatusCompleted      OrderStatus = "completed"
	OrderStatusCancelled      OrderStatus = "cancelled"
)

var (
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderAlreadyPaid       = errors.New("order has already been paid")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current state")
	ErrPaymentExpired         = errors.New("payment deadline has passed")
	ErrOrderNotPaid           = errors.New("order is not paid yet or already processed")
	ErrOrderAlreadyFulfilled  = errors.New("order shipment has already been processed")
)

type Order struct {
	ID                     uint        `gorm:"primaryKey" json:"id"`
	UserID                 uint        `gorm:"not null" json:"user_id"`
	InvoiceNumber          string      `gorm:"type:varchar(60);not null;unique" json:"invoice_number"`
	Status                 OrderStatus `gorm:"type:order_status;not null;default:'waiting_payment'" json:"status"`
	ItemsSubtotal          float64     `gorm:"type:numeric(15,2);not null" json:"items_subtotal"`
	ShippingCost           float64     `gorm:"type:numeric(15,2);not null;default:0" json:"shipping_cost"`
	DiscountAmount         float64     `gorm:"type:numeric(15,2);not null;default:0" json:"discount_amount"`
	AdminFee               float64     `gorm:"type:numeric(15,2);not null;default:0" json:"admin_fee"`
	TotalAmount            float64     `gorm:"type:numeric(15,2);not null" json:"total_amount"`
	DiscountID             *uint       `json:"discount_id,omitempty"`
	DiscountCode           *string     `gorm:"type:varchar(50)" json:"discount_code,omitempty"`
	Courier                string      `gorm:"type:varchar(30);not null" json:"courier"`
	CourierService         string      `gorm:"type:varchar(50);not null" json:"courier_service"`
	TotalWeightGrams       int         `gorm:"not null" json:"total_weight_grams"`
	ShippingTrackingNumber *string     `gorm:"type:varchar(100)" json:"shipping_tracking_number,omitempty"`
	RecipientName          string      `gorm:"type:varchar(150);not null" json:"recipient_name"`
	RecipientPhone         string      `gorm:"type:varchar(30);not null" json:"recipient_phone"`
	RecipientAddress       string      `gorm:"type:text;not null" json:"recipient_address"`
	ProvinceName           string      `gorm:"type:varchar(100);not null" json:"province_name"`
	CityName               string      `gorm:"type:varchar(100);not null" json:"city_name"`
	DistrictName           string      `gorm:"type:varchar(100);not null" json:"district_name"`
	SubdistrictName        string      `gorm:"type:varchar(100);default:''" json:"subdistrict_name"`
	PostalCode             string      `gorm:"type:varchar(10);not null" json:"postal_code"`
	PaymentDeadline        time.Time   `gorm:"not null" json:"payment_deadline"`
	PaidAt                 *time.Time  `json:"paid_at,omitempty"`
	CancelledAt            *time.Time  `json:"cancelled_at,omitempty"`
	CancellationReason     *string     `gorm:"type:text" json:"cancellation_reason,omitempty"`
	CreatedAt              time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	Items                  []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items"`
	Payment                *Payment    `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"payment,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}
