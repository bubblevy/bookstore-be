package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type CartRepository interface {
	GetOrCreateByUserID(ctx context.Context, userID uint) (*entity.Cart, error)
	FindCartWithItems(ctx context.Context, userID uint) (*entity.Cart, error)
	FindItemByID(ctx context.Context, itemID uint) (*entity.CartItem, error)
	FindItemByCartAndBook(ctx context.Context, cartID, bookID uint) (*entity.CartItem, error)
	AddItem(ctx context.Context, item *entity.CartItem) error
	UpdateItemQuantity(ctx context.Context, itemID uint, quantity int) error
	DeleteItem(ctx context.Context, itemID uint) error
	ClearCart(ctx context.Context, cartID uint) error
	FindSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) ([]entity.CartItem, error)
	DeleteSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) error
}
