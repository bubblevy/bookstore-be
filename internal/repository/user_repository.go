package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type UserFilter struct {
	Search   string
	Role     string
	IsActive *bool
	Limit    int
	Offset   int
}

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	FindByActivationToken(ctx context.Context, token string) (*entity.User, error)
	FindByResetPasswordToken(ctx context.Context, token string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	FindAll(ctx context.Context, filter UserFilter) ([]entity.User, int64, error)
}
