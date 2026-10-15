package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type AuthorRepository interface {
	Create(ctx context.Context, author *entity.Author) error
	FindByID(ctx context.Context, id uint) (*entity.Author, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Author, error)
	FindAll(ctx context.Context, limit, offset int) ([]entity.Author, int64, error)
	Update(ctx context.Context, author *entity.Author) error
	Delete(ctx context.Context, id uint) error
	ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error)
}
