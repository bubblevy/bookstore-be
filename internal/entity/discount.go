package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrDiscountNotFound      = errors.New("discount not found")
	ErrDiscountCodeConflict  = errors.New("discount code already exists")
	ErrDiscountInactive      = errors.New("discount voucher is not active")
	ErrDiscountNotStarted    = errors.New("discount voucher is not yet valid")
	ErrDiscountExpired       = errors.New("discount voucher has expired")
	ErrDiscountQuotaExhaust  = errors.New("discount quota has been exhausted")
	ErrDiscountMinPurchase   = errors.New("minimum purchase requirement not met")
	ErrDiscountNotApplicable = errors.New("voucher is not applicable to the selected books")
)

type DiscountType string

const (
	DiscountTypePercentage DiscountType = "PERCENTAGE"
	DiscountTypeFixed      DiscountType = "FIXED"
)

type Discount struct {
	gorm.Model
	Code              string       `gorm:"type:varchar(50);not null;uniqueIndex:uq_discounts_code_active,where:deleted_at IS NULL" json:"code"`
	Name              string       `gorm:"type:varchar(150);not null" json:"name"`
	Description       *string      `gorm:"type:text" json:"description,omitempty"`
	DiscountType      DiscountType `gorm:"type:varchar(20);not null" json:"discount_type"`
	DiscountValue     float64      `gorm:"type:numeric(12,2);not null" json:"discount_value"`
	MinPurchase       float64      `gorm:"type:numeric(12,2);default:0;not null" json:"min_purchase"`
	MaxDiscountAmount *float64     `gorm:"type:numeric(12,2)" json:"max_discount_amount,omitempty"`
	Quota             int          `gorm:"default:0;not null" json:"quota"`
	UsedCount         int          `gorm:"default:0;not null" json:"used_count"`
	IsAllBooks        bool         `gorm:"default:true;not null" json:"is_all_books"`
	IsActive          bool         `gorm:"default:true;not null" json:"is_active"`
	StartDate         time.Time    `gorm:"not null" json:"start_date"`
	EndDate           time.Time    `gorm:"not null" json:"end_date"`

	// Many-to-Many Pivot Relationship
	Books []Book `gorm:"many2many:discount_books;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"books,omitempty"`
}

func (Discount) TableName() string {
	return "discounts"
}

func (d *Discount) IsValidNow() error {
	if !d.IsActive {
		return ErrDiscountInactive
	}

	now := time.Now()
	if now.Before(d.StartDate) {
		return ErrDiscountNotStarted
	}
	if now.After(d.EndDate) {
		return ErrDiscountExpired
	}

	// Check quota if it's greater than 0 (0 means unlimited)
	if d.Quota > 0 && d.UsedCount >= d.Quota {
		return ErrDiscountQuotaExhaust
	}

	return nil
}

// CalculateDiscount calculates the discount amount based on the subtotal and the discount type.
func (d *Discount) CalculateDiscount(subtotal float64) (float64, error) {
	if err := d.IsValidNow(); err != nil {
		return 0, err
	}

	if subtotal < d.MinPurchase {
		return 0, ErrDiscountMinPurchase
	}

	var deduction float64
	switch d.DiscountType {
	case DiscountTypePercentage:
		deduction = (subtotal * d.DiscountValue) / 100.0
		if d.MaxDiscountAmount != nil && deduction > *d.MaxDiscountAmount {
			deduction = *d.MaxDiscountAmount
		}
	case DiscountTypeFixed:
		deduction = d.DiscountValue
		if deduction > subtotal {
			deduction = subtotal
		}
	}

	return deduction, nil
}
