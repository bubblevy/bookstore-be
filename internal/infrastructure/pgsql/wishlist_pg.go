package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type wishlistRepository struct {
	db *gorm.DB
}

func NewWishlistRepository(db *gorm.DB) repository.WishlistRepository {
	return &wishlistRepository{db: db}
}

func (r *wishlistRepository) FindByUserID(ctx context.Context, userID uint, page, perPage int) ([]entity.Wishlist, int64, error) {
	db := extractDB(ctx, r.db)

	var total int64
	if err := db.Model(&entity.Wishlist{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if perPage <= 0 {
		perPage = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	var items []entity.Wishlist
	err := db.Preload("Book.Category").
		Preload("Book.Author").
		Preload("Book.Publisher").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&items).Error

	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *wishlistRepository) FindByUserAndBook(ctx context.Context, userID, bookID uint) (*entity.Wishlist, error) {
	db := extractDB(ctx, r.db)
	var item entity.Wishlist

	err := db.Preload("Book.Category").
		Preload("Book.Author").
		Preload("Book.Publisher").
		Where("user_id = ? AND book_id = ?", userID, bookID).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *wishlistRepository) FindByID(ctx context.Context, id uint) (*entity.Wishlist, error) {
	db := extractDB(ctx, r.db)
	var item entity.Wishlist

	err := db.Preload("Book").First(&item, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *wishlistRepository) Add(ctx context.Context, wishlist *entity.Wishlist) error {
	db := extractDB(ctx, r.db)
	return db.Create(wishlist).Error
}

func (r *wishlistRepository) DeleteByUserAndBook(ctx context.Context, userID, bookID uint) error {
	db := extractDB(ctx, r.db)
	result := db.Where("user_id = ? AND book_id = ?", userID, bookID).Delete(&entity.Wishlist{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrWishlistItemNotFound
	}
	return nil
}

func (r *wishlistRepository) DeleteByID(ctx context.Context, id uint) error {
	db := extractDB(ctx, r.db)
	result := db.Delete(&entity.Wishlist{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrWishlistItemNotFound
	}
	return nil
}

func (r *wishlistRepository) Clear(ctx context.Context, userID uint) error {
	db := extractDB(ctx, r.db)
	return db.Where("user_id = ?", userID).Delete(&entity.Wishlist{}).Error
}

func (r *wishlistRepository) Exists(ctx context.Context, userID, bookID uint) (bool, error) {
	db := extractDB(ctx, r.db)
	var count int64
	err := db.Model(&entity.Wishlist{}).
		Where("user_id = ? AND book_id = ?", userID, bookID).
		Count(&count).Error
	return count > 0, err
}
