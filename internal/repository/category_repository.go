package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *entity.Category) error
	FindByID(ctx context.Context, id uint) (*entity.Category, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Category, error)
	FindAll(ctx context.Context) ([]entity.Category, error)
	Update(ctx context.Context, category *entity.Category) error
	Delete(ctx context.Context, id uint) error
	ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error)
}
