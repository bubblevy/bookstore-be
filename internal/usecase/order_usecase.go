package usecase

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
)

type OrderUsecase interface {
	GetUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error)
	GetOrderDetail(ctx context.Context, userID uint, invoiceNumber string) (*entity.Order, error)
}

type orderUsecase struct {
	orderRepo repository.OrderRepository
}

func NewOrderUsecase(orderRepo repository.OrderRepository) OrderUsecase {
	return &orderUsecase{orderRepo: orderRepo}
}

func (u *orderUsecase) GetUserOrders(ctx context.Context, userID uint, statusFilter string) ([]entity.Order, error) {
	return u.orderRepo.FindUserOrders(ctx, userID, statusFilter)
}

func (u *orderUsecase) GetOrderDetail(ctx context.Context, userID uint, invoiceNumber string) (*entity.Order, error) {
	order, err := u.orderRepo.FindOrderDetailByInvoice(ctx, userID, invoiceNumber)
	if err != nil {
		return nil, entity.ErrOrderNotFound
	}
	return order, nil
}
