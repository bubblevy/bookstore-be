package entity

import (
	"errors"
	"math"
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
	ErrDiscountTierRequired  = errors.New("tiered discount requires at least one tier definition")
	ErrDiscountBookNotFound  = errors.New("one or more selected books do not exist")
)

type DiscountType string

const (
	DiscountTypePercentage DiscountType = "PERCENTAGE" // Flat / standard
	DiscountTypeFixed      DiscountType = "FIXED"      // Flat nominal
	DiscountTypeTiered     DiscountType = "TIERED"     // Diskon bertingkat "Up To"
)

type DiscountTier struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	DiscountID  uint         `gorm:"not null;index" json:"discount_id"`
	MinSpend    int          `gorm:"type:bigint;not null" json:"min_spend"`
	RewardType  DiscountType `gorm:"type:varchar(20);not null" json:"reward_type"`
	RewardValue int          `gorm:"type:bigint;not null" json:"reward_value"`
	MaxReward   *int         `gorm:"type:bigint" json:"max_reward,omitempty"`
}

func (DiscountTier) TableName() string {
	return "discount_tiers"
}

type Discount struct {
	gorm.Model
	Code              string       `gorm:"type:varchar(50);not null;uniqueIndex:uq_discounts_code_active,where:deleted_at IS NULL" json:"code"`
	Name              string       `gorm:"type:varchar(150);not null" json:"name"`
	Description       *string      `gorm:"type:text" json:"description,omitempty"`
	DiscountType      DiscountType `gorm:"type:varchar(20);not null" json:"discount_type"`
	DiscountValue     int          `gorm:"type:bigint;not null" json:"discount_value"`
	MinPurchase       int          `gorm:"type:bigint;default:0;not null" json:"min_purchase"`
	MaxDiscountAmount *int         `gorm:"type:bigint" json:"max_discount_amount,omitempty"`
	Quota             int          `gorm:"default:0;not null" json:"quota"`
	UsedCount         int          `gorm:"default:0;not null" json:"used_count"`
	IsAllBooks        *bool        `gorm:"not null" json:"is_all_books"`
	IsActive          bool         `gorm:"default:true;not null" json:"is_active"`
	StartDate         time.Time    `gorm:"not null" json:"start_date"`
	EndDate           time.Time    `gorm:"not null" json:"end_date"`

	// Relasi Many-to-Many ke Buku
	Books []Book `gorm:"many2many:discount_books;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"books,omitempty"`

	// Relasi One-to-Many ke Tiers (if TIERED)
	Tiers []DiscountTier `gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tiers,omitempty"`
}

func (Discount) TableName() string {
	return "discounts"
}

func (d *Discount) GetIsAllBooks() bool {
	if d.IsAllBooks == nil {
		return true
	}
	return *d.IsAllBooks
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
	if d.Quota > 0 && d.UsedCount >= d.Quota {
		return ErrDiscountQuotaExhaust
	}
	return nil
}

func (d *Discount) CalculateDiscount(subtotal int) (int, error) {
	if err := d.IsValidNow(); err != nil {
		return 0, err
	}

	if subtotal < d.MinPurchase {
		return 0, ErrDiscountMinPurchase
	}

	var deduction int

	switch d.DiscountType {
	case DiscountTypeTiered:
		var bestTier *DiscountTier
		for i := range d.Tiers {
			tier := &d.Tiers[i]
			if subtotal >= tier.MinSpend {
				if bestTier == nil || tier.MinSpend > bestTier.MinSpend {
					bestTier = tier
				}
			}
		}

		if bestTier == nil {
			return 0, ErrDiscountMinPurchase
		}

		if bestTier.RewardType == DiscountTypePercentage {
			deduction = int(math.Round((float64(subtotal) * float64(bestTier.RewardValue)) / 100.0))
			if bestTier.MaxReward != nil && deduction > *bestTier.MaxReward {
				deduction = *bestTier.MaxReward
			}
		} else {
			deduction = bestTier.RewardValue
		}

	case DiscountTypePercentage:
		deduction = int(math.Round((float64(subtotal) * float64(d.DiscountValue)) / 100.0))
		if d.MaxDiscountAmount != nil && deduction > *d.MaxDiscountAmount {
			deduction = *d.MaxDiscountAmount
		}

	case DiscountTypeFixed:
		deduction = d.DiscountValue
	}

	if d.MaxDiscountAmount != nil && deduction > *d.MaxDiscountAmount {
		deduction = *d.MaxDiscountAmount
	}
	if deduction > subtotal {
		deduction = subtotal
	}

	return deduction, nil
}
