package entity

import (
	"errors"
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusSettlement PaymentStatus = "settlement"
	PaymentStatusExpire     PaymentStatus = "expire"
	PaymentStatusCancel     PaymentStatus = "cancel"
	PaymentStatusDeny       PaymentStatus = "deny"
	PaymentStatusRefund     PaymentStatus = "refund"
)

var (
	ErrInvalidPaymentMethod = errors.New("unsupported or invalid payment method")
	ErrMidtransChargeFailed = errors.New("midtrans charge request failed")
)

type Payment struct {
	ID                    uint          `gorm:"primaryKey" json:"id"`
	OrderID               uint          `gorm:"not null" json:"order_id"`
	MidtransTransactionID *string       `gorm:"type:varchar(100)" json:"midtrans_transaction_id,omitempty"`
	PaymentType           string        `gorm:"type:varchar(50);not null" json:"payment_type"`
	Bank                  *string       `gorm:"type:varchar(30)" json:"bank,omitempty"`
	VANumber              *string       `gorm:"type:varchar(100)" json:"va_number,omitempty"`
	BillKey               *string       `gorm:"type:varchar(100)" json:"bill_key,omitempty"`
	BillerCode            *string       `gorm:"type:varchar(100)" json:"biller_code,omitempty"`
	QRURL                 *string       `gorm:"type:text" json:"qr_url,omitempty"`
	Status                PaymentStatus `gorm:"type:payment_status;not null;default:'pending'" json:"status"`
	RawResponse           *string       `gorm:"type:jsonb" json:"raw_response,omitempty"`
	PaidAt                *time.Time    `json:"paid_at,omitempty"`
	CreatedAt             time.Time     `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt             time.Time     `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Payment) TableName() string {
	return "payments"
}
