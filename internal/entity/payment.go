package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	OrderID                uint           `gorm:"not null;index" json:"order_id"`
	MidtransTransactionID  *string        `gorm:"type:varchar(100);index" json:"midtrans_transaction_id,omitempty"`
	GrossAmount            float64        `gorm:"type:numeric(12,2);not null" json:"gross_amount"`
	PaymentType            *string        `gorm:"type:varchar(50)" json:"payment_type,omitempty"`            // qris, bank_transfer, gopay, dll.
	TransactionStatus      string         `gorm:"type:varchar(50);not null;index" json:"transaction_status"` // capture, settlement, pending, deny, expire, cancel
	FraudStatus            *string        `gorm:"type:varchar(50)" json:"fraud_status,omitempty"`            // accept, challenge, deny
	SnapToken              *string        `gorm:"type:varchar(255)" json:"snap_token,omitempty"`
	SnapRedirectURL        *string        `gorm:"type:text" json:"snap_redirect_url,omitempty"`
	PaymentResponsePayload datatypes.JSON `gorm:"type:jsonb" json:"payment_response_payload,omitempty"` // Webhook notification dump
	PaidAt                 *time.Time     `json:"paid_at,omitempty"`
}

func (Payment) TableName() string {
	return "payments"
}
