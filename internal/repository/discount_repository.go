package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type DiscountFilter struct {
	Search    string
	IsActive  *bool
	OnlyValid bool
	Limit     int
	Offset    int
}

type DiscountRepository interface {
	Create(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error
	FindByID(ctx context.Context, id uint) (*entity.Discount, error)
	FindByCode(ctx context.Context, code string) (*entity.Discount, error)
	FindAll(ctx context.Context, filter DiscountFilter) ([]entity.Discount, int64, error)
	Update(ctx context.Context, discount *entity.Discount, bookIDs []uint, tiers []entity.DiscountTier) error
	Delete(ctx context.Context, id uint) error
	ExistsByCode(ctx context.Context, code string, excludeID ...uint) (bool, error)
	IncrementUsedCount(ctx context.Context, id uint) error
}
