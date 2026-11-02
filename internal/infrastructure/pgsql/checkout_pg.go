package pgsql

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type checkoutRepository struct {
	db *gorm.DB
}

func NewCheckoutRepository(db *gorm.DB) repository.CheckoutRepository {
	return &checkoutRepository{db: db}
}

func (r *checkoutRepository) WithTx(tx *gorm.DB) repository.CheckoutRepository {
	return &checkoutRepository{db: tx}
}

func (r *checkoutRepository) GetCartItemsWithBooks(ctx context.Context, userID uint, cartItemIDs []uint) ([]entity.CartItem, error) {
	var cart entity.Cart
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&cart).Error; err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).Preload("Book").Where("cart_id = ?", cart.ID)
	if len(cartItemIDs) > 0 {
		query = query.Where("id IN ?", cartItemIDs)
	}

	var items []entity.CartItem
	err := query.Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *checkoutRepository) LockAndValidateBooks(ctx context.Context, bookIDs []uint) (map[uint]*entity.Book, error) {
	var books []entity.Book
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", bookIDs).
		Find(&books).Error
	if err != nil {
		return nil, err
	}

	bookMap := make(map[uint]*entity.Book, len(books))
	for i := range books {
		bookMap[books[i].ID] = &books[i]
	}
	return bookMap, nil
}

func (r *checkoutRepository) DeductBookStock(ctx context.Context, bookID uint, qty int) error {
	return r.db.WithContext(ctx).Model(&entity.Book{}).
		Where("id = ? AND stock >= ?", bookID, qty).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock - ?", qty),
			"sold":  gorm.Expr("sold + ?", qty),
		}).Error
}

func (r *checkoutRepository) RestoreBookStock(ctx context.Context, bookID uint, qty int) error {
	return r.db.WithContext(ctx).Model(&entity.Book{}).
		Where("id = ?", bookID).
		Updates(map[string]interface{}{
			"stock": gorm.Expr("stock + ?", qty),
			"sold":  gorm.Expr("GREATEST(0, sold - ?)", qty),
		}).Error
}

func (r *checkoutRepository) DeleteCartItems(ctx context.Context, cartItemIDs []uint) error {
	return r.db.WithContext(ctx).
		Where("id IN ?", cartItemIDs).
		Delete(&entity.CartItem{}).Error
}
