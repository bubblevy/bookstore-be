package entity

import (
	"gorm.io/gorm"
)

type ShippingStatus string

const (
	ShippingStatusPending   ShippingStatus = "pending"
	ShippingStatusPickedUp  ShippingStatus = "picked_up"
	ShippingStatusInTransit ShippingStatus = "in_transit"
	ShippingStatusDelivered ShippingStatus = "delivered"
)

type Shipment struct {
	gorm.Model
	OrderID           uint           `gorm:"uniqueIndex;not null" json:"order_id"`
	Courier           string         `gorm:"type:varchar(50);not null" json:"courier"`         // jne, pos, tiki, sicepat
	CourierService    string         `gorm:"type:varchar(50);not null" json:"courier_service"` // REG, YES, OKE, dll.
	ETD               *string        `gorm:"type:varchar(30)" json:"etd,omitempty"`            // Estimasi tiba (e.g. "2-3 HARI")
	TotalWeight       int            `gorm:"not null" json:"total_weight"`                     // Berat total dalam gram
	TrackingNumber    *string        `gorm:"type:varchar(100)" json:"tracking_number,omitempty"`
	Status            ShippingStatus `gorm:"type:shipping_status;default:'pending';not null" json:"status"`
	OriginCityID      int            `gorm:"not null" json:"origin_city_id"`      // ID kota toko/gudang
	DestinationCityID int            `gorm:"not null" json:"destination_city_id"` // ID kota customer
}

func (Shipment) TableName() string {
	return "shipments"
}
