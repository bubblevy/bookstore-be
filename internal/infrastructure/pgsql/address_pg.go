package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) repository.AddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(ctx context.Context, address *entity.UserAddress) error {
	db := extractDB(ctx, r.db)
	return db.Create(address).Error
}

func (r *addressRepository) FindByID(ctx context.Context, id uint) (*entity.UserAddress, error) {
	db := extractDB(ctx, r.db)
	var address entity.UserAddress
	err := db.First(&address, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) FindByIDAndUserID(ctx context.Context, id uint, userID uint) (*entity.UserAddress, error) {
	db := extractDB(ctx, r.db)
	var address entity.UserAddress
	err := db.Where("id = ? AND user_id = ?", id, userID).First(&address).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) FindByUserID(ctx context.Context, userID uint) ([]entity.UserAddress, error) {
	db := extractDB(ctx, r.db)
	var addresses []entity.UserAddress
	err := db.Where("user_id = ?", userID).
		Order("is_primary DESC, created_at DESC").
		Find(&addresses).Error
	return addresses, err
}

func (r *addressRepository) Update(ctx context.Context, address *entity.UserAddress) error {
	db := extractDB(ctx, r.db)
	return db.Save(address).Error
}

func (r *addressRepository) Delete(ctx context.Context, id uint, userID uint) error {
	db := extractDB(ctx, r.db)
	result := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&entity.UserAddress{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *addressRepository) ResetPrimaryStatus(ctx context.Context, userID uint) error {
	db := extractDB(ctx, r.db)
	return db.Model(&entity.UserAddress{}).
		Where("user_id = ? AND is_primary = ?", userID, true).
		Update("is_primary", false).Error
}

func (r *addressRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	db := extractDB(ctx, r.db)
	var count int64
	err := db.Model(&entity.UserAddress{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
