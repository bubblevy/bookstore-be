package pgsql

import (
	"context"
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
	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		return err
	}

	for i := range items {
		items[i].OrderID = order.ID
	}

	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *orderRepository) CreatePayment(ctx context.Context, payment *entity.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).
		Where("id = ?", orderID).
		Update("status", status).Error
}

func (r *orderRepository) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	return r.db.WithContext(ctx).Model(&entity.Payment{}).
		Where("order_id = ?", orderID).
		Update("status", status).Error
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

func (r *orderRepository) FindByID(ctx context.Context, id uint) (*entity.Order, error) {
	var order entity.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ?", id).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error) {
	var order entity.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("invoice_number = ?", invoice).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	var orders []entity.Order
	query := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID)

	if statusFilter != "" && statusFilter != "all" {
		query = query.Where("status = ?", statusFilter)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *orderRepository) FindOrderDetailByInvoice(ctx context.Context, userID uint, invoice string) (*entity.Order, error) {
	var order entity.Order
	query := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Payment").
		Preload("Shipment").
		Where("invoice_number = ?", invoice)

	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}
