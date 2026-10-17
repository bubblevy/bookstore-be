package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"gorm.io/gorm"
)

type OrderRepository interface {
	WithTx(tx *gorm.DB) OrderRepository

	// write operation
	CreateOrder(ctx context.Context, order *entity.Order, items []entity.OrderItem) error
	CreatePayment(ctx context.Context, payment *entity.Payment) error
	UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, orderID uint, status entity.PaymentStatus) error
	MarkAsReturned(ctx context.Context, orderID uint, reason string) error

	// read operation
	FindByID(ctx context.Context, id uint) (*entity.Order, error)
	FindByInvoice(ctx context.Context, invoice string) (*entity.Order, error)
	FindUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error)
	FindOrderDetailByInvoice(ctx context.Context, userID uint, invoice string) (*entity.Order, error)
}
