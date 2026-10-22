package repository

import (
	"context"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/pkg/pagination"
	"gorm.io/gorm"
)

type OrderRepository interface {
	WithTx(tx *gorm.DB) OrderRepository

	// write operation
	CreateOrder(ctx context.Context, order *entity.Order, items []entity.OrderItem) error
	CreatePayment(ctx context.Context, payment *entity.Payment) error
	UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error
	UpdatePaymentDetails(ctx context.Context, orderID uint, status entity.PaymentStatus, rawResponse *string, paidAt *time.Time) error
	UpdateOrderPaymentDeadline(ctx context.Context, orderID uint, deadline time.Time) error
	MarkAsReturned(ctx context.Context, orderID uint, reason string) error

	// read operation
	FindByID(ctx context.Context, id uint) (*entity.Order, error)
	FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error)
	FindUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error)
	FindOrderDetailByInvoice(ctx context.Context, userID uint, invoice string) (*entity.Order, error)
	FindAllOrders(ctx context.Context, filter AdminOrderFilter, pq pagination.Query) ([]entity.Order, int64, error)
	GetOrdersForReport(ctx context.Context, startDate, endDate *time.Time, status string) ([]entity.Order, error)
}

type AdminOrderFilter struct {
	Status    string
	Search    string
	StartDate *time.Time
	EndDate   *time.Time
	SortBy    string
}
