package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrPaymentMethodNotFound      = errors.New("payment method not found")
	ErrPaymentMethodInactive      = errors.New("metode pembayaran sedang dalam pemeliharaan (maintenance)")
	ErrPaymentMethodCodeImmutable = errors.New("payment method code cannot be modified")
)

type PaymentInstruction struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type PaymentInstructions []PaymentInstruction

func (pi PaymentInstructions) Value() (driver.Value, error) {
	if len(pi) == 0 {
		return "[]", nil
	}
	return json.Marshal(pi)
}

func (pi *PaymentInstructions) Scan(value interface{}) error {
	if value == nil {
		*pi = PaymentInstructions{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan PaymentInstructions: invalid type")
	}
	if len(bytes) == 0 {
		*pi = PaymentInstructions{}
		return nil
	}
	return json.Unmarshal(bytes, pi)
}

type PaymentMethod struct {
	ID           uint                `gorm:"primaryKey" json:"id"`
	Code         string              `gorm:"type:varchar(50);not null;unique" json:"code"`
	Name         string              `gorm:"type:varchar(100);not null" json:"name"`
	Category     string              `gorm:"type:varchar(50);not null" json:"category"`
	FeeType      string              `gorm:"type:varchar(20);not null;default:'fixed'" json:"fee_type"`
	FeeAmount    int                 `gorm:"type:bigint;not null;default:0" json:"fee_amount"`
	FeePercent   float64             `gorm:"type:numeric(6,4);not null;default:0" json:"fee_percent"`
	Image        string              `gorm:"type:varchar(255);default:''" json:"image"`
	Description  string              `gorm:"type:text;default:''" json:"description"`
	Instructions PaymentInstructions `gorm:"type:jsonb;not null;default:'[]'" json:"instructions"`
	IsActive     bool                `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time           `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentMethod) TableName() string {
	return "payment_methods"
}
