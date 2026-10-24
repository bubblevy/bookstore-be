package pgsql

import (
	"context"
	"errors"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type paymentMethodRepository struct {
	db *gorm.DB
}

func NewPaymentMethodRepository(db *gorm.DB) repository.PaymentMethodRepository {
	return &paymentMethodRepository{db: db}
}

func (r *paymentMethodRepository) FindAll(ctx context.Context, onlyActive bool) ([]entity.PaymentMethod, error) {
	db := extractDB(ctx, r.db)
	var methods []entity.PaymentMethod
	query := db.Model(&entity.PaymentMethod{})
	if onlyActive {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("id ASC").Find(&methods).Error
	return methods, err
}

func (r *paymentMethodRepository) FindByID(ctx context.Context, id uint) (*entity.PaymentMethod, error) {
	db := extractDB(ctx, r.db)
	var method entity.PaymentMethod
	err := db.Where("id = ?", id).First(&method).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &method, nil
}

func (r *paymentMethodRepository) FindByCode(ctx context.Context, code string) (*entity.PaymentMethod, error) {
	db := extractDB(ctx, r.db)
	var method entity.PaymentMethod
	err := db.Where("code = ?", code).First(&method).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &method, nil
}

func (r *paymentMethodRepository) Update(ctx context.Context, pm *entity.PaymentMethod) error {
	db := extractDB(ctx, r.db)
	// Code is strictly immutable
	return db.Model(&entity.PaymentMethod{}).
		Where("id = ?", pm.ID).
		Updates(map[string]interface{}{
			"name":         pm.Name,
			"category":     pm.Category,
			"fee_type":     pm.FeeType,
			"fee_amount":   pm.FeeAmount,
			"fee_percent":  pm.FeePercent,
			"image":        pm.Image,
			"description":  pm.Description,
			"instructions": pm.Instructions,
			"is_active":    pm.IsActive,
		}).Error
}
