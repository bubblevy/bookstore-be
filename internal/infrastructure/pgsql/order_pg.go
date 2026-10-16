package pgsql

import (
	"context"
	"errors"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) repository.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) WithTx(tx *gorm.DB) repository.OrderRepository {
	return &orderRepository{db: tx}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *entity.Order, items []entity.OrderItem) error {
	db := extractDB(ctx, r.db)

	if err := db.Create(order).Error; err != nil {
		return err
	}

	for i := range items {
		items[i].OrderID = order.ID
	}

	return db.Create(&items).Error
}

func (r *orderRepository) CreatePayment(ctx context.Context, payment *entity.Payment) error {
	db := extractDB(ctx, r.db)
	return db.Create(payment).Error
}

func (r *orderRepository) FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error) {
	db := extractDB(ctx, r.db)
	var order entity.Order

	err := db.Preload("Items").
		Preload("Payment").
		Where("invoice_number = ?", invoice).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindByID(ctx context.Context, id uint) (*entity.Order, error) {
	db := extractDB(ctx, r.db)
	var order entity.Order

	err := db.Preload("Items").
		Preload("Payment").
		First(&order, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error {
	db := extractDB(ctx, r.db)
	return db.Model(&entity.Order{}).Where("id = ?", orderID).Update("status", status).Error
}

func (r *orderRepository) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	db := extractDB(ctx, r.db)
	return db.Model(&entity.Payment{}).Where("order_id = ?", orderID).Update("status", status).Error
}

func (r *orderRepository) MarkAsReturned(ctx context.Context, orderID uint, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":        entity.OrderStatusReturned,
			"returned_at":   now,
			"return_reason": reason,
			"updated_at":    now,
		}).Error
}
