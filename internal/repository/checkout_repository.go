package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"gorm.io/gorm"
)

type CheckoutRepository interface {
	WithTx(tx *gorm.DB) CheckoutRepository
	GetCartItemsWithBooks(ctx context.Context, userID uint, cartItemIDs []uint) ([]entity.CartItem, error)
	LockAndValidateBooks(ctx context.Context, bookIDs []uint) (map[uint]*entity.Book, error)
	DeductBookStock(ctx context.Context, bookID uint, qty int) error
	RestoreBookStock(ctx context.Context, bookID uint, qty int) error
	DeleteCartItems(ctx context.Context, cartItemIDs []uint) error
}
