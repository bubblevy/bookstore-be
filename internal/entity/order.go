package entity

import (
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

type Order struct {
	gorm.Model
	OrderCode    string      `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_code"` // Digunakan sebagai order_id di Midtrans
	UserID       uint        `gorm:"not null;index" json:"user_id"`
	Subtotal     float64     `gorm:"type:numeric(12,2);not null" json:"subtotal"`
	ShippingCost float64     `gorm:"type:numeric(12,2);not null" json:"shipping_cost"`
	TotalAmount  float64     `gorm:"type:numeric(12,2);not null" json:"total_amount"`
	Status       OrderStatus `gorm:"type:order_status;default:'pending';not null;index" json:"status"`
	Notes        *string     `gorm:"type:text" json:"notes,omitempty"`

	// Relasi
	User       *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order_items,omitempty"`
	Shipment   *Shipment   `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"shipment,omitempty"`
	Payment    *Payment    `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"payment,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}
