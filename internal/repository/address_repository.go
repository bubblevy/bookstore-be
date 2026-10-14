package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type AddressRepository interface {
	Create(ctx context.Context, address *entity.UserAddress) error
	FindByID(ctx context.Context, id uint) (*entity.UserAddress, error)
	FindByIDAndUserID(ctx context.Context, id uint, userID uint) (*entity.UserAddress, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.UserAddress, error)
	Update(ctx context.Context, address *entity.UserAddress) error
	Delete(ctx context.Context, id uint, userID uint) error
	ResetPrimaryStatus(ctx context.Context, userID uint) error
	CountByUserID(ctx context.Context, userID uint) (int64, error)
}
