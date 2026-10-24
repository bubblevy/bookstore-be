package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type PaymentMethodRepository interface {
	FindAll(ctx context.Context, onlyActive bool) ([]entity.PaymentMethod, error)
	FindByID(ctx context.Context, id uint) (*entity.PaymentMethod, error)
	FindByCode(ctx context.Context, code string) (*entity.PaymentMethod, error)
	Update(ctx context.Context, pm *entity.PaymentMethod) error
}
