package entity

import (
	"errors"
	"time"
)

type ShippingStatus string

const (
	ShippingStatusPlaced          ShippingStatus = "placed"
	ShippingStatusScheduled       ShippingStatus = "scheduled"
	ShippingStatusConfirmed       ShippingStatus = "confirmed"
	ShippingStatusAllocated       ShippingStatus = "allocated"
	ShippingStatusPickingUp       ShippingStatus = "picking_up"
	ShippingStatusPicked          ShippingStatus = "picked"
	ShippingStatusDroppingOff     ShippingStatus = "dropping_off"
	ShippingStatusInTransit       ShippingStatus = "in_transit"
	ShippingStatusReturnInTransit ShippingStatus = "return_in_transit"
	ShippingStatusDelivered       ShippingStatus = "delivered"
	ShippingStatusRejected        ShippingStatus = "rejected"
	ShippingStatusCourierNotFound ShippingStatus = "courier_not_found"
	ShippingStatusCancelled       ShippingStatus = "cancelled"
	ShippingStatusReturned        ShippingStatus = "returned"
	ShippingStatusOnHold          ShippingStatus = "on_hold"
	ShippingStatusDisposed        ShippingStatus = "disposed"
)

var (
	ErrShipmentNotFound       = errors.New("shipment details not found for this order")
	ErrTrackingNumberNotReady = errors.New("tracking number has not been issued by the courier yet")
	ErrUnauthorizedTracking   = errors.New("unauthorized access to tracking details")
	ErrInvalidWebhookPayload  = errors.New("invalid webhook payload")
	ErrInvalidOrderIdentifier = errors.New("no valid order identifier found in webhook payload")
	ErrDispatchCourierFailed  = errors.New("failed to dispatch pickup request to courier provider")
	ErrShippingRateFailed     = errors.New("failed to retrieve shipping rates from courier")
	ErrInvalidCourierService  = errors.New("invalid or unsupported courier service")
)

type Shipment struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	OrderID            uint           `gorm:"not null;uniqueIndex" json:"order_id"`
	BiteshipOrderID    string         `gorm:"size:100;uniqueIndex;not null" json:"biteship_order_id"`
	BiteshipTrackingID *string        `gorm:"size:100" json:"biteship_tracking_id,omitempty"`
	WaybillNumber      *string        `gorm:"size:100;index" json:"waybill_number,omitempty"`
	Courier            string         `gorm:"size:50;not null" json:"courier"`
	CourierService     string         `gorm:"size:50;not null" json:"courier_service"`
	Status             ShippingStatus `gorm:"type:shipping_status;not null;default:'allocated'" json:"status"`
	TrackingURL        *string        `gorm:"type:text" json:"tracking_url,omitempty"`
	OrderNote          *string        `gorm:"type:text" json:"order_note,omitempty"`
	CreatedAt          time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	Order *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

func (Shipment) TableName() string {
	return "shipments"
}
