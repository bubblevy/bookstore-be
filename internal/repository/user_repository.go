package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
}
