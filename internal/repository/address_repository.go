package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"gorm.io/gorm"
)

type AddressRepository interface {
	WithTx(tx *gorm.DB) AddressRepository
	Create(ctx context.Context, address *entity.Address) error
	FindByID(ctx context.Context, id uint) (*entity.Address, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.Address, error)
	FindPrimaryByUserID(ctx context.Context, userID uint) (*entity.Address, error)
	Update(ctx context.Context, address *entity.Address) error
	Delete(ctx context.Context, id uint) error
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	ResetPrimaryStatus(ctx context.Context, userID uint) error
	SetPrimary(ctx context.Context, userID, addressID uint) error
}
