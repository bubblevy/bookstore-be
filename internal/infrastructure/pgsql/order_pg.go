package pgsql

import (
	"context"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
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
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == entity.OrderStatusPaid {
		updates["paid_at"] = time.Now()
	} else if status == entity.OrderStatusCancelled {
		updates["cancelled_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&entity.Order{}).
		Where("id = ?", orderID).
		Updates(updates).Error
}

func (r *orderRepository) UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == entity.PaymentStatusSettlement {
		updates["paid_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&entity.Payment{}).
		Where("order_id = ?", orderID).
		Updates(updates).Error
}

func (r *orderRepository) UpdatePaymentDetails(ctx context.Context, orderID uint, status entity.PaymentStatus, rawResponse *string, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if rawResponse != nil && *rawResponse != "" {
		updates["raw_response"] = *rawResponse
	}
	if paidAt != nil {
		updates["paid_at"] = *paidAt
	} else if status == entity.PaymentStatusSettlement {
		updates["paid_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&entity.Payment{}).
		Where("order_id = ?", orderID).
		Updates(updates).Error
}

func (r *orderRepository) UpdateOrderPaymentDeadline(ctx context.Context, orderID uint, deadline time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.Order{}).
		Where("id = ?", orderID).
		Update("payment_deadline", deadline).Error
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
		Preload("User").
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

func (r *orderRepository) FindAllOrders(ctx context.Context, filter repository.AdminOrderFilter, pq pagination.Query) ([]entity.Order, int64, error) {
	var orders []entity.Order
	var total int64

	db := r.db.WithContext(ctx).Model(&entity.Order{})

	if filter.Status != "" && filter.Status != "all" {
		db = db.Where("status = ?", filter.Status)
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		db = db.Where("invoice_number ILIKE ? OR recipient_name ILIKE ? OR recipient_phone ILIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if filter.StartDate != nil {
		db = db.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		db = db.Where("created_at <= ?", *filter.EndDate)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply Sorting
	switch strings.ToLower(filter.SortBy) {
	case "oldest":
		db = db.Order("created_at ASC")
	case "highest_amount":
		db = db.Order("total_amount DESC")
	case "lowest_amount":
		db = db.Order("total_amount ASC")
	default:
		db = db.Order("created_at DESC")
	}

	err := db.Preload("Items").
		Preload("User").
		Preload("Payment").
		Preload("Shipment").
		Offset(pq.Offset()).
		Limit(pq.PerPage).
		Find(&orders).Error

	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *orderRepository) GetOrdersForReport(ctx context.Context, startDate, endDate *time.Time, status string) ([]entity.Order, error) {
	var orders []entity.Order

	db := r.db.WithContext(ctx).Model(&entity.Order{})

	if startDate != nil {
		db = db.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		db = db.Where("created_at <= ?", *endDate)
	}

	if status != "" && status != "all" {
		db = db.Where("status = ?", status)
	}

	err := db.Preload("Items").
		Preload("User").
		Preload("Payment").
		Preload("Shipment").
		Order("created_at ASC").
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	return orders, nil
}
