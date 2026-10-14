package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type PublisherRepository interface {
	Create(ctx context.Context, publisher *entity.Publisher) error
	FindByID(ctx context.Context, id uint) (*entity.Publisher, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Publisher, error)
	FindAll(ctx context.Context) ([]entity.Publisher, error)
	Update(ctx context.Context, publisher *entity.Publisher) error
	Delete(ctx context.Context, id uint) error
	ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error)
}
