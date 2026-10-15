package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) repository.CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) GetOrCreateByUserID(ctx context.Context, userID uint) (*entity.Cart, error) {
	db := extractDB(ctx, r.db)
	var cart entity.Cart

	err := db.Where("user_id = ?", userID).
		Attrs(entity.Cart{UserID: userID}).
		FirstOrCreate(&cart).Error

	if err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) FindCartWithItems(ctx context.Context, userID uint) (*entity.Cart, error) {
	db := extractDB(ctx, r.db)
	var cart entity.Cart

	err := db.Preload("Items.Book").
		Where("user_id = ?", userID).
		First(&cart).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.GetOrCreateByUserID(ctx, userID)
		}
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) FindItemByID(ctx context.Context, itemID uint) (*entity.CartItem, error) {
	db := extractDB(ctx, r.db)
	var item entity.CartItem

	err := db.Preload("Book").First(&item, itemID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *cartRepository) FindItemByCartAndBook(ctx context.Context, cartID, bookID uint) (*entity.CartItem, error) {
	db := extractDB(ctx, r.db)
	var item entity.CartItem

	err := db.Where("cart_id = ? AND book_id = ?", cartID, bookID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *cartRepository) AddItem(ctx context.Context, item *entity.CartItem) error {
	db := extractDB(ctx, r.db)

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cart_id"}, {Name: "book_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("cart_items.quantity + EXCLUDED.quantity"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(item).Error
}

func (r *cartRepository) UpdateItemQuantity(ctx context.Context, itemID uint, quantity int) error {
	db := extractDB(ctx, r.db)
	return db.Model(&entity.CartItem{}).Where("id = ?", itemID).Update("quantity", quantity).Error
}

func (r *cartRepository) DeleteItem(ctx context.Context, itemID uint) error {
	db := extractDB(ctx, r.db)
	return db.Delete(&entity.CartItem{}, itemID).Error
}

func (r *cartRepository) ClearCart(ctx context.Context, cartID uint) error {
	db := extractDB(ctx, r.db)
	return db.Where("cart_id = ?", cartID).Delete(&entity.CartItem{}).Error
}

func (r *cartRepository) FindSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) ([]entity.CartItem, error) {
	if len(itemIDs) == 0 {
		return []entity.CartItem{}, nil
	}
	db := extractDB(ctx, r.db)
	var items []entity.CartItem

	err := db.Preload("Book").
		Where("cart_id = ? AND id IN ?", cartID, itemIDs).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *cartRepository) DeleteSelectedItems(ctx context.Context, cartID uint, itemIDs []uint) error {
	if len(itemIDs) == 0 {
		return nil
	}
	db := extractDB(ctx, r.db)
	return db.Where("cart_id = ? AND id IN ?", cartID, itemIDs).Delete(&entity.CartItem{}).Error
}
