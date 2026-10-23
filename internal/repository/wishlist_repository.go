package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type WishlistRepository interface {
	FindByUserID(ctx context.Context, userID uint, page, perPage int) ([]entity.Wishlist, int64, error)
	FindByUserAndBook(ctx context.Context, userID, bookID uint) (*entity.Wishlist, error)
	FindByID(ctx context.Context, id uint) (*entity.Wishlist, error)
	Add(ctx context.Context, wishlist *entity.Wishlist) error
	DeleteByUserAndBook(ctx context.Context, userID, bookID uint) error
	DeleteByID(ctx context.Context, id uint) error
	Clear(ctx context.Context, userID uint) error
	Exists(ctx context.Context, userID, bookID uint) (bool, error)
}
