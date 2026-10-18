package repository

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
)

type BookFilter struct {
	Search        string
	CategorySlug  string
	AuthorSlug    string
	PublisherSlug string
	MinPrice      *int
	MaxPrice      *int
	IsActive      *bool
	SortBy        string // "newest", "price_asc", "price_desc", "title_asc"
	Limit         int
	Offset        int
}

type BookRepository interface {
	Create(ctx context.Context, book *entity.Book) error
	FindByID(ctx context.Context, id uint) (*entity.Book, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Book, error)
	FindAll(ctx context.Context, filter BookFilter) ([]entity.Book, int64, error)
	Update(ctx context.Context, book *entity.Book) error
	Delete(ctx context.Context, id uint) error
	ExistsBySlug(ctx context.Context, slug string, excludeID ...uint) (bool, error)
	ExistsByISBN(ctx context.Context, isbn string, excludeID ...uint) (bool, error)
	UpdateStock(ctx context.Context, id uint, quantity int) error
	CountExistingIDs(ctx context.Context, ids []uint) (int64, error)
	FindByIDs(ctx context.Context, ids []uint) ([]entity.Book, error)
}
