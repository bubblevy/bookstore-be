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

func (r *addressRepository) WithTx(tx *gorm.DB) repository.AddressRepository {
	return &addressRepository{db: tx}
}

func (r *addressRepository) Create(ctx context.Context, address *entity.Address) error {
	return r.db.WithContext(ctx).Create(address).Error
}

func (r *addressRepository) FindByID(ctx context.Context, id uint) (*entity.Address, error) {
	var address entity.Address
	err := r.db.WithContext(ctx).First(&address, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) FindByUserID(ctx context.Context, userID uint) ([]entity.Address, error) {
	var addresses []entity.Address
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_primary DESC, created_at DESC").
		Find(&addresses).Error
	return addresses, err
}

func (r *addressRepository) FindPrimaryByUserID(ctx context.Context, userID uint) (*entity.Address, error) {
	var address entity.Address
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_primary = true", userID).First(&address).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) Update(ctx context.Context, address *entity.Address) error {
	return r.db.WithContext(ctx).Save(address).Error
}

func (r *addressRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Address{}, id).Error
}

func (r *addressRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Address{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *addressRepository) ResetPrimaryStatus(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&entity.Address{}).
		Where("user_id = ? AND is_primary = true", userID).
		Update("is_primary", false).Error
}

func (r *addressRepository) SetPrimary(ctx context.Context, userID, addressID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Address{}).
			Where("user_id = ?", userID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		res := tx.Model(&entity.Address{}).
			Where("id = ? AND user_id = ?", addressID, userID).
			Update("is_primary", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return entity.ErrAddressNotFound
		}
		return nil
	})
}
